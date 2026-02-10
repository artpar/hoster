package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// parseSQLiteTime parses a time string that may be RFC3339 (from Go code)
// or SQLite datetime format "2006-01-02 15:04:05" (from migrations).
func parseSQLiteTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	return time.Time{}
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

// =============================================================================
// Executor Interface - Shared by DB and Transaction
// =============================================================================

// executor abstracts database operations that can be performed on both
// a database connection and a transaction.
type executor interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// resolveRefID resolves a string reference_id to its integer primary key.
// The store layer handles ref_id → int_id conversion for all FK relationships.
func resolveRefID(ctx context.Context, exec executor, table, refID string) (int, error) {
	var id int
	query := fmt.Sprintf("SELECT id FROM %s WHERE reference_id = ?", table)
	if err := exec.GetContext(ctx, &id, query, refID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, NewStoreError("resolveRefID", table, refID, table+" not found", ErrNotFound)
		}
		return 0, NewStoreError("resolveRefID", table, refID, err.Error(), err)
	}
	return id, nil
}

// =============================================================================
// SQLiteStore
// =============================================================================

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sqlx.DB
}

// NewSQLiteStore creates a new SQLite store and runs migrations.
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	// Open database connection
	db, err := sqlx.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, NewStoreError("NewSQLiteStore", "", "", "failed to open database", ErrConnectionFailed)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, NewStoreError("NewSQLiteStore", "", "", "failed to ping database", ErrConnectionFailed)
	}

	// Run migrations
	if err := runMigrations(db.DB); err != nil {
		db.Close()
		return nil, NewStoreError("NewSQLiteStore", "", "", err.Error(), ErrMigrationFailed)
	}

	return &SQLiteStore{db: db}, nil
}

// runMigrations runs database migrations using embedded SQL files.
func runMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{NoTxWrap: true})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// =============================================================================
// User Resolution
// =============================================================================

func (s *SQLiteStore) ResolveUser(ctx context.Context, referenceID, email, name, planID string) (int, error) {
	return resolveUser(ctx, s.db, referenceID, email, name, planID)
}

func (s *txSQLiteStore) ResolveUser(ctx context.Context, referenceID, email, name, planID string) (int, error) {
	return resolveUser(ctx, s.tx, referenceID, email, name, planID)
}

func resolveUser(ctx context.Context, exec executor, referenceID, email, name, planID string) (int, error) {
	// Upsert user: insert if not exists, update fields if they changed
	_, err := exec.ExecContext(ctx, `
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
	err = exec.GetContext(ctx, &userID, "SELECT id FROM users WHERE reference_id = ?", referenceID)
	if err != nil {
		return 0, fmt.Errorf("resolve user: %w", err)
	}
	return userID, nil
}

// =============================================================================
// Template Operations
// =============================================================================

// templateRow represents a template row in the database.
type templateRow struct {
	ID                   int     `db:"id"`
	ReferenceID          string  `db:"reference_id"`
	Name                 string  `db:"name"`
	Slug                 string  `db:"slug"`
	Description          string  `db:"description"`
	Version              string  `db:"version"`
	ComposeSpec          string  `db:"compose_spec"`
	Variables            *string `db:"variables"`
	ConfigFiles          *string `db:"config_files"`
	ResourcesCPU         float64 `db:"resources_cpu_cores"`
	ResourcesMemory      int64   `db:"resources_memory_mb"`
	ResourcesDisk        int64   `db:"resources_disk_mb"`
	PriceMonthly         int64   `db:"price_monthly_cents"`
	Category             string  `db:"category"`
	Tags                 *string `db:"tags"`
	RequiredCapabilities *string `db:"required_capabilities"`
	Published            bool    `db:"published"`
	CreatorID            int     `db:"creator_id"`
	CreatedAt            string  `db:"created_at"`
	UpdatedAt            string  `db:"updated_at"`
}

func (s *SQLiteStore) CreateTemplate(ctx context.Context, template *domain.Template) error {
	return createTemplate(ctx, s.db, template)
}

func (s *SQLiteStore) GetTemplate(ctx context.Context, id string) (*domain.Template, error) {
	return getTemplate(ctx, s.db, id)
}

func (s *SQLiteStore) GetTemplateBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	return getTemplateBySlug(ctx, s.db, slug)
}

func (s *SQLiteStore) UpdateTemplate(ctx context.Context, template *domain.Template) error {
	return updateTemplate(ctx, s.db, template)
}

func (s *SQLiteStore) DeleteTemplate(ctx context.Context, id string) error {
	return deleteTemplate(ctx, s.db, id)
}

func (s *SQLiteStore) ListTemplates(ctx context.Context, opts ListOptions) ([]domain.Template, error) {
	return listTemplates(ctx, s.db, opts)
}

// =============================================================================
// Deployment Operations
// =============================================================================

// deploymentRow represents a deployment row in the database.
type deploymentRow struct {
	ID                  int     `db:"id"`
	ReferenceID         string  `db:"reference_id"`
	Name                string  `db:"name"`
	TemplateID          int     `db:"template_id"`
	TemplateRefID       *string `db:"template_reference_id"` // populated via JOIN
	TemplateVersion     string  `db:"template_version"`
	CustomerID          int     `db:"customer_id"`
	NodeID              string  `db:"node_id"`
	Status              string  `db:"status"`
	Variables           *string `db:"variables"`
	Domains             *string `db:"domains"`
	Containers          *string `db:"containers"`
	ResourcesCPU        float64 `db:"resources_cpu_cores"`
	ResourcesMemory     int64   `db:"resources_memory_mb"`
	ResourcesDisk       int64   `db:"resources_disk_mb"`
	ProxyPort           *int    `db:"proxy_port"`
	ErrorMessage        string  `db:"error_message"`
	CreatedAt           string  `db:"created_at"`
	UpdatedAt           string  `db:"updated_at"`
	StartedAt           *string `db:"started_at"`
	StoppedAt           *string `db:"stopped_at"`
}

// deploymentSelectColumns is the standard column list for deployment queries with template JOIN.
const deploymentSelectColumns = `
	d.id, d.reference_id, d.name, d.template_id, t.reference_id AS template_reference_id,
	d.template_version, d.customer_id, d.node_id, d.status,
	d.variables, d.domains, d.containers,
	d.resources_cpu_cores, d.resources_memory_mb, d.resources_disk_mb,
	d.proxy_port, d.error_message, d.created_at, d.updated_at, d.started_at, d.stopped_at`

// deploymentFromClause is the standard FROM clause for deployment queries with template JOIN.
const deploymentFromClause = `FROM deployments d LEFT JOIN templates t ON t.id = d.template_id`

func (s *SQLiteStore) CreateDeployment(ctx context.Context, deployment *domain.Deployment) error {
	return createDeployment(ctx, s.db, deployment)
}

func (s *SQLiteStore) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	return getDeployment(ctx, s.db, id)
}

func (s *SQLiteStore) UpdateDeployment(ctx context.Context, deployment *domain.Deployment) error {
	return updateDeployment(ctx, s.db, deployment)
}

func (s *SQLiteStore) DeleteDeployment(ctx context.Context, id string) error {
	return deleteDeployment(ctx, s.db, id)
}

func (s *SQLiteStore) ListDeployments(ctx context.Context, opts ListOptions) ([]domain.Deployment, error) {
	return listDeployments(ctx, s.db, opts)
}

func (s *SQLiteStore) ListDeploymentsByTemplate(ctx context.Context, templateID string, opts ListOptions) ([]domain.Deployment, error) {
	return listDeploymentsByTemplate(ctx, s.db, templateID, opts)
}

func (s *SQLiteStore) ListDeploymentsByCustomer(ctx context.Context, customerID int, opts ListOptions) ([]domain.Deployment, error) {
	return listDeploymentsByCustomer(ctx, s.db, customerID, opts)
}

// =============================================================================
// Transaction Support
// =============================================================================

func (s *SQLiteStore) WithTx(ctx context.Context, fn func(Store) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return NewStoreError("WithTx", "", "", "failed to begin transaction", ErrTxFailed)
	}

	txS := &txSQLiteStore{tx: tx}

	if err := fn(txS); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return NewStoreError("WithTx", "", "", fmt.Sprintf("rollback failed after error: %v", err), ErrTxFailed)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return NewStoreError("WithTx", "", "", "failed to commit transaction", ErrTxFailed)
	}

	return nil
}

// =============================================================================
// Transaction Store
// =============================================================================

// txSQLiteStore implements Store within a transaction.
type txSQLiteStore struct {
	tx *sqlx.Tx
}

func (s *txSQLiteStore) CreateTemplate(ctx context.Context, template *domain.Template) error {
	return createTemplate(ctx, s.tx, template)
}

func (s *txSQLiteStore) GetTemplate(ctx context.Context, id string) (*domain.Template, error) {
	return getTemplate(ctx, s.tx, id)
}

func (s *txSQLiteStore) GetTemplateBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	return getTemplateBySlug(ctx, s.tx, slug)
}

func (s *txSQLiteStore) UpdateTemplate(ctx context.Context, template *domain.Template) error {
	return updateTemplate(ctx, s.tx, template)
}

func (s *txSQLiteStore) DeleteTemplate(ctx context.Context, id string) error {
	return deleteTemplate(ctx, s.tx, id)
}

func (s *txSQLiteStore) ListTemplates(ctx context.Context, opts ListOptions) ([]domain.Template, error) {
	return listTemplates(ctx, s.tx, opts)
}

func (s *txSQLiteStore) CreateDeployment(ctx context.Context, deployment *domain.Deployment) error {
	return createDeployment(ctx, s.tx, deployment)
}

func (s *txSQLiteStore) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	return getDeployment(ctx, s.tx, id)
}

func (s *txSQLiteStore) UpdateDeployment(ctx context.Context, deployment *domain.Deployment) error {
	return updateDeployment(ctx, s.tx, deployment)
}

func (s *txSQLiteStore) DeleteDeployment(ctx context.Context, id string) error {
	return deleteDeployment(ctx, s.tx, id)
}

func (s *txSQLiteStore) ListDeployments(ctx context.Context, opts ListOptions) ([]domain.Deployment, error) {
	return listDeployments(ctx, s.tx, opts)
}

func (s *txSQLiteStore) ListDeploymentsByTemplate(ctx context.Context, templateID string, opts ListOptions) ([]domain.Deployment, error) {
	return listDeploymentsByTemplate(ctx, s.tx, templateID, opts)
}

func (s *txSQLiteStore) ListDeploymentsByCustomer(ctx context.Context, customerID int, opts ListOptions) ([]domain.Deployment, error) {
	return listDeploymentsByCustomer(ctx, s.tx, customerID, opts)
}

func (s *txSQLiteStore) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
	return getDeploymentByDomain(ctx, s.tx, hostname)
}

func (s *txSQLiteStore) GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error) {
	return getUsedProxyPorts(ctx, s.tx, nodeID)
}

func (s *txSQLiteStore) CountRoutableDeployments(ctx context.Context) (int, error) {
	return countRoutableDeployments(ctx, s.tx)
}

func (s *txSQLiteStore) WithTx(ctx context.Context, fn func(Store) error) error {
	// Already in a transaction, just run the function
	return fn(s)
}

func (s *txSQLiteStore) Close() error {
	// No-op for tx store
	return nil
}

func (s *txSQLiteStore) CreateUsageEvent(ctx context.Context, event *domain.MeterEvent) error {
	return createUsageEvent(ctx, s.tx, event)
}

func (s *txSQLiteStore) GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error) {
	return getUnreportedEvents(ctx, s.tx, limit)
}

func (s *txSQLiteStore) MarkEventsReported(ctx context.Context, ids []string, reportedAt time.Time) error {
	return markEventsReported(ctx, s.tx, ids, reportedAt)
}

func (s *txSQLiteStore) CreateNode(ctx context.Context, node *domain.Node) error {
	return createNode(ctx, s.tx, node)
}

func (s *txSQLiteStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	return getNode(ctx, s.tx, id)
}

func (s *txSQLiteStore) UpdateNode(ctx context.Context, node *domain.Node) error {
	return updateNode(ctx, s.tx, node)
}

func (s *txSQLiteStore) DeleteNode(ctx context.Context, id string) error {
	return deleteNode(ctx, s.tx, id)
}

func (s *txSQLiteStore) ListNodesByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.Node, error) {
	return listNodesByCreator(ctx, s.tx, creatorID, opts)
}

func (s *txSQLiteStore) ListOnlineNodes(ctx context.Context) ([]domain.Node, error) {
	return listOnlineNodes(ctx, s.tx)
}

func (s *txSQLiteStore) ListCheckableNodes(ctx context.Context) ([]domain.Node, error) {
	return listCheckableNodes(ctx, s.tx)
}

func (s *txSQLiteStore) CreateSSHKey(ctx context.Context, key *domain.SSHKey) error {
	return createSSHKey(ctx, s.tx, key)
}

func (s *txSQLiteStore) GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error) {
	return getSSHKey(ctx, s.tx, id)
}

func (s *txSQLiteStore) DeleteSSHKey(ctx context.Context, id string) error {
	return deleteSSHKey(ctx, s.tx, id)
}

func (s *txSQLiteStore) ListSSHKeysByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.SSHKey, error) {
	return listSSHKeysByCreator(ctx, s.tx, creatorID, opts)
}

func (s *txSQLiteStore) CreateCloudCredential(ctx context.Context, cred *domain.CloudCredential) error {
	return createCloudCredential(ctx, s.tx, cred)
}

func (s *txSQLiteStore) GetCloudCredential(ctx context.Context, id string) (*domain.CloudCredential, error) {
	return getCloudCredential(ctx, s.tx, id)
}

func (s *txSQLiteStore) DeleteCloudCredential(ctx context.Context, id string) error {
	return deleteCloudCredential(ctx, s.tx, id)
}

func (s *txSQLiteStore) ListCloudCredentialsByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.CloudCredential, error) {
	return listCloudCredentialsByCreator(ctx, s.tx, creatorID, opts)
}

func (s *txSQLiteStore) CreateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error {
	return createCloudProvision(ctx, s.tx, prov)
}

func (s *txSQLiteStore) GetCloudProvision(ctx context.Context, id string) (*domain.CloudProvision, error) {
	return getCloudProvision(ctx, s.tx, id)
}

func (s *txSQLiteStore) UpdateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error {
	return updateCloudProvision(ctx, s.tx, prov)
}

func (s *txSQLiteStore) ListCloudProvisionsByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.CloudProvision, error) {
	return listCloudProvisionsByCreator(ctx, s.tx, creatorID, opts)
}

func (s *txSQLiteStore) ListActiveProvisions(ctx context.Context) ([]domain.CloudProvision, error) {
	return listActiveProvisions(ctx, s.tx)
}

func (s *txSQLiteStore) ListCloudProvisionsByCredential(ctx context.Context, credentialID int) ([]domain.CloudProvision, error) {
	return listCloudProvisionsByCredential(ctx, s.tx, credentialID)
}

func (s *txSQLiteStore) ListDeploymentsByNode(ctx context.Context, nodeRefID string) ([]domain.Deployment, error) {
	return listDeploymentsByNode(ctx, s.tx, nodeRefID)
}

func (s *txSQLiteStore) ListNodesBySSHKey(ctx context.Context, sshKeyID int) ([]domain.Node, error) {
	return listNodesBySSHKey(ctx, s.tx, sshKeyID)
}

// =============================================================================
// Shared Implementation Functions
// =============================================================================

func createTemplate(ctx context.Context, exec executor, template *domain.Template) error {
	// Serialize JSON fields
	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ReferenceID, "failed to serialize variables", ErrInvalidData)
	}
	configFilesJSON, err := json.Marshal(template.ConfigFiles)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ReferenceID, "failed to serialize config_files", ErrInvalidData)
	}
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ReferenceID, "failed to serialize tags", ErrInvalidData)
	}
	requiredCapabilitiesJSON, err := json.Marshal(template.RequiredCapabilities)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ReferenceID, "failed to serialize required_capabilities", ErrInvalidData)
	}

	query := `
		INSERT INTO templates (
			reference_id, name, slug, description, version, compose_spec, variables, config_files,
			resources_cpu_cores, resources_memory_mb, resources_disk_mb,
			price_monthly_cents, category, tags, required_capabilities, published, creator_id,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?, ?, ?,
			?, ?
		)`

	result, err := exec.ExecContext(ctx, query,
		template.ReferenceID,
		template.Name,
		template.Slug,
		template.Description,
		template.Version,
		template.ComposeSpec,
		string(variablesJSON),
		string(configFilesJSON),
		template.ResourceRequirements.CPUCores,
		template.ResourceRequirements.MemoryMB,
		template.ResourceRequirements.DiskMB,
		template.PriceMonthly,
		template.Category,
		string(tagsJSON),
		string(requiredCapabilitiesJSON),
		template.Published,
		template.CreatorID,
		template.CreatedAt.Format(time.RFC3339),
		template.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: templates.reference_id") {
			return NewStoreError("CreateTemplate", "template", template.ReferenceID, "template with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed: templates.slug") {
			return NewStoreError("CreateTemplate", "template", template.ReferenceID, "template with this slug already exists", ErrDuplicateSlug)
		}
		return NewStoreError("CreateTemplate", "template", template.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ReferenceID, "failed to get last insert ID", err)
	}
	template.ID = int(lastID)

	return nil
}

func getTemplate(ctx context.Context, exec executor, id string) (*domain.Template, error) {
	query := `SELECT id, reference_id, name, slug, description, version, compose_spec, variables, config_files,
		resources_cpu_cores, resources_memory_mb, resources_disk_mb,
		price_monthly_cents, category, tags, required_capabilities, published, creator_id,
		created_at, updated_at
		FROM templates WHERE reference_id = ?`

	var row templateRow
	err := exec.GetContext(ctx, &row, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetTemplate", "template", id, "template not found", ErrNotFound)
		}
		return nil, NewStoreError("GetTemplate", "template", id, err.Error(), err)
	}

	return rowToTemplate(&row)
}

func getTemplateBySlug(ctx context.Context, exec executor, slug string) (*domain.Template, error) {
	query := `SELECT id, reference_id, name, slug, description, version, compose_spec, variables, config_files,
		resources_cpu_cores, resources_memory_mb, resources_disk_mb,
		price_monthly_cents, category, tags, required_capabilities, published, creator_id,
		created_at, updated_at
		FROM templates WHERE slug = ?`

	var row templateRow
	err := exec.GetContext(ctx, &row, query, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetTemplateBySlug", "template", slug, "template not found", ErrNotFound)
		}
		return nil, NewStoreError("GetTemplateBySlug", "template", slug, err.Error(), err)
	}

	return rowToTemplate(&row)
}

func updateTemplate(ctx context.Context, exec executor, template *domain.Template) error {
	// Serialize JSON fields
	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ReferenceID, "failed to serialize variables", ErrInvalidData)
	}
	configFilesJSON, err := json.Marshal(template.ConfigFiles)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ReferenceID, "failed to serialize config_files", ErrInvalidData)
	}
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ReferenceID, "failed to serialize tags", ErrInvalidData)
	}
	requiredCapabilitiesJSON, err := json.Marshal(template.RequiredCapabilities)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ReferenceID, "failed to serialize required_capabilities", ErrInvalidData)
	}

	query := `
		UPDATE templates SET
			name = :name,
			slug = :slug,
			description = :description,
			version = :version,
			compose_spec = :compose_spec,
			variables = :variables,
			config_files = :config_files,
			resources_cpu_cores = :resources_cpu_cores,
			resources_memory_mb = :resources_memory_mb,
			resources_disk_mb = :resources_disk_mb,
			price_monthly_cents = :price_monthly_cents,
			category = :category,
			tags = :tags,
			required_capabilities = :required_capabilities,
			published = :published,
			creator_id = :creator_id,
			updated_at = :updated_at
		WHERE reference_id = :reference_id`

	row := map[string]any{
		"reference_id":          template.ReferenceID,
		"name":                  template.Name,
		"slug":                  template.Slug,
		"description":           template.Description,
		"version":               template.Version,
		"compose_spec":          template.ComposeSpec,
		"variables":             string(variablesJSON),
		"config_files":          string(configFilesJSON),
		"resources_cpu_cores":   template.ResourceRequirements.CPUCores,
		"resources_memory_mb":   template.ResourceRequirements.MemoryMB,
		"resources_disk_mb":     template.ResourceRequirements.DiskMB,
		"price_monthly_cents":   template.PriceMonthly,
		"category":              template.Category,
		"tags":                  string(tagsJSON),
		"required_capabilities": string(requiredCapabilitiesJSON),
		"published":             template.Published,
		"creator_id":            template.CreatorID,
		"updated_at":            template.UpdatedAt.Format(time.RFC3339),
	}

	result, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ReferenceID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateTemplate", "template", template.ReferenceID, "template not found", ErrNotFound)
	}

	return nil
}

func deleteTemplate(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM templates WHERE reference_id = ?`

	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return NewStoreError("DeleteTemplate", "template", id, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("DeleteTemplate", "template", id, "template not found", ErrNotFound)
	}

	return nil
}

func listTemplates(ctx context.Context, exec executor, opts ListOptions) ([]domain.Template, error) {
	opts = opts.Normalize()
	query := `SELECT id, reference_id, name, slug, description, version, compose_spec, variables, config_files,
		resources_cpu_cores, resources_memory_mb, resources_disk_mb,
		price_monthly_cents, category, tags, required_capabilities, published, creator_id,
		created_at, updated_at
		FROM templates ORDER BY created_at DESC LIMIT ? OFFSET ?`

	var rows []templateRow
	err := exec.SelectContext(ctx, &rows, query, opts.Limit, opts.Offset)
	if err != nil {
		return nil, NewStoreError("ListTemplates", "template", "", err.Error(), err)
	}

	templates := make([]domain.Template, 0, len(rows))
	for _, row := range rows {
		template, err := rowToTemplate(&row)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	return templates, nil
}

func createDeployment(ctx context.Context, exec executor, deployment *domain.Deployment) error {
	// Serialize JSON fields
	variablesJSON, err := json.Marshal(deployment.Variables)
	if err != nil {
		return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "failed to serialize variables", ErrInvalidData)
	}
	domainsJSON, err := json.Marshal(deployment.Domains)
	if err != nil {
		return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "failed to serialize domains", ErrInvalidData)
	}
	containersJSON, err := json.Marshal(deployment.Containers)
	if err != nil {
		return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "failed to serialize containers", ErrInvalidData)
	}

	var startedAt, stoppedAt *string
	if deployment.StartedAt != nil {
		s := deployment.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	if deployment.StoppedAt != nil {
		s := deployment.StoppedAt.Format(time.RFC3339)
		stoppedAt = &s
	}

	if deployment.TemplateID == 0 && deployment.TemplateRefID != "" {
		resolved, err := resolveRefID(ctx, exec, "templates", deployment.TemplateRefID)
		if err != nil {
			return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "failed to resolve template reference", err)
		}
		deployment.TemplateID = resolved
	}

	var proxyPort *int
	if deployment.ProxyPort > 0 {
		proxyPort = &deployment.ProxyPort
	}

	query := `
		INSERT INTO deployments (
			reference_id, name, template_id, template_version, customer_id, node_id,
			status, variables, domains, containers,
			resources_cpu_cores, resources_memory_mb, resources_disk_mb,
			proxy_port, error_message, created_at, updated_at, started_at, stopped_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?, ?, ?
		)`

	result, err := exec.ExecContext(ctx, query,
		deployment.ReferenceID,
		deployment.Name,
		deployment.TemplateID,
		deployment.TemplateVersion,
		deployment.CustomerID,
		deployment.NodeID,
		string(deployment.Status),
		string(variablesJSON),
		string(domainsJSON),
		string(containersJSON),
		deployment.Resources.CPUCores,
		deployment.Resources.MemoryMB,
		deployment.Resources.DiskMB,
		proxyPort,
		deployment.ErrorMessage,
		deployment.CreatedAt.Format(time.RFC3339),
		deployment.UpdatedAt.Format(time.RFC3339),
		startedAt,
		stoppedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: deployments.reference_id") {
			return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "deployment with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "template not found", ErrForeignKey)
		}
		return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateDeployment", "deployment", deployment.ReferenceID, "failed to get last insert ID", err)
	}
	deployment.ID = int(lastID)

	return nil
}

func getDeployment(ctx context.Context, exec executor, id string) (*domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + ` ` + deploymentFromClause + ` WHERE d.reference_id = ?`

	var row deploymentRow
	err := exec.GetContext(ctx, &row, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetDeployment", "deployment", id, "deployment not found", ErrNotFound)
		}
		return nil, NewStoreError("GetDeployment", "deployment", id, err.Error(), err)
	}

	return rowToDeployment(&row)
}

func updateDeployment(ctx context.Context, exec executor, deployment *domain.Deployment) error {
	// Serialize JSON fields
	variablesJSON, err := json.Marshal(deployment.Variables)
	if err != nil {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ReferenceID, "failed to serialize variables", ErrInvalidData)
	}
	domainsJSON, err := json.Marshal(deployment.Domains)
	if err != nil {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ReferenceID, "failed to serialize domains", ErrInvalidData)
	}
	containersJSON, err := json.Marshal(deployment.Containers)
	if err != nil {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ReferenceID, "failed to serialize containers", ErrInvalidData)
	}

	var startedAt, stoppedAt *string
	if deployment.StartedAt != nil {
		s := deployment.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	if deployment.StoppedAt != nil {
		s := deployment.StoppedAt.Format(time.RFC3339)
		stoppedAt = &s
	}

	if deployment.TemplateID == 0 && deployment.TemplateRefID != "" {
		resolved, err := resolveRefID(ctx, exec, "templates", deployment.TemplateRefID)
		if err != nil {
			return NewStoreError("UpdateDeployment", "deployment", deployment.ReferenceID, "failed to resolve template reference", err)
		}
		deployment.TemplateID = resolved
	}

	var proxyPort *int
	if deployment.ProxyPort > 0 {
		proxyPort = &deployment.ProxyPort
	}

	query := `
		UPDATE deployments SET
			name = :name,
			template_id = :template_id,
			template_version = :template_version,
			customer_id = :customer_id,
			node_id = :node_id,
			status = :status,
			variables = :variables,
			domains = :domains,
			containers = :containers,
			resources_cpu_cores = :resources_cpu_cores,
			resources_memory_mb = :resources_memory_mb,
			resources_disk_mb = :resources_disk_mb,
			proxy_port = :proxy_port,
			error_message = :error_message,
			updated_at = :updated_at,
			started_at = :started_at,
			stopped_at = :stopped_at
		WHERE reference_id = :reference_id`

	row := map[string]any{
		"reference_id":         deployment.ReferenceID,
		"name":                 deployment.Name,
		"template_id":          deployment.TemplateID,
		"template_version":     deployment.TemplateVersion,
		"customer_id":          deployment.CustomerID,
		"node_id":              deployment.NodeID,
		"status":               string(deployment.Status),
		"variables":            string(variablesJSON),
		"domains":              string(domainsJSON),
		"containers":           string(containersJSON),
		"resources_cpu_cores":  deployment.Resources.CPUCores,
		"resources_memory_mb":  deployment.Resources.MemoryMB,
		"resources_disk_mb":    deployment.Resources.DiskMB,
		"proxy_port":           proxyPort,
		"error_message":        deployment.ErrorMessage,
		"updated_at":           deployment.UpdatedAt.Format(time.RFC3339),
		"started_at":           startedAt,
		"stopped_at":           stoppedAt,
	}

	result, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ReferenceID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ReferenceID, "deployment not found", ErrNotFound)
	}

	return nil
}

func deleteDeployment(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM deployments WHERE reference_id = ?`

	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return NewStoreError("DeleteDeployment", "deployment", id, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("DeleteDeployment", "deployment", id, "deployment not found", ErrNotFound)
	}

	return nil
}

func listDeployments(ctx context.Context, exec executor, opts ListOptions) ([]domain.Deployment, error) {
	opts = opts.Normalize()
	query := `SELECT ` + deploymentSelectColumns + ` ` + deploymentFromClause + ` ORDER BY d.created_at DESC LIMIT ? OFFSET ?`

	var rows []deploymentRow
	err := exec.SelectContext(ctx, &rows, query, opts.Limit, opts.Offset)
	if err != nil {
		return nil, NewStoreError("ListDeployments", "deployment", "", err.Error(), err)
	}

	deployments := make([]domain.Deployment, 0, len(rows))
	for _, row := range rows {
		deployment, err := rowToDeployment(&row)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *deployment)
	}

	return deployments, nil
}

func listDeploymentsByTemplate(ctx context.Context, exec executor, templateID string, opts ListOptions) ([]domain.Deployment, error) {
	opts = opts.Normalize()
	query := `SELECT ` + deploymentSelectColumns + ` ` + deploymentFromClause + `
		WHERE d.template_id = (SELECT id FROM templates WHERE reference_id = ?)
		ORDER BY d.created_at DESC LIMIT ? OFFSET ?`

	var rows []deploymentRow
	err := exec.SelectContext(ctx, &rows, query, templateID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, NewStoreError("ListDeploymentsByTemplate", "deployment", "", err.Error(), err)
	}

	deployments := make([]domain.Deployment, 0, len(rows))
	for _, row := range rows {
		deployment, err := rowToDeployment(&row)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *deployment)
	}

	return deployments, nil
}

func listDeploymentsByCustomer(ctx context.Context, exec executor, customerID int, opts ListOptions) ([]domain.Deployment, error) {
	opts = opts.Normalize()
	query := `SELECT ` + deploymentSelectColumns + ` ` + deploymentFromClause + `
		WHERE d.customer_id = ? ORDER BY d.created_at DESC LIMIT ? OFFSET ?`

	var rows []deploymentRow
	err := exec.SelectContext(ctx, &rows, query, customerID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, NewStoreError("ListDeploymentsByCustomer", "deployment", "", err.Error(), err)
	}

	deployments := make([]domain.Deployment, 0, len(rows))
	for _, row := range rows {
		deployment, err := rowToDeployment(&row)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *deployment)
	}

	return deployments, nil
}

// =============================================================================
// Proxy-related Deployment Operations
// =============================================================================

// GetDeploymentByDomain finds a deployment by its domain hostname.
func (s *SQLiteStore) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
	return getDeploymentByDomain(ctx, s.db, hostname)
}

func getDeploymentByDomain(ctx context.Context, exec executor, hostname string) (*domain.Deployment, error) {
	// Query for deployment where any domain in the JSON array matches the hostname
	// Uses json_each() to handle any number of domains per deployment
	query := `
		SELECT ` + deploymentSelectColumns + ` ` + deploymentFromClause + `
		WHERE EXISTS (
			SELECT 1 FROM json_each(d.domains) AS je
			WHERE json_extract(je.value, '$.hostname') = ?
		)
		LIMIT 1
	`

	var row deploymentRow
	if err := exec.GetContext(ctx, &row, query, hostname); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetDeploymentByDomain", "deployment", hostname, "deployment not found for hostname", ErrNotFound)
		}
		return nil, NewStoreError("GetDeploymentByDomain", "deployment", hostname, err.Error(), err)
	}

	return rowToDeployment(&row)
}

// GetUsedProxyPorts returns all proxy ports in use on a node.
func (s *SQLiteStore) GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error) {
	return getUsedProxyPorts(ctx, s.db, nodeID)
}

func getUsedProxyPorts(ctx context.Context, exec executor, nodeID string) ([]int, error) {
	query := `
		SELECT proxy_port FROM deployments
		WHERE node_id = ? AND proxy_port IS NOT NULL AND status != 'deleted'
	`

	var ports []int
	if err := exec.SelectContext(ctx, &ports, query, nodeID); err != nil {
		return nil, NewStoreError("GetUsedProxyPorts", "deployment", nodeID, err.Error(), err)
	}

	return ports, nil
}

// CountRoutableDeployments returns the count of deployments that can be routed to.
// A routable deployment is one that is running and has a proxy port assigned.
func (s *SQLiteStore) CountRoutableDeployments(ctx context.Context) (int, error) {
	return countRoutableDeployments(ctx, s.db)
}

func countRoutableDeployments(ctx context.Context, exec executor) (int, error) {
	query := `
		SELECT COUNT(*) FROM deployments
		WHERE status = 'running' AND proxy_port IS NOT NULL
	`

	var count int
	if err := exec.GetContext(ctx, &count, query); err != nil {
		return 0, NewStoreError("CountRoutableDeployments", "deployment", "", err.Error(), err)
	}

	return count, nil
}

// =============================================================================
// Row Conversion Functions
// =============================================================================

// rowToTemplate converts a database row to a domain.Template.
func rowToTemplate(row *templateRow) (*domain.Template, error) {
	createdAt := parseSQLiteTime(row.CreatedAt)
	updatedAt := parseSQLiteTime(row.UpdatedAt)

	var variables []domain.Variable
	if row.Variables != nil && *row.Variables != "" && *row.Variables != "null" {
		if err := json.Unmarshal([]byte(*row.Variables), &variables); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ReferenceID, "failed to parse variables", ErrInvalidData)
		}
	}

	var configFiles []domain.ConfigFile
	if row.ConfigFiles != nil && *row.ConfigFiles != "" && *row.ConfigFiles != "null" {
		if err := json.Unmarshal([]byte(*row.ConfigFiles), &configFiles); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ReferenceID, "failed to parse config_files", ErrInvalidData)
		}
	}

	var tags []string
	if row.Tags != nil && *row.Tags != "" && *row.Tags != "null" {
		if err := json.Unmarshal([]byte(*row.Tags), &tags); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ReferenceID, "failed to parse tags", ErrInvalidData)
		}
	}

	var requiredCapabilities []string
	if row.RequiredCapabilities != nil && *row.RequiredCapabilities != "" && *row.RequiredCapabilities != "null" {
		if err := json.Unmarshal([]byte(*row.RequiredCapabilities), &requiredCapabilities); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ReferenceID, "failed to parse required_capabilities", ErrInvalidData)
		}
	}

	return &domain.Template{
		ID:          row.ID,
		ReferenceID: row.ReferenceID,
		Name:        row.Name,
		Slug:        row.Slug,
		Description: row.Description,
		Version:     row.Version,
		ComposeSpec: row.ComposeSpec,
		Variables:   variables,
		ConfigFiles: configFiles,
		ResourceRequirements: domain.Resources{
			CPUCores: row.ResourcesCPU,
			MemoryMB: row.ResourcesMemory,
			DiskMB:   row.ResourcesDisk,
		},
		PriceMonthly:         row.PriceMonthly,
		Category:             row.Category,
		Tags:                 tags,
		RequiredCapabilities: requiredCapabilities,
		Published:            row.Published,
		CreatorID:            row.CreatorID,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}, nil
}

// rowToDeployment converts a database row to a domain.Deployment.
func rowToDeployment(row *deploymentRow) (*domain.Deployment, error) {
	createdAt := parseSQLiteTime(row.CreatedAt)
	updatedAt := parseSQLiteTime(row.UpdatedAt)

	var startedAt, stoppedAt *time.Time
	if row.StartedAt != nil && *row.StartedAt != "" {
		t := parseSQLiteTime(*row.StartedAt)
		startedAt = &t
	}
	if row.StoppedAt != nil && *row.StoppedAt != "" {
		t := parseSQLiteTime(*row.StoppedAt)
		stoppedAt = &t
	}

	var variables map[string]string
	if row.Variables != nil && *row.Variables != "" && *row.Variables != "null" {
		if err := json.Unmarshal([]byte(*row.Variables), &variables); err != nil {
			return nil, NewStoreError("rowToDeployment", "deployment", row.ReferenceID, "failed to parse variables", ErrInvalidData)
		}
	}

	var domains []domain.Domain
	if row.Domains != nil && *row.Domains != "" && *row.Domains != "null" {
		if err := json.Unmarshal([]byte(*row.Domains), &domains); err != nil {
			return nil, NewStoreError("rowToDeployment", "deployment", row.ReferenceID, "failed to parse domains", ErrInvalidData)
		}
	}

	var containers []domain.ContainerInfo
	if row.Containers != nil && *row.Containers != "" && *row.Containers != "null" {
		if err := json.Unmarshal([]byte(*row.Containers), &containers); err != nil {
			return nil, NewStoreError("rowToDeployment", "deployment", row.ReferenceID, "failed to parse containers", ErrInvalidData)
		}
	}

	var proxyPort int
	if row.ProxyPort != nil {
		proxyPort = *row.ProxyPort
	}

	var templateRefID string
	if row.TemplateRefID != nil {
		templateRefID = *row.TemplateRefID
	}

	return &domain.Deployment{
		ID:              row.ID,
		ReferenceID:     row.ReferenceID,
		Name:            row.Name,
		TemplateID:      row.TemplateID,
		TemplateRefID:   templateRefID,
		TemplateVersion: row.TemplateVersion,
		CustomerID:      row.CustomerID,
		NodeID:          row.NodeID,
		Status:          domain.DeploymentStatus(row.Status),
		Variables:       variables,
		Domains:         domains,
		Containers:      containers,
		Resources: domain.Resources{
			CPUCores: row.ResourcesCPU,
			MemoryMB: row.ResourcesMemory,
			DiskMB:   row.ResourcesDisk,
		},
		ProxyPort:    proxyPort,
		ErrorMessage: row.ErrorMessage,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		StartedAt:    startedAt,
		StoppedAt:    stoppedAt,
	}, nil
}

// =============================================================================
// Usage Event Operations (F009: Billing Integration)
// =============================================================================

// usageEventRow represents a usage event row in the database.
type usageEventRow struct {
	ID           int     `db:"id"`
	ReferenceID  string  `db:"reference_id"`
	UserID       int     `db:"user_id"`
	EventType    string  `db:"event_type"`
	ResourceID   string  `db:"resource_id"`
	ResourceType string  `db:"resource_type"`
	Quantity     int64   `db:"quantity"`
	Metadata     *string `db:"metadata"`
	Timestamp    string  `db:"timestamp"`
	ReportedAt   *string `db:"reported_at"`
	CreatedAt    string  `db:"created_at"`
}

// CreateUsageEvent inserts a new usage event.
func (s *SQLiteStore) CreateUsageEvent(ctx context.Context, event *domain.MeterEvent) error {
	return createUsageEvent(ctx, s.db, event)
}

func createUsageEvent(ctx context.Context, exec executor, event *domain.MeterEvent) error {
	var metadataJSON *string
	if len(event.Metadata) > 0 {
		data, err := json.Marshal(event.Metadata)
		if err != nil {
			return NewStoreError("CreateUsageEvent", "usage_event", event.ReferenceID, "failed to marshal metadata", ErrInvalidData)
		}
		s := string(data)
		metadataJSON = &s
	}

	query := `
		INSERT INTO usage_events (reference_id, user_id, event_type, resource_id, resource_type, quantity, metadata, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := exec.ExecContext(ctx, query,
		event.ReferenceID,
		event.UserID,
		string(event.EventType),
		event.ResourceID,
		event.ResourceType,
		event.Quantity,
		metadataJSON,
		event.Timestamp.Format(time.RFC3339),
		event.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return NewStoreError("CreateUsageEvent", "usage_event", event.ReferenceID, "event already exists", ErrDuplicateID)
		}
		return NewStoreError("CreateUsageEvent", "usage_event", event.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateUsageEvent", "usage_event", event.ReferenceID, "failed to get last insert ID", err)
	}
	event.ID = int(lastID)

	return nil
}

// GetUnreportedEvents retrieves usage events that haven't been reported to APIGate.
func (s *SQLiteStore) GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error) {
	return getUnreportedEvents(ctx, s.db, limit)
}

func getUnreportedEvents(ctx context.Context, exec executor, limit int) ([]domain.MeterEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, reference_id, user_id, event_type, resource_id, resource_type, quantity, metadata, timestamp, reported_at, created_at
		FROM usage_events
		WHERE reported_at IS NULL
		ORDER BY timestamp ASC
		LIMIT ?`

	var rows []usageEventRow
	if err := exec.SelectContext(ctx, &rows, query, limit); err != nil {
		return nil, NewStoreError("GetUnreportedEvents", "usage_event", "", err.Error(), err)
	}

	events := make([]domain.MeterEvent, 0, len(rows))
	for _, row := range rows {
		event, err := rowToUsageEvent(&row)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}

	return events, nil
}

// MarkEventsReported marks usage events as reported to APIGate.
func (s *SQLiteStore) MarkEventsReported(ctx context.Context, ids []string, reportedAt time.Time) error {
	return markEventsReported(ctx, s.db, ids, reportedAt)
}

func markEventsReported(ctx context.Context, exec executor, ids []string, reportedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}

	// Build placeholder string for IN clause
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	args[0] = reportedAt.Format(time.RFC3339)
	for i, id := range ids {
		placeholders[i] = "?"
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		UPDATE usage_events
		SET reported_at = ?
		WHERE reference_id IN (%s)`, strings.Join(placeholders, ","))

	_, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return NewStoreError("MarkEventsReported", "usage_event", "", err.Error(), err)
	}

	return nil
}

// rowToUsageEvent converts a database row to a domain.MeterEvent.
func rowToUsageEvent(row *usageEventRow) (*domain.MeterEvent, error) {
	timestamp := parseSQLiteTime(row.Timestamp)
	createdAt := parseSQLiteTime(row.CreatedAt)

	var reportedAt *time.Time
	if row.ReportedAt != nil && *row.ReportedAt != "" {
		t := parseSQLiteTime(*row.ReportedAt)
		reportedAt = &t
	}

	var metadata map[string]string
	if row.Metadata != nil && *row.Metadata != "" && *row.Metadata != "null" {
		if err := json.Unmarshal([]byte(*row.Metadata), &metadata); err != nil {
			return nil, NewStoreError("rowToUsageEvent", "usage_event", row.ReferenceID, "failed to parse metadata", ErrInvalidData)
		}
	}

	return &domain.MeterEvent{
		ID:           row.ID,
		ReferenceID:  row.ReferenceID,
		UserID:       row.UserID,
		EventType:    domain.EventType(row.EventType),
		ResourceID:   row.ResourceID,
		ResourceType: row.ResourceType,
		Quantity:     row.Quantity,
		Metadata:     metadata,
		Timestamp:    timestamp,
		ReportedAt:   reportedAt,
		CreatedAt:    createdAt,
	}, nil
}

// =============================================================================
// Container Events (F010: Monitoring)
// =============================================================================

// containerEventRow represents a database row for container events.
type containerEventRow struct {
	ID           int    `db:"id"`
	ReferenceID  string `db:"reference_id"`
	DeploymentID int    `db:"deployment_id"`
	Type         string `db:"type"`
	Container    string `db:"container"`
	Message      string `db:"message"`
	Timestamp    string `db:"timestamp"`
	CreatedAt    string `db:"created_at"`
}

// CreateContainerEvent stores a new container event.
func (s *SQLiteStore) CreateContainerEvent(ctx context.Context, event *domain.ContainerEvent) error {
	return createContainerEvent(ctx, s.db, event)
}

func (s *txSQLiteStore) CreateContainerEvent(ctx context.Context, event *domain.ContainerEvent) error {
	return createContainerEvent(ctx, s.tx, event)
}

func createContainerEvent(ctx context.Context, exec executor, event *domain.ContainerEvent) error {
	query := `
		INSERT INTO container_events (reference_id, deployment_id, type, container, message, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := exec.ExecContext(ctx, query,
		event.ReferenceID,
		event.DeploymentID,
		string(event.Type),
		event.Container,
		event.Message,
		event.Timestamp.Format(time.RFC3339),
		event.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return NewStoreError("CreateContainerEvent", "container_event", event.ReferenceID, "event already exists", ErrDuplicateID)
		}
		return NewStoreError("CreateContainerEvent", "container_event", event.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateContainerEvent", "container_event", event.ReferenceID, "failed to get last insert ID", err)
	}
	event.ID = int(lastID)

	return nil
}

// GetContainerEvents retrieves container events for a deployment.
func (s *SQLiteStore) GetContainerEvents(ctx context.Context, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error) {
	return getContainerEvents(ctx, s.db, deploymentID, limit, eventType)
}

func (s *txSQLiteStore) GetContainerEvents(ctx context.Context, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error) {
	return getContainerEvents(ctx, s.tx, deploymentID, limit, eventType)
}

func getContainerEvents(ctx context.Context, exec executor, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	var query string
	var args []any

	if eventType != nil && *eventType != "" {
		query = `
			SELECT id, reference_id, deployment_id, type, container, message, timestamp, created_at
			FROM container_events
			WHERE deployment_id = (SELECT id FROM deployments WHERE reference_id = ?) AND type = ?
			ORDER BY timestamp DESC
			LIMIT ?`
		args = []any{deploymentID, *eventType, limit}
	} else {
		query = `
			SELECT id, reference_id, deployment_id, type, container, message, timestamp, created_at
			FROM container_events
			WHERE deployment_id = (SELECT id FROM deployments WHERE reference_id = ?)
			ORDER BY timestamp DESC
			LIMIT ?`
		args = []any{deploymentID, limit}
	}

	var rows []containerEventRow
	if err := exec.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, NewStoreError("GetContainerEvents", "container_event", deploymentID, err.Error(), err)
	}

	events := make([]domain.ContainerEvent, 0, len(rows))
	for _, row := range rows {
		event, err := rowToContainerEvent(&row)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}

	return events, nil
}

// rowToContainerEvent converts a database row to a domain.ContainerEvent.
func rowToContainerEvent(row *containerEventRow) (*domain.ContainerEvent, error) {
	timestamp := parseSQLiteTime(row.Timestamp)
	createdAt := parseSQLiteTime(row.CreatedAt)

	return &domain.ContainerEvent{
		ID:           row.ID,
		ReferenceID:  row.ReferenceID,
		DeploymentID: row.DeploymentID,
		Type:         domain.ContainerEventType(row.Type),
		Container:    row.Container,
		Message:      row.Message,
		Timestamp:    timestamp,
		CreatedAt:    createdAt,
	}, nil
}

// =============================================================================
// Node Operations
// =============================================================================

// nodeRow represents a node row in the database.
type nodeRow struct {
	ID                   int     `db:"id"`
	ReferenceID          string  `db:"reference_id"`
	Name                 string  `db:"name"`
	CreatorID            int     `db:"creator_id"`
	SSHHost              string  `db:"ssh_host"`
	SSHPort              int     `db:"ssh_port"`
	SSHUser              string  `db:"ssh_user"`
	SSHKeyID             *int    `db:"ssh_key_id"`
	SSHKeyRefID          *string `db:"ssh_key_reference_id"` // populated via JOIN
	DockerSocket         string  `db:"docker_socket"`
	Status               string  `db:"status"`
	Capabilities         string  `db:"capabilities"`
	CapacityCPUCores     float64 `db:"capacity_cpu_cores"`
	CapacityMemoryMB     int64   `db:"capacity_memory_mb"`
	CapacityDiskMB       int64   `db:"capacity_disk_mb"`
	CapacityCPUUsed      float64 `db:"capacity_cpu_used"`
	CapacityMemoryUsedMB int64   `db:"capacity_memory_used_mb"`
	CapacityDiskUsedMB   int64   `db:"capacity_disk_used_mb"`
	Location             string  `db:"location"`
	LastHealthCheck      *string `db:"last_health_check"`
	ErrorMessage         string  `db:"error_message"`
	ProviderType         string  `db:"provider_type"`
	ProvisionID          string  `db:"provision_id"`
	BaseDomain           string  `db:"base_domain"`
	CreatedAt            string  `db:"created_at"`
	UpdatedAt            string  `db:"updated_at"`
}

// nodeSelectColumns is the standard column list for node queries with ssh_key JOIN.
const nodeSelectColumns = `
	n.id, n.reference_id, n.name, n.creator_id, n.ssh_host, n.ssh_port, n.ssh_user, n.ssh_key_id,
	sk.reference_id AS ssh_key_reference_id,
	n.docker_socket, n.status, n.capabilities,
	n.capacity_cpu_cores, n.capacity_memory_mb, n.capacity_disk_mb,
	n.capacity_cpu_used, n.capacity_memory_used_mb, n.capacity_disk_used_mb,
	n.location, n.last_health_check, n.error_message,
	n.provider_type, n.provision_id, n.base_domain,
	n.created_at, n.updated_at`

// nodeFromClause is the standard FROM clause for node queries with ssh_key JOIN.
const nodeFromClause = `FROM nodes n LEFT JOIN ssh_keys sk ON sk.id = n.ssh_key_id`

// CreateNode creates a new node in the database.
func (s *SQLiteStore) CreateNode(ctx context.Context, node *domain.Node) error {
	return createNode(ctx, s.db, node)
}

func createNode(ctx context.Context, exec executor, node *domain.Node) error {
	capabilities, err := json.Marshal(node.Capabilities)
	if err != nil {
		return NewStoreError("CreateNode", "node", node.ReferenceID, "failed to marshal capabilities", err)
	}

	var lastHealthCheck *string
	if node.LastHealthCheck != nil {
		hc := node.LastHealthCheck.Format(time.RFC3339)
		lastHealthCheck = &hc
	}

	if node.SSHKeyID == 0 && node.SSHKeyRefID != "" {
		resolved, err := resolveRefID(ctx, exec, "ssh_keys", node.SSHKeyRefID)
		if err != nil {
			return NewStoreError("CreateNode", "node", node.ReferenceID, "failed to resolve SSH key reference", err)
		}
		node.SSHKeyID = resolved
	}

	var sshKeyID *int
	if node.SSHKeyID != 0 {
		sshKeyID = &node.SSHKeyID
	}

	query := `
		INSERT INTO nodes (
			reference_id, name, creator_id, ssh_host, ssh_port, ssh_user, ssh_key_id,
			docker_socket, status, capabilities,
			capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
			capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
			location, last_health_check, error_message,
			provider_type, provision_id, base_domain,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?
		)`

	result, err := exec.ExecContext(ctx, query,
		node.ReferenceID,
		node.Name,
		node.CreatorID,
		node.SSHHost,
		node.SSHPort,
		node.SSHUser,
		sshKeyID,
		node.DockerSocket,
		string(node.Status),
		string(capabilities),
		node.Capacity.CPUCores,
		node.Capacity.MemoryMB,
		node.Capacity.DiskMB,
		node.Capacity.CPUUsed,
		node.Capacity.MemoryUsedMB,
		node.Capacity.DiskUsedMB,
		node.Location,
		lastHealthCheck,
		node.ErrorMessage,
		node.ProviderType,
		node.ProvisionID,
		node.BaseDomain,
		node.CreatedAt.Format(time.RFC3339),
		node.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: nodes.reference_id") {
			return NewStoreError("CreateNode", "node", node.ReferenceID, "node with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateNode", "node", node.ReferenceID, "node with this name already exists for creator", ErrDuplicateKey)
		}
		return NewStoreError("CreateNode", "node", node.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateNode", "node", node.ReferenceID, "failed to get last insert ID", err)
	}
	node.ID = int(lastID)

	return nil
}

// GetNode retrieves a node by ID.
func (s *SQLiteStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	return getNode(ctx, s.db, id)
}

func getNode(ctx context.Context, exec executor, id string) (*domain.Node, error) {
	var row nodeRow
	query := `SELECT ` + nodeSelectColumns + ` ` + nodeFromClause + ` WHERE n.reference_id = ?`

	if err := exec.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetNode", "node", id, "node not found", ErrNotFound)
		}
		return nil, NewStoreError("GetNode", "node", id, err.Error(), err)
	}

	return rowToNode(&row)
}

// UpdateNode updates an existing node.
func (s *SQLiteStore) UpdateNode(ctx context.Context, node *domain.Node) error {
	return updateNode(ctx, s.db, node)
}

func updateNode(ctx context.Context, exec executor, node *domain.Node) error {
	capabilities, err := json.Marshal(node.Capabilities)
	if err != nil {
		return NewStoreError("UpdateNode", "node", node.ReferenceID, "failed to marshal capabilities", err)
	}

	var lastHealthCheck *string
	if node.LastHealthCheck != nil {
		hc := node.LastHealthCheck.Format(time.RFC3339)
		lastHealthCheck = &hc
	}

	if node.SSHKeyID == 0 && node.SSHKeyRefID != "" {
		resolved, err := resolveRefID(ctx, exec, "ssh_keys", node.SSHKeyRefID)
		if err != nil {
			return NewStoreError("UpdateNode", "node", node.ReferenceID, "failed to resolve SSH key reference", err)
		}
		node.SSHKeyID = resolved
	}

	var sshKeyID *int
	if node.SSHKeyID != 0 {
		sshKeyID = &node.SSHKeyID
	}

	query := `
		UPDATE nodes SET
			name = :name,
			ssh_host = :ssh_host,
			ssh_port = :ssh_port,
			ssh_user = :ssh_user,
			ssh_key_id = :ssh_key_id,
			docker_socket = :docker_socket,
			status = :status,
			capabilities = :capabilities,
			capacity_cpu_cores = :capacity_cpu_cores,
			capacity_memory_mb = :capacity_memory_mb,
			capacity_disk_mb = :capacity_disk_mb,
			capacity_cpu_used = :capacity_cpu_used,
			capacity_memory_used_mb = :capacity_memory_used_mb,
			capacity_disk_used_mb = :capacity_disk_used_mb,
			location = :location,
			last_health_check = :last_health_check,
			error_message = :error_message,
			provider_type = :provider_type,
			provision_id = :provision_id,
			base_domain = :base_domain,
			updated_at = :updated_at
		WHERE reference_id = :reference_id`

	row := map[string]any{
		"reference_id":            node.ReferenceID,
		"name":                    node.Name,
		"creator_id":              node.CreatorID,
		"ssh_host":                node.SSHHost,
		"ssh_port":                node.SSHPort,
		"ssh_user":                node.SSHUser,
		"ssh_key_id":              sshKeyID,
		"docker_socket":           node.DockerSocket,
		"status":                  string(node.Status),
		"capabilities":            string(capabilities),
		"capacity_cpu_cores":      node.Capacity.CPUCores,
		"capacity_memory_mb":      node.Capacity.MemoryMB,
		"capacity_disk_mb":        node.Capacity.DiskMB,
		"capacity_cpu_used":       node.Capacity.CPUUsed,
		"capacity_memory_used_mb": node.Capacity.MemoryUsedMB,
		"capacity_disk_used_mb":   node.Capacity.DiskUsedMB,
		"location":                node.Location,
		"last_health_check":       lastHealthCheck,
		"error_message":           node.ErrorMessage,
		"provider_type":           node.ProviderType,
		"provision_id":            node.ProvisionID,
		"base_domain":             node.BaseDomain,
		"created_at":              node.CreatedAt.Format(time.RFC3339),
		"updated_at":              node.UpdatedAt.Format(time.RFC3339),
	}

	result, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		return NewStoreError("UpdateNode", "node", node.ReferenceID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateNode", "node", node.ReferenceID, "node not found", ErrNotFound)
	}

	return nil
}

// DeleteNode deletes a node by ID.
func (s *SQLiteStore) DeleteNode(ctx context.Context, id string) error {
	return deleteNode(ctx, s.db, id)
}

func deleteNode(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM nodes WHERE reference_id = ?`

	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return NewStoreError("DeleteNode", "node", id, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("DeleteNode", "node", id, "node not found", ErrNotFound)
	}

	return nil
}

// ListNodesByCreator lists all nodes for a creator.
func (s *SQLiteStore) ListNodesByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.Node, error) {
	return listNodesByCreator(ctx, s.db, creatorID, opts)
}

func listNodesByCreator(ctx context.Context, exec executor, creatorID int, opts ListOptions) ([]domain.Node, error) {
	opts = opts.Normalize()

	query := `SELECT ` + nodeSelectColumns + ` ` + nodeFromClause + `
		WHERE n.creator_id = ?
		ORDER BY n.created_at DESC
		LIMIT ? OFFSET ?`

	var rows []nodeRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListNodesByCreator", "node", "", err.Error(), err)
	}

	nodes := make([]domain.Node, 0, len(rows))
	for _, row := range rows {
		node, err := rowToNode(&row)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}

	return nodes, nil
}

// ListOnlineNodes returns all online nodes (for scheduling).
func (s *SQLiteStore) ListOnlineNodes(ctx context.Context) ([]domain.Node, error) {
	return listOnlineNodes(ctx, s.db)
}

func listOnlineNodes(ctx context.Context, exec executor) ([]domain.Node, error) {
	query := `SELECT ` + nodeSelectColumns + ` ` + nodeFromClause + `
		WHERE n.status = 'online'
		ORDER BY n.created_at ASC`

	var rows []nodeRow
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, NewStoreError("ListOnlineNodes", "node", "", err.Error(), err)
	}

	nodes := make([]domain.Node, 0, len(rows))
	for _, row := range rows {
		node, err := rowToNode(&row)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}

	return nodes, nil
}

// ListCheckableNodes lists all nodes that should be health-checked (not in maintenance mode).
func (s *SQLiteStore) ListCheckableNodes(ctx context.Context) ([]domain.Node, error) {
	return listCheckableNodes(ctx, s.db)
}

func listCheckableNodes(ctx context.Context, exec executor) ([]domain.Node, error) {
	query := `SELECT ` + nodeSelectColumns + ` ` + nodeFromClause + `
		WHERE n.status != 'maintenance'
		ORDER BY n.created_at ASC`

	var rows []nodeRow
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, NewStoreError("ListCheckableNodes", "node", "", err.Error(), err)
	}

	nodes := make([]domain.Node, 0, len(rows))
	for _, row := range rows {
		node, err := rowToNode(&row)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}

	return nodes, nil
}

// rowToNode converts a database row to a domain.Node.
func rowToNode(row *nodeRow) (*domain.Node, error) {
	var capabilities []string
	if err := json.Unmarshal([]byte(row.Capabilities), &capabilities); err != nil {
		return nil, NewStoreError("rowToNode", "node", row.ReferenceID, "failed to unmarshal capabilities", err)
	}

	createdAt := parseSQLiteTime(row.CreatedAt)
	updatedAt := parseSQLiteTime(row.UpdatedAt)

	var lastHealthCheck *time.Time
	if row.LastHealthCheck != nil && *row.LastHealthCheck != "" {
		hc := parseSQLiteTime(*row.LastHealthCheck)
		lastHealthCheck = &hc
	}

	var sshKeyID int
	if row.SSHKeyID != nil {
		sshKeyID = *row.SSHKeyID
	}

	var sshKeyRefID string
	if row.SSHKeyRefID != nil {
		sshKeyRefID = *row.SSHKeyRefID
	}

	return &domain.Node{
		ID:           row.ID,
		ReferenceID:  row.ReferenceID,
		Name:         row.Name,
		CreatorID:    row.CreatorID,
		SSHHost:      row.SSHHost,
		SSHPort:      row.SSHPort,
		SSHUser:      row.SSHUser,
		SSHKeyID:     sshKeyID,
		SSHKeyRefID:  sshKeyRefID,
		DockerSocket: row.DockerSocket,
		Status:       domain.NodeStatus(row.Status),
		Capabilities: capabilities,
		Capacity: domain.NodeCapacity{
			CPUCores:     row.CapacityCPUCores,
			MemoryMB:     row.CapacityMemoryMB,
			DiskMB:       row.CapacityDiskMB,
			CPUUsed:      row.CapacityCPUUsed,
			MemoryUsedMB: row.CapacityMemoryUsedMB,
			DiskUsedMB:   row.CapacityDiskUsedMB,
		},
		Location:        row.Location,
		LastHealthCheck: lastHealthCheck,
		ErrorMessage:    row.ErrorMessage,
		ProviderType:    row.ProviderType,
		ProvisionID:     row.ProvisionID,
		BaseDomain:      row.BaseDomain,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

// =============================================================================
// SSH Key Operations
// =============================================================================

// sshKeyRow represents an SSH key row in the database.
type sshKeyRow struct {
	ID                  int    `db:"id"`
	ReferenceID         string `db:"reference_id"`
	CreatorID           int    `db:"creator_id"`
	Name                string `db:"name"`
	PrivateKeyEncrypted []byte `db:"private_key_encrypted"`
	Fingerprint         string `db:"fingerprint"`
	CreatedAt           string `db:"created_at"`
}

// CreateSSHKey creates a new SSH key in the database.
func (s *SQLiteStore) CreateSSHKey(ctx context.Context, key *domain.SSHKey) error {
	return createSSHKey(ctx, s.db, key)
}

func createSSHKey(ctx context.Context, exec executor, key *domain.SSHKey) error {
	query := `
		INSERT INTO ssh_keys (reference_id, creator_id, name, private_key_encrypted, fingerprint, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	result, err := exec.ExecContext(ctx, query,
		key.ReferenceID,
		key.CreatorID,
		key.Name,
		key.PrivateKeyEncrypted,
		key.Fingerprint,
		key.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateSSHKey", "ssh_key", key.ReferenceID, "SSH key with this name already exists for creator", ErrDuplicateKey)
		}
		return NewStoreError("CreateSSHKey", "ssh_key", key.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateSSHKey", "ssh_key", key.ReferenceID, "failed to get last insert ID", err)
	}
	key.ID = int(lastID)

	return nil
}

// GetSSHKey retrieves an SSH key by ID.
func (s *SQLiteStore) GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error) {
	return getSSHKey(ctx, s.db, id)
}

func getSSHKey(ctx context.Context, exec executor, id string) (*domain.SSHKey, error) {
	var row sshKeyRow
	query := `SELECT id, reference_id, creator_id, name, private_key_encrypted, fingerprint, created_at FROM ssh_keys WHERE reference_id = ?`

	if err := exec.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetSSHKey", "ssh_key", id, "SSH key not found", ErrNotFound)
		}
		return nil, NewStoreError("GetSSHKey", "ssh_key", id, err.Error(), err)
	}

	createdAt := parseSQLiteTime(row.CreatedAt)

	return &domain.SSHKey{
		ID:                  row.ID,
		ReferenceID:         row.ReferenceID,
		CreatorID:           row.CreatorID,
		Name:                row.Name,
		PrivateKeyEncrypted: row.PrivateKeyEncrypted,
		Fingerprint:         row.Fingerprint,
		CreatedAt:           createdAt,
	}, nil
}

// DeleteSSHKey deletes an SSH key by ID.
func (s *SQLiteStore) DeleteSSHKey(ctx context.Context, id string) error {
	return deleteSSHKey(ctx, s.db, id)
}

func deleteSSHKey(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM ssh_keys WHERE reference_id = ?`

	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return NewStoreError("DeleteSSHKey", "ssh_key", id, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("DeleteSSHKey", "ssh_key", id, "SSH key not found", ErrNotFound)
	}

	return nil
}

// ListSSHKeysByCreator lists all SSH keys for a creator.
func (s *SQLiteStore) ListSSHKeysByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.SSHKey, error) {
	return listSSHKeysByCreator(ctx, s.db, creatorID, opts)
}

func listSSHKeysByCreator(ctx context.Context, exec executor, creatorID int, opts ListOptions) ([]domain.SSHKey, error) {
	opts = opts.Normalize()

	query := `
		SELECT id, reference_id, creator_id, name, private_key_encrypted, fingerprint, created_at
		FROM ssh_keys
		WHERE creator_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	var rows []sshKeyRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListSSHKeysByCreator", "ssh_key", "", err.Error(), err)
	}

	keys := make([]domain.SSHKey, 0, len(rows))
	for _, row := range rows {
		createdAt := parseSQLiteTime(row.CreatedAt)
		keys = append(keys, domain.SSHKey{
			ID:                  row.ID,
			ReferenceID:         row.ReferenceID,
			CreatorID:           row.CreatorID,
			Name:                row.Name,
			PrivateKeyEncrypted: row.PrivateKeyEncrypted,
			Fingerprint:         row.Fingerprint,
			CreatedAt:           createdAt,
		})
	}

	return keys, nil
}

// =============================================================================
// Cloud Credential Operations
// =============================================================================

// cloudCredentialRow represents a cloud credential row in the database.
type cloudCredentialRow struct {
	ID                   int    `db:"id"`
	ReferenceID          string `db:"reference_id"`
	CreatorID            int    `db:"creator_id"`
	Name                 string `db:"name"`
	Provider             string `db:"provider"`
	CredentialsEncrypted []byte `db:"credentials_encrypted"`
	DefaultRegion        string `db:"default_region"`
	CreatedAt            string `db:"created_at"`
	UpdatedAt            string `db:"updated_at"`
}

// CreateCloudCredential creates a new cloud credential in the database.
func (s *SQLiteStore) CreateCloudCredential(ctx context.Context, cred *domain.CloudCredential) error {
	return createCloudCredential(ctx, s.db, cred)
}

func createCloudCredential(ctx context.Context, exec executor, cred *domain.CloudCredential) error {
	query := `
		INSERT INTO cloud_credentials (reference_id, creator_id, name, provider, credentials_encrypted, default_region, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := exec.ExecContext(ctx, query,
		cred.ReferenceID,
		cred.CreatorID,
		cred.Name,
		string(cred.Provider),
		cred.CredentialsEncrypted,
		cred.DefaultRegion,
		cred.CreatedAt.Format(time.RFC3339),
		cred.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: cloud_credentials.reference_id") {
			return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ReferenceID, "cloud credential with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ReferenceID, "cloud credential with this name already exists for creator", ErrDuplicateKey)
		}
		return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ReferenceID, "failed to get last insert ID", err)
	}
	cred.ID = int(lastID)

	return nil
}

// GetCloudCredential retrieves a cloud credential by ID.
func (s *SQLiteStore) GetCloudCredential(ctx context.Context, id string) (*domain.CloudCredential, error) {
	return getCloudCredential(ctx, s.db, id)
}

func getCloudCredential(ctx context.Context, exec executor, id string) (*domain.CloudCredential, error) {
	var row cloudCredentialRow
	query := `
		SELECT id, reference_id, creator_id, name, provider, credentials_encrypted, default_region, created_at, updated_at
		FROM cloud_credentials WHERE reference_id = ?`

	if err := exec.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetCloudCredential", "cloud_credential", id, "cloud credential not found", ErrNotFound)
		}
		return nil, NewStoreError("GetCloudCredential", "cloud_credential", id, err.Error(), err)
	}

	return rowToCloudCredential(&row)
}

// DeleteCloudCredential deletes a cloud credential by ID.
func (s *SQLiteStore) DeleteCloudCredential(ctx context.Context, id string) error {
	return deleteCloudCredential(ctx, s.db, id)
}

func deleteCloudCredential(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM cloud_credentials WHERE reference_id = ?`

	result, err := exec.ExecContext(ctx, query, id)
	if err != nil {
		return NewStoreError("DeleteCloudCredential", "cloud_credential", id, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("DeleteCloudCredential", "cloud_credential", id, "cloud credential not found", ErrNotFound)
	}

	return nil
}

// ListCloudCredentialsByCreator lists all cloud credentials for a creator.
func (s *SQLiteStore) ListCloudCredentialsByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.CloudCredential, error) {
	return listCloudCredentialsByCreator(ctx, s.db, creatorID, opts)
}

func listCloudCredentialsByCreator(ctx context.Context, exec executor, creatorID int, opts ListOptions) ([]domain.CloudCredential, error) {
	opts = opts.Normalize()

	query := `
		SELECT id, reference_id, creator_id, name, provider, credentials_encrypted, default_region, created_at, updated_at
		FROM cloud_credentials
		WHERE creator_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	var rows []cloudCredentialRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListCloudCredentialsByCreator", "cloud_credential", "", err.Error(), err)
	}

	creds := make([]domain.CloudCredential, 0, len(rows))
	for _, row := range rows {
		cred, err := rowToCloudCredential(&row)
		if err != nil {
			return nil, err
		}
		creds = append(creds, *cred)
	}

	return creds, nil
}

// rowToCloudCredential converts a database row to a domain.CloudCredential.
func rowToCloudCredential(row *cloudCredentialRow) (*domain.CloudCredential, error) {
	createdAt := parseSQLiteTime(row.CreatedAt)
	updatedAt := parseSQLiteTime(row.UpdatedAt)

	return &domain.CloudCredential{
		ID:                   row.ID,
		ReferenceID:          row.ReferenceID,
		CreatorID:            row.CreatorID,
		Name:                 row.Name,
		Provider:             domain.ProviderType(row.Provider),
		CredentialsEncrypted: row.CredentialsEncrypted,
		DefaultRegion:        row.DefaultRegion,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}, nil
}

// =============================================================================
// Cloud Provision Operations
// =============================================================================

// cloudProvisionRow represents a cloud provision row in the database.
type cloudProvisionRow struct {
	ID                 int     `db:"id"`
	ReferenceID        string  `db:"reference_id"`
	CreatorID          int     `db:"creator_id"`
	CredentialID       int     `db:"credential_id"`
	CredentialRefID    *string `db:"credential_reference_id"` // populated via JOIN
	Provider           string  `db:"provider"`
	Status             string  `db:"status"`
	InstanceName       string  `db:"instance_name"`
	Region             string  `db:"region"`
	Size               string  `db:"size"`
	ProviderInstanceID string  `db:"provider_instance_id"`
	PublicIP           string  `db:"public_ip"`
	NodeID             string  `db:"node_id"`
	SSHKeyID           string  `db:"ssh_key_id"`
	CurrentStep        string  `db:"current_step"`
	ErrorMessage       string  `db:"error_message"`
	CreatedAt          string  `db:"created_at"`
	UpdatedAt          string  `db:"updated_at"`
	CompletedAt        *string `db:"completed_at"`
}

// provisionSelectColumns is the standard column list for provision queries with credential JOIN.
const provisionSelectColumns = `
	p.id, p.reference_id, p.creator_id, p.credential_id,
	cc.reference_id AS credential_reference_id,
	p.provider, p.status,
	p.instance_name, p.region, p.size,
	p.provider_instance_id, p.public_ip, p.node_id, p.ssh_key_id,
	p.current_step, p.error_message,
	p.created_at, p.updated_at, p.completed_at`

// provisionFromClause is the standard FROM clause for provision queries with credential JOIN.
const provisionFromClause = `FROM cloud_provisions p LEFT JOIN cloud_credentials cc ON cc.id = p.credential_id`

// CreateCloudProvision creates a new cloud provision in the database.
func (s *SQLiteStore) CreateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error {
	return createCloudProvision(ctx, s.db, prov)
}

func createCloudProvision(ctx context.Context, exec executor, prov *domain.CloudProvision) error {
	if prov.CredentialID == 0 && prov.CredentialRefID != "" {
		resolved, err := resolveRefID(ctx, exec, "cloud_credentials", prov.CredentialRefID)
		if err != nil {
			return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ReferenceID, "failed to resolve credential reference", err)
		}
		prov.CredentialID = resolved
	}

	var completedAt *string
	if prov.CompletedAt != nil {
		s := prov.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	query := `
		INSERT INTO cloud_provisions (
			reference_id, creator_id, credential_id, provider, status,
			instance_name, region, size,
			provider_instance_id, public_ip, node_id, ssh_key_id,
			current_step, error_message,
			created_at, updated_at, completed_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?
		)`

	result, err := exec.ExecContext(ctx, query,
		prov.ReferenceID,
		prov.CreatorID,
		prov.CredentialID,
		string(prov.Provider),
		string(prov.Status),
		prov.InstanceName,
		prov.Region,
		prov.Size,
		prov.ProviderInstanceID,
		prov.PublicIP,
		prov.NodeID,
		prov.SSHKeyID,
		prov.CurrentStep,
		prov.ErrorMessage,
		prov.CreatedAt.Format(time.RFC3339),
		prov.UpdatedAt.Format(time.RFC3339),
		completedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: cloud_provisions.reference_id") {
			return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ReferenceID, "cloud provision with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ReferenceID, "cloud provision with this name already exists", ErrDuplicateKey)
		}
		return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ReferenceID, err.Error(), err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ReferenceID, "failed to get last insert ID", err)
	}
	prov.ID = int(lastID)

	return nil
}

// GetCloudProvision retrieves a cloud provision by ID.
func (s *SQLiteStore) GetCloudProvision(ctx context.Context, id string) (*domain.CloudProvision, error) {
	return getCloudProvision(ctx, s.db, id)
}

func getCloudProvision(ctx context.Context, exec executor, id string) (*domain.CloudProvision, error) {
	var row cloudProvisionRow
	query := `SELECT ` + provisionSelectColumns + ` ` + provisionFromClause + ` WHERE p.reference_id = ?`

	if err := exec.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetCloudProvision", "cloud_provision", id, "cloud provision not found", ErrNotFound)
		}
		return nil, NewStoreError("GetCloudProvision", "cloud_provision", id, err.Error(), err)
	}

	return rowToCloudProvision(&row)
}

// UpdateCloudProvision updates an existing cloud provision.
func (s *SQLiteStore) UpdateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error {
	return updateCloudProvision(ctx, s.db, prov)
}

func updateCloudProvision(ctx context.Context, exec executor, prov *domain.CloudProvision) error {
	if prov.CredentialID == 0 && prov.CredentialRefID != "" {
		resolved, err := resolveRefID(ctx, exec, "cloud_credentials", prov.CredentialRefID)
		if err != nil {
			return NewStoreError("UpdateCloudProvision", "cloud_provision", prov.ReferenceID, "failed to resolve credential reference", err)
		}
		prov.CredentialID = resolved
	}

	var completedAt *string
	if prov.CompletedAt != nil {
		s := prov.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	query := `
		UPDATE cloud_provisions SET
			creator_id = :creator_id,
			credential_id = :credential_id,
			provider = :provider,
			status = :status,
			instance_name = :instance_name,
			region = :region,
			size = :size,
			provider_instance_id = :provider_instance_id,
			public_ip = :public_ip,
			node_id = :node_id,
			ssh_key_id = :ssh_key_id,
			current_step = :current_step,
			error_message = :error_message,
			updated_at = :updated_at,
			completed_at = :completed_at
		WHERE reference_id = :reference_id`

	row := map[string]any{
		"reference_id":         prov.ReferenceID,
		"creator_id":           prov.CreatorID,
		"credential_id":        prov.CredentialID,
		"provider":             string(prov.Provider),
		"status":               string(prov.Status),
		"instance_name":        prov.InstanceName,
		"region":               prov.Region,
		"size":                 prov.Size,
		"provider_instance_id": prov.ProviderInstanceID,
		"public_ip":            prov.PublicIP,
		"node_id":              prov.NodeID,
		"ssh_key_id":           prov.SSHKeyID,
		"current_step":         prov.CurrentStep,
		"error_message":        prov.ErrorMessage,
		"updated_at":           prov.UpdatedAt.Format(time.RFC3339),
		"completed_at":         completedAt,
	}

	result, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		return NewStoreError("UpdateCloudProvision", "cloud_provision", prov.ReferenceID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateCloudProvision", "cloud_provision", prov.ReferenceID, "cloud provision not found", ErrNotFound)
	}

	return nil
}

// ListCloudProvisionsByCreator lists all cloud provisions for a creator.
func (s *SQLiteStore) ListCloudProvisionsByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.CloudProvision, error) {
	return listCloudProvisionsByCreator(ctx, s.db, creatorID, opts)
}

func listCloudProvisionsByCreator(ctx context.Context, exec executor, creatorID int, opts ListOptions) ([]domain.CloudProvision, error) {
	opts = opts.Normalize()

	query := `SELECT ` + provisionSelectColumns + ` ` + provisionFromClause + `
		WHERE p.creator_id = ?
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?`

	var rows []cloudProvisionRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListCloudProvisionsByCreator", "cloud_provision", "", err.Error(), err)
	}

	provisions := make([]domain.CloudProvision, 0, len(rows))
	for _, row := range rows {
		prov, err := rowToCloudProvision(&row)
		if err != nil {
			return nil, err
		}
		provisions = append(provisions, *prov)
	}

	return provisions, nil
}

// ListActiveProvisions returns all provisions that are still in progress.
func (s *SQLiteStore) ListActiveProvisions(ctx context.Context) ([]domain.CloudProvision, error) {
	return listActiveProvisions(ctx, s.db)
}

func (s *SQLiteStore) ListCloudProvisionsByCredential(ctx context.Context, credentialID int) ([]domain.CloudProvision, error) {
	return listCloudProvisionsByCredential(ctx, s.db, credentialID)
}

func (s *SQLiteStore) ListDeploymentsByNode(ctx context.Context, nodeRefID string) ([]domain.Deployment, error) {
	return listDeploymentsByNode(ctx, s.db, nodeRefID)
}

func (s *SQLiteStore) ListNodesBySSHKey(ctx context.Context, sshKeyID int) ([]domain.Node, error) {
	return listNodesBySSHKey(ctx, s.db, sshKeyID)
}

func listActiveProvisions(ctx context.Context, exec executor) ([]domain.CloudProvision, error) {
	query := `SELECT ` + provisionSelectColumns + ` ` + provisionFromClause + `
		WHERE p.status IN ('pending', 'creating', 'configuring', 'destroying')
		ORDER BY p.created_at ASC`

	var rows []cloudProvisionRow
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, NewStoreError("ListActiveProvisions", "cloud_provision", "", err.Error(), err)
	}

	provisions := make([]domain.CloudProvision, 0, len(rows))
	for _, row := range rows {
		prov, err := rowToCloudProvision(&row)
		if err != nil {
			return nil, err
		}
		provisions = append(provisions, *prov)
	}

	return provisions, nil
}

// rowToCloudProvision converts a database row to a domain.CloudProvision.
func rowToCloudProvision(row *cloudProvisionRow) (*domain.CloudProvision, error) {
	createdAt := parseSQLiteTime(row.CreatedAt)
	updatedAt := parseSQLiteTime(row.UpdatedAt)

	var completedAt *time.Time
	if row.CompletedAt != nil && *row.CompletedAt != "" {
		t := parseSQLiteTime(*row.CompletedAt)
		completedAt = &t
	}

	var credentialRefID string
	if row.CredentialRefID != nil {
		credentialRefID = *row.CredentialRefID
	}

	return &domain.CloudProvision{
		ID:                 row.ID,
		ReferenceID:        row.ReferenceID,
		CreatorID:          row.CreatorID,
		CredentialID:       row.CredentialID,
		CredentialRefID:    credentialRefID,
		Provider:           domain.ProviderType(row.Provider),
		Status:             domain.ProvisionStatus(row.Status),
		InstanceName:       row.InstanceName,
		Region:             row.Region,
		Size:               row.Size,
		ProviderInstanceID: row.ProviderInstanceID,
		PublicIP:           row.PublicIP,
		NodeID:             row.NodeID,
		SSHKeyID:           row.SSHKeyID,
		CurrentStep:        row.CurrentStep,
		ErrorMessage:       row.ErrorMessage,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		CompletedAt:        completedAt,
	}, nil
}

// =============================================================================
// Dependency Lookup Functions (for safe deletion checks)
// =============================================================================

func listCloudProvisionsByCredential(ctx context.Context, exec executor, credentialID int) ([]domain.CloudProvision, error) {
	query := `SELECT ` + provisionSelectColumns + ` ` + provisionFromClause + `
		WHERE p.credential_id = ? AND p.status NOT IN ('destroyed')
		ORDER BY p.created_at DESC`

	var rows []cloudProvisionRow
	if err := exec.SelectContext(ctx, &rows, query, credentialID); err != nil {
		return nil, NewStoreError("ListCloudProvisionsByCredential", "cloud_provision", "", err.Error(), err)
	}

	provisions := make([]domain.CloudProvision, 0, len(rows))
	for _, row := range rows {
		prov, err := rowToCloudProvision(&row)
		if err != nil {
			return nil, err
		}
		provisions = append(provisions, *prov)
	}
	return provisions, nil
}

func listDeploymentsByNode(ctx context.Context, exec executor, nodeRefID string) ([]domain.Deployment, error) {
	query := `SELECT ` + deploymentSelectColumns + ` ` + deploymentFromClause + `
		WHERE d.node_id = ? AND d.status NOT IN ('deleted', 'stopped')
		ORDER BY d.created_at DESC`

	var rows []deploymentRow
	if err := exec.SelectContext(ctx, &rows, query, nodeRefID); err != nil {
		return nil, NewStoreError("ListDeploymentsByNode", "deployment", "", err.Error(), err)
	}

	deployments := make([]domain.Deployment, 0, len(rows))
	for _, row := range rows {
		dep, err := rowToDeployment(&row)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, *dep)
	}
	return deployments, nil
}

func listNodesBySSHKey(ctx context.Context, exec executor, sshKeyID int) ([]domain.Node, error) {
	query := `SELECT ` + nodeSelectColumns + ` ` + nodeFromClause + `
		WHERE n.ssh_key_id = ?
		ORDER BY n.created_at DESC`

	var rows []nodeRow
	if err := exec.SelectContext(ctx, &rows, query, sshKeyID); err != nil {
		return nil, NewStoreError("ListNodesBySSHKey", "node", "", err.Error(), err)
	}

	nodes := make([]domain.Node, 0, len(rows))
	for _, row := range rows {
		node, err := rowToNode(&row)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *node)
	}
	return nodes, nil
}
