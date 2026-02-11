package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Store errors
var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrGuardFailed       = errors.New("transition guard failed")
	ErrValidation        = errors.New("validation error")
)

// Store provides generic CRUD operations for all resources defined in the schema.
type Store struct {
	db       *sqlx.DB
	schema   map[string]*Resource
	ordered  []Resource // ordered list for migrations
}

// NewStore creates a new generic store, runs migrations, and prepares for queries.
func NewStore(db *sqlx.DB, resources []Resource) (*Store, error) {
	schema := make(map[string]*Resource, len(resources))
	ordered := make([]Resource, len(resources))
	for i := range resources {
		r := resources[i]
		schema[r.Name] = &r
		ordered[i] = r
	}
	s := &Store{
		db:      db,
		schema:  schema,
		ordered: ordered,
	}
	return s, nil
}

// DB returns the underlying sqlx.DB for use by legacy code during migration.
func (s *Store) DB() *sqlx.DB {
	return s.db
}

// Resource returns the resource definition by name.
func (s *Store) Resource(name string) *Resource {
	return s.schema[name]
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// =============================================================================
// Pagination
// =============================================================================

type Page struct {
	Limit  int
	Offset int
}

func DefaultPage() Page {
	return Page{Limit: 100, Offset: 0}
}

func (p Page) Normalize() Page {
	if p.Limit <= 0 {
		p.Limit = 100
	}
	if p.Limit > 1000 {
		p.Limit = 1000
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	return p
}

// =============================================================================
// Filters
// =============================================================================

type Filter struct {
	Field string
	Value any
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create inserts a new row for the given resource.
// Validates fields, generates reference_id, applies computed fields and defaults.
func (s *Store) Create(ctx context.Context, resource string, data map[string]any) (map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	// Generate reference_id
	refID := res.RefPrefix + uuid.New().String()[:8]
	if res.RefPrefix == "" {
		refID = uuid.New().String()
	}
	data["reference_id"] = refID

	// Apply defaults
	for _, f := range res.Fields {
		if _, exists := data[f.Name]; !exists && f.DefaultValue != nil {
			data[f.Name] = f.DefaultValue
		}
	}

	// Apply computed fields
	for _, f := range res.Fields {
		if f.Computed != nil {
			data[f.Name] = f.Computed(data)
		}
	}

	// Apply state machine initial state
	if res.StateMachine != nil {
		if _, exists := data[res.StateMachine.Field]; !exists {
			data[res.StateMachine.Field] = res.StateMachine.Initial
		}
	}

	// Validate
	if err := s.validate(res, data); err != nil {
		return nil, err
	}

	// Set timestamps
	now := time.Now().UTC().Format(time.RFC3339)
	data["created_at"] = now
	data["updated_at"] = now

	// Build INSERT
	cols := []string{"reference_id"}
	placeholders := []string{":reference_id"}
	for _, f := range res.Fields {
		if _, exists := data[f.Name]; exists {
			cols = append(cols, f.Name)
			placeholders = append(placeholders, ":"+f.Name)
		}
	}
	cols = append(cols, "created_at", "updated_at")
	placeholders = append(placeholders, ":created_at", ":updated_at")

	// JSON-encode JSON fields
	for _, f := range res.Fields {
		if f.Type == TypeJSON {
			if v, ok := data[f.Name]; ok && v != nil {
				switch val := v.(type) {
				case string:
					// already string, keep as-is
				case []byte:
					data[f.Name] = string(val)
				default:
					b, err := json.Marshal(val)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal %s: %w", f.Name, err)
					}
					data[f.Name] = string(b)
				}
			}
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		resource, strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	result, err := s.db.NamedExecContext(ctx, query, data)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", resource, err)
	}

	id, _ := result.LastInsertId()
	data["id"] = id

	return data, nil
}

// Get retrieves a single row by reference_id.
func (s *Store) Get(ctx context.Context, resource string, refID string) (map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	cols := s.selectColumns(res)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE reference_id = ?", cols, resource)

	row := s.db.QueryRowxContext(ctx, query, refID)
	result := make(map[string]any)
	if err := row.MapScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s %s: %w", resource, refID, ErrNotFound)
		}
		return nil, fmt.Errorf("get %s: %w", resource, err)
	}

	s.decodeRow(res, result)
	return result, nil
}

// GetByID retrieves a single row by integer primary key.
func (s *Store) GetByID(ctx context.Context, resource string, id int) (map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	cols := s.selectColumns(res)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", cols, resource)

	row := s.db.QueryRowxContext(ctx, query, id)
	result := make(map[string]any)
	if err := row.MapScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s id=%d: %w", resource, id, ErrNotFound)
		}
		return nil, fmt.Errorf("get %s by id: %w", resource, err)
	}

	s.decodeRow(res, result)
	return result, nil
}

// GetRefIDByIntID returns the reference_id for a given internal integer ID.
func (s *Store) GetRefIDByIntID(resource string, id int) (string, error) {
	var refID string
	err := s.db.QueryRow(fmt.Sprintf("SELECT reference_id FROM %s WHERE id = ?", resource), id).Scan(&refID)
	return refID, err
}

// List retrieves rows with optional filters and pagination.
func (s *Store) List(ctx context.Context, resource string, filters []Filter, page Page) ([]map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	page = page.Normalize()
	cols := s.selectColumns(res)

	var where []string
	var args []any
	for _, f := range filters {
		where = append(where, fmt.Sprintf("%s = ?", f.Field))
		args = append(args, f.Value)
	}

	query := fmt.Sprintf("SELECT %s FROM %s", cols, resource)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY id DESC"
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", page.Limit, page.Offset)

	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", resource, err)
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return nil, fmt.Errorf("scan %s row: %w", resource, err)
		}
		s.decodeRow(res, row)
		results = append(results, row)
	}

	return results, rows.Err()
}

// Update updates a row by reference_id with the given data.
// Only fields present in data are updated.
func (s *Store) Update(ctx context.Context, resource string, refID string, data map[string]any) (map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	// Don't allow updating reference_id, id, created_at
	delete(data, "reference_id")
	delete(data, "id")
	delete(data, "created_at")

	// Set updated_at
	data["updated_at"] = time.Now().UTC().Format(time.RFC3339)

	// JSON-encode JSON fields
	for _, f := range res.Fields {
		if f.Type == TypeJSON {
			if v, ok := data[f.Name]; ok && v != nil {
				switch val := v.(type) {
				case string:
					// keep as-is
				case []byte:
					data[f.Name] = string(val)
				default:
					b, err := json.Marshal(val)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal %s: %w", f.Name, err)
					}
					data[f.Name] = string(b)
				}
			}
		}
	}

	// Build SET clause
	var setClauses []string
	var args []any
	for key, val := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
		args = append(args, val)
	}

	if len(setClauses) == 0 {
		return s.Get(ctx, resource, refID)
	}

	args = append(args, refID)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE reference_id = ?",
		resource, strings.Join(setClauses, ", "))

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", resource, err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, fmt.Errorf("%s %s: %w", resource, refID, ErrNotFound)
	}

	return s.Get(ctx, resource, refID)
}

// Delete removes a row by reference_id.
func (s *Store) Delete(ctx context.Context, resource string, refID string) error {
	if _, ok := s.schema[resource]; !ok {
		return fmt.Errorf("unknown resource: %s", resource)
	}

	result, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE reference_id = ?", resource), refID)
	if err != nil {
		return fmt.Errorf("delete %s: %w", resource, err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("%s %s: %w", resource, refID, ErrNotFound)
	}

	return nil
}

// =============================================================================
// State Machine Transitions
// =============================================================================

// Transition atomically transitions a resource's state machine to a new state.
// Returns the updated row and the command name to dispatch (if any).
func (s *Store) Transition(ctx context.Context, resource string, refID string, toState string) (map[string]any, string, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, "", fmt.Errorf("unknown resource: %s", resource)
	}

	if res.StateMachine == nil {
		return nil, "", fmt.Errorf("resource %s has no state machine", resource)
	}

	sm := res.StateMachine

	// Get current row
	row, err := s.Get(ctx, resource, refID)
	if err != nil {
		return nil, "", err
	}

	fromState, _ := row[sm.Field].(string)
	// Handle []byte from SQLite
	if b, ok := row[sm.Field].([]byte); ok {
		fromState = string(b)
	}

	// Validate transition
	if !sm.CanTransition(fromState, toState) {
		return nil, "", fmt.Errorf("%w: %s → %s", ErrInvalidTransition, fromState, toState)
	}

	// Run guard
	if guard, ok := sm.Guards[toState]; ok {
		if err := guard(row); err != nil {
			return nil, "", fmt.Errorf("%w: %v", ErrGuardFailed, err)
		}
	}

	// Update the state
	updated, err := s.Update(ctx, resource, refID, map[string]any{
		sm.Field: toState,
	})
	if err != nil {
		return nil, "", err
	}

	// Return command to dispatch
	cmd := sm.OnEnter[toState]
	return updated, cmd, nil
}

// =============================================================================
// User Resolution
// =============================================================================

// ResolveUser upserts a user and returns their integer ID.
func (s *Store) ResolveUser(ctx context.Context, referenceID, email, name, planID string) (int, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (reference_id, email, name, plan_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(reference_id) DO UPDATE SET
			email = CASE WHEN excluded.email != '' THEN excluded.email ELSE users.email END,
			name = CASE WHEN excluded.name != '' THEN excluded.name ELSE users.name END,
			plan_id = CASE WHEN excluded.plan_id != '' THEN excluded.plan_id ELSE users.plan_id END,
			updated_at = datetime('now')
	`, referenceID, email, name, planID)
	if err != nil {
		return 0, fmt.Errorf("resolve user: %w", err)
	}

	var userID int
	err = s.db.GetContext(ctx, &userID, "SELECT id FROM users WHERE reference_id = ?", referenceID)
	if err != nil {
		return 0, fmt.Errorf("resolve user: %w", err)
	}
	return userID, nil
}

// =============================================================================
// Special queries (needed by workers/proxy/scheduler that the generic CRUD doesn't cover)
// =============================================================================

// GetByField retrieves a row by an arbitrary field value.
func (s *Store) GetByField(ctx context.Context, resource, field string, value any) (map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	cols := s.selectColumns(res)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", cols, resource, field)

	row := s.db.QueryRowxContext(ctx, query, value)
	result := make(map[string]any)
	if err := row.MapScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s %s=%v: %w", resource, field, value, ErrNotFound)
		}
		return nil, fmt.Errorf("get %s by %s: %w", resource, field, err)
	}

	s.decodeRow(res, result)
	return result, nil
}

// GetByTwoFields retrieves a row by two field values (AND).
func (s *Store) GetByTwoFields(ctx context.Context, resource, field1 string, value1 any, field2 string, value2 any) (map[string]any, error) {
	res, ok := s.schema[resource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resource)
	}

	cols := s.selectColumns(res)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? AND %s = ?", cols, resource, field1, field2)

	row := s.db.QueryRowxContext(ctx, query, value1, value2)
	result := make(map[string]any)
	if err := row.MapScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", resource, ErrNotFound)
		}
		return nil, fmt.Errorf("get %s: %w", resource, err)
	}

	s.decodeRow(res, result)
	return result, nil
}

// RawQuery executes a raw SQL query and returns rows as maps.
func (s *Store) RawQuery(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := s.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// RawExec executes a raw SQL statement.
func (s *Store) RawExec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// WithTx executes fn within a database transaction.
func (s *Store) WithTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// =============================================================================
// docker.NodeStore implementation — satisfies infrastructure layer interfaces
// =============================================================================

// GetNode returns a domain.Node for use by the docker NodePool.
func (s *Store) GetNode(ctx context.Context, nodeID string) (*domain.Node, error) {
	row, err := s.Get(ctx, "nodes", nodeID)
	if err != nil {
		return nil, err
	}
	node := mapToNode(row)
	// Resolve SSH key reference_id from integer FK
	if node.SSHKeyID > 0 {
		sshKeyRow, err := s.GetByID(ctx, "ssh_keys", node.SSHKeyID)
		if err == nil {
			node.SSHKeyRefID = strVal(sshKeyRow["reference_id"])
		}
	}
	return node, nil
}

// GetSSHKey returns a domain.SSHKey for use by the docker NodePool.
func (s *Store) GetSSHKey(ctx context.Context, sshKeyRefID string) (*domain.SSHKey, error) {
	row, err := s.Get(ctx, "ssh_keys", sshKeyRefID)
	if err != nil {
		return nil, err
	}
	return mapToSSHKey(row), nil
}

func mapToNode(row map[string]any) *domain.Node {
	intID, _ := toInt64(row["id"])
	sshKeyID, _ := toInt64(row["ssh_key_id"])
	sshPort, _ := toInt64(row["ssh_port"])
	if sshPort == 0 {
		sshPort = 22
	}
	n := &domain.Node{
		ID:          int(intID),
		ReferenceID: strVal(row["reference_id"]),
		Name:        strVal(row["name"]),
		SSHHost:     strVal(row["ssh_host"]),
		SSHPort:     int(sshPort),
		SSHUser:     strVal(row["ssh_user"]),
		SSHKeyID:    int(sshKeyID),
		Status:      domain.NodeStatus(strVal(row["status"])),
	}
	return n
}

func mapToSSHKey(row map[string]any) *domain.SSHKey {
	intID, _ := toInt64(row["id"])
	k := &domain.SSHKey{
		ID:          int(intID),
		ReferenceID: strVal(row["reference_id"]),
		Name:        strVal(row["name"]),
		Fingerprint: strVal(row["fingerprint"]),
	}
	// PrivateKeyEncrypted can be []byte or string
	switch v := row["private_key_encrypted"].(type) {
	case []byte:
		k.PrivateKeyEncrypted = v
	case string:
		k.PrivateKeyEncrypted = []byte(v)
	}
	return k
}

// =============================================================================
// proxy.ProxyStore implementation — satisfies proxy server interface
// =============================================================================

// GetDeploymentByDomain finds a deployment where any domain in the JSON array matches the hostname.
func (s *Store) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
	query := `
		SELECT id, reference_id, name, template_id, template_version, customer_id,
		       node_id, status, variables, domains, containers,
		       resources_cpu_cores, resources_memory_mb, resources_disk_mb,
		       proxy_port, error_message, started_at, stopped_at,
		       created_at, updated_at
		FROM deployments
		WHERE EXISTS (
			SELECT 1 FROM json_each(deployments.domains) AS je
			WHERE json_extract(je.value, '$.hostname') = ?
		)
		LIMIT 1
	`

	row := s.db.QueryRowxContext(ctx, query, hostname)
	result := make(map[string]any)
	if err := row.MapScan(result); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("deployment for hostname %s: %w", hostname, ErrNotFound)
		}
		return nil, fmt.Errorf("get deployment by domain: %w", err)
	}

	// Decode the row using the deployments resource schema
	if res := s.schema["deployments"]; res != nil {
		s.decodeRow(res, result)
	}

	return mapToDeployment(result), nil
}

// CountRoutableDeployments counts deployments that are running with a proxy port assigned.
func (s *Store) CountRoutableDeployments(ctx context.Context) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM deployments WHERE status = 'running' AND proxy_port IS NOT NULL")
	if err != nil {
		return 0, fmt.Errorf("count routable deployments: %w", err)
	}
	return count, nil
}

// mapToDeployment converts a store row to a domain.Deployment for infrastructure consumers.
func mapToDeployment(data map[string]any) *domain.Deployment {
	d := &domain.Deployment{
		ReferenceID: strVal(data["reference_id"]),
		Name:        strVal(data["name"]),
		NodeID:      strVal(data["node_id"]),
		Status:      domain.DeploymentStatus(strVal(data["status"])),
	}

	if id, ok := toInt64(data["id"]); ok {
		d.ID = int(id)
	}
	if id, ok := toInt64(data["template_id"]); ok {
		d.TemplateID = int(id)
	}
	if id, ok := toInt64(data["customer_id"]); ok {
		d.CustomerID = int(id)
	}
	if p, ok := toInt64(data["proxy_port"]); ok {
		d.ProxyPort = int(p)
	}

	// Parse domains JSON
	if dom, ok := data["domains"]; ok {
		switch val := dom.(type) {
		case string:
			json.Unmarshal([]byte(val), &d.Domains)
		case []any:
			b, _ := json.Marshal(val)
			json.Unmarshal(b, &d.Domains)
		}
	}

	// Parse containers JSON
	if c, ok := data["containers"]; ok {
		switch val := c.(type) {
		case string:
			json.Unmarshal([]byte(val), &d.Containers)
		case []any:
			b, _ := json.Marshal(val)
			json.Unmarshal(b, &d.Containers)
		}
	}

	// Parse variables JSON
	if v, ok := data["variables"]; ok {
		switch val := v.(type) {
		case string:
			json.Unmarshal([]byte(val), &d.Variables)
		case map[string]any:
			d.Variables = make(map[string]string)
			for k, v := range val {
				d.Variables[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return d
}

// =============================================================================
// billing.BillingStore implementation — satisfies billing reporter interface
// =============================================================================

// CreateUsageEvent inserts a usage event for later batch reporting.
func (s *Store) CreateUsageEvent(ctx context.Context, event *domain.MeterEvent) error {
	var metadataJSON *string
	if event.Metadata != nil {
		data, _ := json.Marshal(event.Metadata)
		str := string(data)
		metadataJSON = &str
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO usage_events (reference_id, user_id, event_type, resource_id, resource_type, quantity, metadata, timestamp, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ReferenceID, event.UserID, string(event.EventType),
		event.ResourceID, event.ResourceType, event.Quantity,
		metadataJSON, event.Timestamp.Format(time.RFC3339), event.CreatedAt.Format(time.RFC3339))
	return err
}

// GetUnreportedEvents retrieves usage events that haven't been reported to APIGate yet.
func (s *Store) GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryxContext(ctx,
		`SELECT id, reference_id, user_id, event_type, resource_id, resource_type, quantity, metadata, timestamp, reported_at, created_at
		 FROM usage_events WHERE reported_at IS NULL ORDER BY timestamp ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []domain.MeterEvent
	for rows.Next() {
		row := make(map[string]any)
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		ev := domain.MeterEvent{
			ReferenceID:  strVal(row["reference_id"]),
			EventType:    domain.EventType(strVal(row["event_type"])),
			ResourceID:   strVal(row["resource_id"]),
			ResourceType: strVal(row["resource_type"]),
		}
		if id, ok := toInt64(row["id"]); ok {
			ev.ID = int(id)
		}
		if uid, ok := toInt64(row["user_id"]); ok {
			ev.UserID = int(uid)
		}
		if q, ok := toInt64(row["quantity"]); ok {
			ev.Quantity = q
		}
		if ts := strVal(row["timestamp"]); ts != "" {
			ev.Timestamp, _ = time.Parse(time.RFC3339, ts)
		}
		if ca := strVal(row["created_at"]); ca != "" {
			ev.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		}
		if md := strVal(row["metadata"]); md != "" {
			json.Unmarshal([]byte(md), &ev.Metadata)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

// MarkEventsReported marks usage events as reported to APIGate.
func (s *Store) MarkEventsReported(ctx context.Context, ids []string, reportedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = reportedAt.Format(time.RFC3339)
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+1] = id
	}
	query := fmt.Sprintf("UPDATE usage_events SET reported_at = ? WHERE reference_id IN (%s)",
		strings.Join(placeholders, ","))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return ""
}

// =============================================================================
// Helpers
// =============================================================================

// selectColumns returns the SELECT column list for a resource.
func (s *Store) selectColumns(res *Resource) string {
	cols := []string{"id", "reference_id"}
	for _, f := range res.Fields {
		cols = append(cols, f.Name)
	}
	cols = append(cols, "created_at", "updated_at")
	return strings.Join(cols, ", ")
}

// decodeRow converts SQLite types to Go types (especially []byte → string, JSON strings → parsed).
func (s *Store) decodeRow(res *Resource, row map[string]any) {
	// Convert []byte to string for all text columns
	for key, val := range row {
		if b, ok := val.([]byte); ok {
			row[key] = string(b)
		}
	}

	// Coerce bool fields from SQLite integer (0/1) to Go bool
	for _, f := range res.Fields {
		if f.Type == TypeBool {
			if v, ok := row[f.Name]; ok {
				switch val := v.(type) {
				case int64:
					row[f.Name] = val != 0
				case int:
					row[f.Name] = val != 0
				case float64:
					row[f.Name] = val != 0
				}
			}
		}
	}

	// Parse JSON fields
	for _, f := range res.Fields {
		if f.Type == TypeJSON {
			if v, ok := row[f.Name]; ok {
				if str, ok := v.(string); ok && str != "" {
					var parsed any
					if err := json.Unmarshal([]byte(str), &parsed); err == nil {
						row[f.Name] = parsed
					}
				}
			}
		}
	}

	// Parse timestamps
	for _, name := range []string{"created_at", "updated_at"} {
		if v, ok := row[name]; ok {
			if str, ok := v.(string); ok {
				if t, err := time.Parse(time.RFC3339, str); err == nil {
					row[name] = t
				} else if t, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
					row[name] = t
				}
			}
		}
	}
	for _, f := range res.Fields {
		if f.Type == TypeTimestamp {
			if v, ok := row[f.Name]; ok {
				if str, ok := v.(string); ok && str != "" {
					if t, err := time.Parse(time.RFC3339, str); err == nil {
						row[f.Name] = t
					} else if t, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
						row[f.Name] = t
					}
				}
			}
		}
	}
}

// validate validates field constraints on the data.
func (s *Store) validate(res *Resource, data map[string]any) error {
	for _, f := range res.Fields {
		v, exists := data[f.Name]

		if f.Required && (!exists || v == nil || v == "") {
			return fmt.Errorf("%w: %s is required", ErrValidation, f.Name)
		}

		if !exists || v == nil {
			continue
		}

		// String validations
		if str, ok := v.(string); ok {
			if f.MinLen != nil && len(str) < *f.MinLen {
				return fmt.Errorf("%w: %s must be at least %d characters", ErrValidation, f.Name, *f.MinLen)
			}
			if f.MaxLen != nil && len(str) > *f.MaxLen {
				return fmt.Errorf("%w: %s must be at most %d characters", ErrValidation, f.Name, *f.MaxLen)
			}
			if f.Pattern != nil && !f.Pattern.MatchString(str) {
				return fmt.Errorf("%w: %s has invalid format", ErrValidation, f.Name)
			}
		}

		// Int validations
		if f.MinInt != nil {
			if intVal, ok := toInt64(v); ok && intVal < *f.MinInt {
				return fmt.Errorf("%w: %s must be >= %d", ErrValidation, f.Name, *f.MinInt)
			}
		}
		if f.MaxInt != nil {
			if intVal, ok := toInt64(v); ok && intVal > *f.MaxInt {
				return fmt.Errorf("%w: %s must be <= %d", ErrValidation, f.Name, *f.MaxInt)
			}
		}
	}
	return nil
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		return int64(val), true
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return i, true
		}
	}
	return 0, false
}
