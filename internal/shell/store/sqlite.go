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
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
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
// Template Operations
// =============================================================================

// templateRow represents a template row in the database.
type templateRow struct {
	ID                   string  `db:"id"`
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
	CreatorID            string  `db:"creator_id"`
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
	ID              string  `db:"id"`
	Name            string  `db:"name"`
	TemplateID      string  `db:"template_id"`
	TemplateVersion string  `db:"template_version"`
	CustomerID      string  `db:"customer_id"`
	NodeID          string  `db:"node_id"`
	Status          string  `db:"status"`
	Variables       *string `db:"variables"`
	Domains         *string `db:"domains"`
	Containers      *string `db:"containers"`
	ResourcesCPU    float64 `db:"resources_cpu_cores"`
	ResourcesMemory int64   `db:"resources_memory_mb"`
	ResourcesDisk   int64   `db:"resources_disk_mb"`
	ProxyPort       *int    `db:"proxy_port"`
	ErrorMessage    string  `db:"error_message"`
	CreatedAt       string  `db:"created_at"`
	UpdatedAt       string  `db:"updated_at"`
	StartedAt       *string `db:"started_at"`
	StoppedAt       *string `db:"stopped_at"`
}

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

func (s *SQLiteStore) ListDeploymentsByCustomer(ctx context.Context, customerID string, opts ListOptions) ([]domain.Deployment, error) {
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

func (s *txSQLiteStore) ListDeploymentsByCustomer(ctx context.Context, customerID string, opts ListOptions) ([]domain.Deployment, error) {
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

func (s *txSQLiteStore) ListNodesByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.Node, error) {
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

func (s *txSQLiteStore) ListSSHKeysByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.SSHKey, error) {
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

func (s *txSQLiteStore) ListCloudCredentialsByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.CloudCredential, error) {
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

func (s *txSQLiteStore) ListCloudProvisionsByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.CloudProvision, error) {
	return listCloudProvisionsByCreator(ctx, s.tx, creatorID, opts)
}

func (s *txSQLiteStore) ListActiveProvisions(ctx context.Context) ([]domain.CloudProvision, error) {
	return listActiveProvisions(ctx, s.tx)
}

// =============================================================================
// Shared Implementation Functions
// =============================================================================

func createTemplate(ctx context.Context, exec executor, template *domain.Template) error {
	// Serialize JSON fields
	variablesJSON, err := json.Marshal(template.Variables)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ID, "failed to serialize variables", ErrInvalidData)
	}
	configFilesJSON, err := json.Marshal(template.ConfigFiles)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ID, "failed to serialize config_files", ErrInvalidData)
	}
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ID, "failed to serialize tags", ErrInvalidData)
	}
	requiredCapabilitiesJSON, err := json.Marshal(template.RequiredCapabilities)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ID, "failed to serialize required_capabilities", ErrInvalidData)
	}

	query := `
		INSERT INTO templates (
			id, name, slug, description, version, compose_spec, variables, config_files,
			resources_cpu_cores, resources_memory_mb, resources_disk_mb,
			price_monthly_cents, category, tags, required_capabilities, published, creator_id,
			created_at, updated_at
		) VALUES (
			:id, :name, :slug, :description, :version, :compose_spec, :variables, :config_files,
			:resources_cpu_cores, :resources_memory_mb, :resources_disk_mb,
			:price_monthly_cents, :category, :tags, :required_capabilities, :published, :creator_id,
			:created_at, :updated_at
		)`

	row := map[string]any{
		"id":                    template.ID,
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
		"created_at":            template.CreatedAt.Format(time.RFC3339),
		"updated_at":            template.UpdatedAt.Format(time.RFC3339),
	}

	_, err = exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: templates.id") {
			return NewStoreError("CreateTemplate", "template", template.ID, "template with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed: templates.slug") {
			return NewStoreError("CreateTemplate", "template", template.ID, "template with this slug already exists", ErrDuplicateSlug)
		}
		return NewStoreError("CreateTemplate", "template", template.ID, err.Error(), err)
	}

	return nil
}

func getTemplate(ctx context.Context, exec executor, id string) (*domain.Template, error) {
	query := `SELECT * FROM templates WHERE id = ?`

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
	query := `SELECT * FROM templates WHERE slug = ?`

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
		return NewStoreError("UpdateTemplate", "template", template.ID, "failed to serialize variables", ErrInvalidData)
	}
	configFilesJSON, err := json.Marshal(template.ConfigFiles)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ID, "failed to serialize config_files", ErrInvalidData)
	}
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ID, "failed to serialize tags", ErrInvalidData)
	}
	requiredCapabilitiesJSON, err := json.Marshal(template.RequiredCapabilities)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ID, "failed to serialize required_capabilities", ErrInvalidData)
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
		WHERE id = :id`

	row := map[string]any{
		"id":                    template.ID,
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
		return NewStoreError("UpdateTemplate", "template", template.ID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateTemplate", "template", template.ID, "template not found", ErrNotFound)
	}

	return nil
}

func deleteTemplate(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM templates WHERE id = ?`

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
	query := `SELECT * FROM templates ORDER BY created_at DESC LIMIT ? OFFSET ?`

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
		return NewStoreError("CreateDeployment", "deployment", deployment.ID, "failed to serialize variables", ErrInvalidData)
	}
	domainsJSON, err := json.Marshal(deployment.Domains)
	if err != nil {
		return NewStoreError("CreateDeployment", "deployment", deployment.ID, "failed to serialize domains", ErrInvalidData)
	}
	containersJSON, err := json.Marshal(deployment.Containers)
	if err != nil {
		return NewStoreError("CreateDeployment", "deployment", deployment.ID, "failed to serialize containers", ErrInvalidData)
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

	var proxyPort *int
	if deployment.ProxyPort > 0 {
		proxyPort = &deployment.ProxyPort
	}

	query := `
		INSERT INTO deployments (
			id, name, template_id, template_version, customer_id, node_id,
			status, variables, domains, containers,
			resources_cpu_cores, resources_memory_mb, resources_disk_mb,
			proxy_port, error_message, created_at, updated_at, started_at, stopped_at
		) VALUES (
			:id, :name, :template_id, :template_version, :customer_id, :node_id,
			:status, :variables, :domains, :containers,
			:resources_cpu_cores, :resources_memory_mb, :resources_disk_mb,
			:proxy_port, :error_message, :created_at, :updated_at, :started_at, :stopped_at
		)`

	row := map[string]any{
		"id":                   deployment.ID,
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
		"created_at":           deployment.CreatedAt.Format(time.RFC3339),
		"updated_at":           deployment.UpdatedAt.Format(time.RFC3339),
		"started_at":           startedAt,
		"stopped_at":           stoppedAt,
	}

	_, err = exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: deployments.id") {
			return NewStoreError("CreateDeployment", "deployment", deployment.ID, "deployment with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return NewStoreError("CreateDeployment", "deployment", deployment.ID, "template not found", ErrForeignKey)
		}
		return NewStoreError("CreateDeployment", "deployment", deployment.ID, err.Error(), err)
	}

	return nil
}

func getDeployment(ctx context.Context, exec executor, id string) (*domain.Deployment, error) {
	query := `SELECT * FROM deployments WHERE id = ?`

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
		return NewStoreError("UpdateDeployment", "deployment", deployment.ID, "failed to serialize variables", ErrInvalidData)
	}
	domainsJSON, err := json.Marshal(deployment.Domains)
	if err != nil {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ID, "failed to serialize domains", ErrInvalidData)
	}
	containersJSON, err := json.Marshal(deployment.Containers)
	if err != nil {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ID, "failed to serialize containers", ErrInvalidData)
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
		WHERE id = :id`

	row := map[string]any{
		"id":                   deployment.ID,
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
		return NewStoreError("UpdateDeployment", "deployment", deployment.ID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateDeployment", "deployment", deployment.ID, "deployment not found", ErrNotFound)
	}

	return nil
}

func deleteDeployment(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM deployments WHERE id = ?`

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
	query := `SELECT * FROM deployments ORDER BY created_at DESC LIMIT ? OFFSET ?`

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
	query := `SELECT * FROM deployments WHERE template_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

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

func listDeploymentsByCustomer(ctx context.Context, exec executor, customerID string, opts ListOptions) ([]domain.Deployment, error) {
	opts = opts.Normalize()
	query := `SELECT * FROM deployments WHERE customer_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

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
		SELECT d.* FROM deployments d
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
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var variables []domain.Variable
	if row.Variables != nil && *row.Variables != "" && *row.Variables != "null" {
		if err := json.Unmarshal([]byte(*row.Variables), &variables); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ID, "failed to parse variables", ErrInvalidData)
		}
	}

	var configFiles []domain.ConfigFile
	if row.ConfigFiles != nil && *row.ConfigFiles != "" && *row.ConfigFiles != "null" {
		if err := json.Unmarshal([]byte(*row.ConfigFiles), &configFiles); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ID, "failed to parse config_files", ErrInvalidData)
		}
	}

	var tags []string
	if row.Tags != nil && *row.Tags != "" && *row.Tags != "null" {
		if err := json.Unmarshal([]byte(*row.Tags), &tags); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ID, "failed to parse tags", ErrInvalidData)
		}
	}

	var requiredCapabilities []string
	if row.RequiredCapabilities != nil && *row.RequiredCapabilities != "" && *row.RequiredCapabilities != "null" {
		if err := json.Unmarshal([]byte(*row.RequiredCapabilities), &requiredCapabilities); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ID, "failed to parse required_capabilities", ErrInvalidData)
		}
	}

	return &domain.Template{
		ID:          row.ID,
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
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var startedAt, stoppedAt *time.Time
	if row.StartedAt != nil && *row.StartedAt != "" {
		t, _ := time.Parse(time.RFC3339, *row.StartedAt)
		startedAt = &t
	}
	if row.StoppedAt != nil && *row.StoppedAt != "" {
		t, _ := time.Parse(time.RFC3339, *row.StoppedAt)
		stoppedAt = &t
	}

	var variables map[string]string
	if row.Variables != nil && *row.Variables != "" && *row.Variables != "null" {
		if err := json.Unmarshal([]byte(*row.Variables), &variables); err != nil {
			return nil, NewStoreError("rowToDeployment", "deployment", row.ID, "failed to parse variables", ErrInvalidData)
		}
	}

	var domains []domain.Domain
	if row.Domains != nil && *row.Domains != "" && *row.Domains != "null" {
		if err := json.Unmarshal([]byte(*row.Domains), &domains); err != nil {
			return nil, NewStoreError("rowToDeployment", "deployment", row.ID, "failed to parse domains", ErrInvalidData)
		}
	}

	var containers []domain.ContainerInfo
	if row.Containers != nil && *row.Containers != "" && *row.Containers != "null" {
		if err := json.Unmarshal([]byte(*row.Containers), &containers); err != nil {
			return nil, NewStoreError("rowToDeployment", "deployment", row.ID, "failed to parse containers", ErrInvalidData)
		}
	}

	var proxyPort int
	if row.ProxyPort != nil {
		proxyPort = *row.ProxyPort
	}

	return &domain.Deployment{
		ID:              row.ID,
		Name:            row.Name,
		TemplateID:      row.TemplateID,
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
	ID           string  `db:"id"`
	UserID       string  `db:"user_id"`
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
			return NewStoreError("CreateUsageEvent", "usage_event", event.ID, "failed to marshal metadata", ErrInvalidData)
		}
		s := string(data)
		metadataJSON = &s
	}

	row := usageEventRow{
		ID:           event.ID,
		UserID:       event.UserID,
		EventType:    string(event.EventType),
		ResourceID:   event.ResourceID,
		ResourceType: event.ResourceType,
		Quantity:     event.Quantity,
		Metadata:     metadataJSON,
		Timestamp:    event.Timestamp.Format(time.RFC3339),
		CreatedAt:    event.CreatedAt.Format(time.RFC3339),
	}

	query := `
		INSERT INTO usage_events (id, user_id, event_type, resource_id, resource_type, quantity, metadata, timestamp, created_at)
		VALUES (:id, :user_id, :event_type, :resource_id, :resource_type, :quantity, :metadata, :timestamp, :created_at)`

	_, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return NewStoreError("CreateUsageEvent", "usage_event", event.ID, "event already exists", ErrDuplicateID)
		}
		return NewStoreError("CreateUsageEvent", "usage_event", event.ID, err.Error(), err)
	}

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
		SELECT id, user_id, event_type, resource_id, resource_type, quantity, metadata, timestamp, reported_at, created_at
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
		WHERE id IN (%s)`, strings.Join(placeholders, ","))

	_, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return NewStoreError("MarkEventsReported", "usage_event", "", err.Error(), err)
	}

	return nil
}

// rowToUsageEvent converts a database row to a domain.MeterEvent.
func rowToUsageEvent(row *usageEventRow) (*domain.MeterEvent, error) {
	timestamp, _ := time.Parse(time.RFC3339, row.Timestamp)
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)

	var reportedAt *time.Time
	if row.ReportedAt != nil && *row.ReportedAt != "" {
		t, _ := time.Parse(time.RFC3339, *row.ReportedAt)
		reportedAt = &t
	}

	var metadata map[string]string
	if row.Metadata != nil && *row.Metadata != "" && *row.Metadata != "null" {
		if err := json.Unmarshal([]byte(*row.Metadata), &metadata); err != nil {
			return nil, NewStoreError("rowToUsageEvent", "usage_event", row.ID, "failed to parse metadata", ErrInvalidData)
		}
	}

	return &domain.MeterEvent{
		ID:           row.ID,
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
	ID           string `db:"id"`
	DeploymentID string `db:"deployment_id"`
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
		INSERT INTO container_events (id, deployment_id, type, container, message, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := exec.ExecContext(ctx, query,
		event.ID,
		event.DeploymentID,
		string(event.Type),
		event.Container,
		event.Message,
		event.Timestamp.Format(time.RFC3339),
		event.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return NewStoreError("CreateContainerEvent", "container_event", event.ID, "event already exists", ErrDuplicateID)
		}
		return NewStoreError("CreateContainerEvent", "container_event", event.ID, err.Error(), err)
	}

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
			SELECT id, deployment_id, type, container, message, timestamp, created_at
			FROM container_events
			WHERE deployment_id = ? AND type = ?
			ORDER BY timestamp DESC
			LIMIT ?`
		args = []any{deploymentID, *eventType, limit}
	} else {
		query = `
			SELECT id, deployment_id, type, container, message, timestamp, created_at
			FROM container_events
			WHERE deployment_id = ?
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
	timestamp, _ := time.Parse(time.RFC3339, row.Timestamp)
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)

	return &domain.ContainerEvent{
		ID:           row.ID,
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
	ID                   string  `db:"id"`
	Name                 string  `db:"name"`
	CreatorID            string  `db:"creator_id"`
	SSHHost              string  `db:"ssh_host"`
	SSHPort              int     `db:"ssh_port"`
	SSHUser              string  `db:"ssh_user"`
	SSHKeyID             *string `db:"ssh_key_id"`
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

// CreateNode creates a new node in the database.
func (s *SQLiteStore) CreateNode(ctx context.Context, node *domain.Node) error {
	return createNode(ctx, s.db, node)
}

func createNode(ctx context.Context, exec executor, node *domain.Node) error {
	capabilities, err := json.Marshal(node.Capabilities)
	if err != nil {
		return NewStoreError("CreateNode", "node", node.ID, "failed to marshal capabilities", err)
	}

	var lastHealthCheck *string
	if node.LastHealthCheck != nil {
		hc := node.LastHealthCheck.Format(time.RFC3339)
		lastHealthCheck = &hc
	}

	query := `
		INSERT INTO nodes (
			id, name, creator_id, ssh_host, ssh_port, ssh_user, ssh_key_id,
			docker_socket, status, capabilities,
			capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
			capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
			location, last_health_check, error_message,
			provider_type, provision_id, base_domain,
			created_at, updated_at
		) VALUES (
			:id, :name, :creator_id, :ssh_host, :ssh_port, :ssh_user, :ssh_key_id,
			:docker_socket, :status, :capabilities,
			:capacity_cpu_cores, :capacity_memory_mb, :capacity_disk_mb,
			:capacity_cpu_used, :capacity_memory_used_mb, :capacity_disk_used_mb,
			:location, :last_health_check, :error_message,
			:provider_type, :provision_id, :base_domain,
			:created_at, :updated_at
		)`

	var sshKeyID *string
	if node.SSHKeyID != "" {
		sshKeyID = &node.SSHKeyID
	}

	row := nodeRow{
		ID:                   node.ID,
		Name:                 node.Name,
		CreatorID:            node.CreatorID,
		SSHHost:              node.SSHHost,
		SSHPort:              node.SSHPort,
		SSHUser:              node.SSHUser,
		SSHKeyID:             sshKeyID,
		DockerSocket:         node.DockerSocket,
		Status:               string(node.Status),
		Capabilities:         string(capabilities),
		CapacityCPUCores:     node.Capacity.CPUCores,
		CapacityMemoryMB:     node.Capacity.MemoryMB,
		CapacityDiskMB:       node.Capacity.DiskMB,
		CapacityCPUUsed:      node.Capacity.CPUUsed,
		CapacityMemoryUsedMB: node.Capacity.MemoryUsedMB,
		CapacityDiskUsedMB:   node.Capacity.DiskUsedMB,
		Location:             node.Location,
		LastHealthCheck:      lastHealthCheck,
		ErrorMessage:         node.ErrorMessage,
		ProviderType:         node.ProviderType,
		ProvisionID:          node.ProvisionID,
		BaseDomain:           node.BaseDomain,
		CreatedAt:            node.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            node.UpdatedAt.Format(time.RFC3339),
	}

	_, err = exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: nodes.id") {
			return NewStoreError("CreateNode", "node", node.ID, "node with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateNode", "node", node.ID, "node with this name already exists for creator", ErrDuplicateKey)
		}
		return NewStoreError("CreateNode", "node", node.ID, err.Error(), err)
	}

	return nil
}

// GetNode retrieves a node by ID.
func (s *SQLiteStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	return getNode(ctx, s.db, id)
}

func getNode(ctx context.Context, exec executor, id string) (*domain.Node, error) {
	var row nodeRow
	query := `
		SELECT id, name, creator_id, ssh_host, ssh_port, ssh_user, ssh_key_id,
			docker_socket, status, capabilities,
			capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
			capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
			location, last_health_check, error_message,
			provider_type, provision_id, base_domain,
			created_at, updated_at
		FROM nodes WHERE id = ?`

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
		return NewStoreError("UpdateNode", "node", node.ID, "failed to marshal capabilities", err)
	}

	var lastHealthCheck *string
	if node.LastHealthCheck != nil {
		hc := node.LastHealthCheck.Format(time.RFC3339)
		lastHealthCheck = &hc
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
		WHERE id = :id`

	var sshKeyID *string
	if node.SSHKeyID != "" {
		sshKeyID = &node.SSHKeyID
	}

	row := nodeRow{
		ID:                   node.ID,
		Name:                 node.Name,
		CreatorID:            node.CreatorID,
		SSHHost:              node.SSHHost,
		SSHPort:              node.SSHPort,
		SSHUser:              node.SSHUser,
		SSHKeyID:             sshKeyID,
		DockerSocket:         node.DockerSocket,
		Status:               string(node.Status),
		Capabilities:         string(capabilities),
		CapacityCPUCores:     node.Capacity.CPUCores,
		CapacityMemoryMB:     node.Capacity.MemoryMB,
		CapacityDiskMB:       node.Capacity.DiskMB,
		CapacityCPUUsed:      node.Capacity.CPUUsed,
		CapacityMemoryUsedMB: node.Capacity.MemoryUsedMB,
		CapacityDiskUsedMB:   node.Capacity.DiskUsedMB,
		Location:             node.Location,
		LastHealthCheck:      lastHealthCheck,
		ErrorMessage:         node.ErrorMessage,
		ProviderType:         node.ProviderType,
		ProvisionID:          node.ProvisionID,
		BaseDomain:           node.BaseDomain,
		CreatedAt:            node.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            node.UpdatedAt.Format(time.RFC3339),
	}

	result, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		return NewStoreError("UpdateNode", "node", node.ID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateNode", "node", node.ID, "node not found", ErrNotFound)
	}

	return nil
}

// DeleteNode deletes a node by ID.
func (s *SQLiteStore) DeleteNode(ctx context.Context, id string) error {
	return deleteNode(ctx, s.db, id)
}

func deleteNode(ctx context.Context, exec executor, id string) error {
	query := `DELETE FROM nodes WHERE id = ?`

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
func (s *SQLiteStore) ListNodesByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.Node, error) {
	return listNodesByCreator(ctx, s.db, creatorID, opts)
}

func listNodesByCreator(ctx context.Context, exec executor, creatorID string, opts ListOptions) ([]domain.Node, error) {
	opts = opts.Normalize()

	query := `
		SELECT id, name, creator_id, ssh_host, ssh_port, ssh_user, ssh_key_id,
			docker_socket, status, capabilities,
			capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
			capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
			location, last_health_check, error_message,
			provider_type, provision_id, base_domain,
			created_at, updated_at
		FROM nodes
		WHERE creator_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	var rows []nodeRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListNodesByCreator", "node", creatorID, err.Error(), err)
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
	query := `
		SELECT id, name, creator_id, ssh_host, ssh_port, ssh_user, ssh_key_id,
			docker_socket, status, capabilities,
			capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
			capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
			location, last_health_check, error_message,
			provider_type, provision_id, base_domain,
			created_at, updated_at
		FROM nodes
		WHERE status = 'online'
		ORDER BY created_at ASC`

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
	query := `
		SELECT id, name, creator_id, ssh_host, ssh_port, ssh_user, ssh_key_id,
			docker_socket, status, capabilities,
			capacity_cpu_cores, capacity_memory_mb, capacity_disk_mb,
			capacity_cpu_used, capacity_memory_used_mb, capacity_disk_used_mb,
			location, last_health_check, error_message,
			provider_type, provision_id, base_domain,
			created_at, updated_at
		FROM nodes
		WHERE status != 'maintenance'
		ORDER BY created_at ASC`

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
		return nil, NewStoreError("rowToNode", "node", row.ID, "failed to unmarshal capabilities", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var lastHealthCheck *time.Time
	if row.LastHealthCheck != nil && *row.LastHealthCheck != "" {
		hc, _ := time.Parse(time.RFC3339, *row.LastHealthCheck)
		lastHealthCheck = &hc
	}

	sshKeyID := ""
	if row.SSHKeyID != nil {
		sshKeyID = *row.SSHKeyID
	}

	return &domain.Node{
		ID:           row.ID,
		Name:         row.Name,
		CreatorID:    row.CreatorID,
		SSHHost:      row.SSHHost,
		SSHPort:      row.SSHPort,
		SSHUser:      row.SSHUser,
		SSHKeyID:     sshKeyID,
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
	ID                  string `db:"id"`
	CreatorID           string `db:"creator_id"`
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
		INSERT INTO ssh_keys (id, creator_id, name, private_key_encrypted, fingerprint, created_at)
		VALUES (:id, :creator_id, :name, :private_key_encrypted, :fingerprint, :created_at)`

	row := sshKeyRow{
		ID:                  key.ID,
		CreatorID:           key.CreatorID,
		Name:                key.Name,
		PrivateKeyEncrypted: key.PrivateKeyEncrypted,
		Fingerprint:         key.Fingerprint,
		CreatedAt:           key.CreatedAt.Format(time.RFC3339),
	}

	_, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateSSHKey", "ssh_key", key.ID, "SSH key with this name already exists for creator", ErrDuplicateKey)
		}
		return NewStoreError("CreateSSHKey", "ssh_key", key.ID, err.Error(), err)
	}

	return nil
}

// GetSSHKey retrieves an SSH key by ID.
func (s *SQLiteStore) GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error) {
	return getSSHKey(ctx, s.db, id)
}

func getSSHKey(ctx context.Context, exec executor, id string) (*domain.SSHKey, error) {
	var row sshKeyRow
	query := `SELECT id, creator_id, name, private_key_encrypted, fingerprint, created_at FROM ssh_keys WHERE id = ?`

	if err := exec.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewStoreError("GetSSHKey", "ssh_key", id, "SSH key not found", ErrNotFound)
		}
		return nil, NewStoreError("GetSSHKey", "ssh_key", id, err.Error(), err)
	}

	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)

	return &domain.SSHKey{
		ID:                  row.ID,
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
	query := `DELETE FROM ssh_keys WHERE id = ?`

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
func (s *SQLiteStore) ListSSHKeysByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.SSHKey, error) {
	return listSSHKeysByCreator(ctx, s.db, creatorID, opts)
}

func listSSHKeysByCreator(ctx context.Context, exec executor, creatorID string, opts ListOptions) ([]domain.SSHKey, error) {
	opts = opts.Normalize()

	query := `
		SELECT id, creator_id, name, private_key_encrypted, fingerprint, created_at
		FROM ssh_keys
		WHERE creator_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	var rows []sshKeyRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListSSHKeysByCreator", "ssh_key", creatorID, err.Error(), err)
	}

	keys := make([]domain.SSHKey, 0, len(rows))
	for _, row := range rows {
		createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
		keys = append(keys, domain.SSHKey{
			ID:                  row.ID,
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
	ID                   string `db:"id"`
	CreatorID            string `db:"creator_id"`
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
		INSERT INTO cloud_credentials (id, creator_id, name, provider, credentials_encrypted, default_region, created_at, updated_at)
		VALUES (:id, :creator_id, :name, :provider, :credentials_encrypted, :default_region, :created_at, :updated_at)`

	row := cloudCredentialRow{
		ID:                   cred.ID,
		CreatorID:            cred.CreatorID,
		Name:                 cred.Name,
		Provider:             string(cred.Provider),
		CredentialsEncrypted: cred.CredentialsEncrypted,
		DefaultRegion:        cred.DefaultRegion,
		CreatedAt:            cred.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            cred.UpdatedAt.Format(time.RFC3339),
	}

	_, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: cloud_credentials.id") {
			return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ID, "cloud credential with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ID, "cloud credential with this name already exists for creator", ErrDuplicateKey)
		}
		return NewStoreError("CreateCloudCredential", "cloud_credential", cred.ID, err.Error(), err)
	}

	return nil
}

// GetCloudCredential retrieves a cloud credential by ID.
func (s *SQLiteStore) GetCloudCredential(ctx context.Context, id string) (*domain.CloudCredential, error) {
	return getCloudCredential(ctx, s.db, id)
}

func getCloudCredential(ctx context.Context, exec executor, id string) (*domain.CloudCredential, error) {
	var row cloudCredentialRow
	query := `
		SELECT id, creator_id, name, provider, credentials_encrypted, default_region, created_at, updated_at
		FROM cloud_credentials WHERE id = ?`

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
	query := `DELETE FROM cloud_credentials WHERE id = ?`

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
func (s *SQLiteStore) ListCloudCredentialsByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.CloudCredential, error) {
	return listCloudCredentialsByCreator(ctx, s.db, creatorID, opts)
}

func listCloudCredentialsByCreator(ctx context.Context, exec executor, creatorID string, opts ListOptions) ([]domain.CloudCredential, error) {
	opts = opts.Normalize()

	query := `
		SELECT id, creator_id, name, provider, credentials_encrypted, default_region, created_at, updated_at
		FROM cloud_credentials
		WHERE creator_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	var rows []cloudCredentialRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListCloudCredentialsByCreator", "cloud_credential", creatorID, err.Error(), err)
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
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	return &domain.CloudCredential{
		ID:                   row.ID,
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
	ID                 string  `db:"id"`
	CreatorID          string  `db:"creator_id"`
	CredentialID       string  `db:"credential_id"`
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

// CreateCloudProvision creates a new cloud provision in the database.
func (s *SQLiteStore) CreateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error {
	return createCloudProvision(ctx, s.db, prov)
}

func createCloudProvision(ctx context.Context, exec executor, prov *domain.CloudProvision) error {
	var completedAt *string
	if prov.CompletedAt != nil {
		s := prov.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	query := `
		INSERT INTO cloud_provisions (
			id, creator_id, credential_id, provider, status,
			instance_name, region, size,
			provider_instance_id, public_ip, node_id, ssh_key_id,
			current_step, error_message,
			created_at, updated_at, completed_at
		) VALUES (
			:id, :creator_id, :credential_id, :provider, :status,
			:instance_name, :region, :size,
			:provider_instance_id, :public_ip, :node_id, :ssh_key_id,
			:current_step, :error_message,
			:created_at, :updated_at, :completed_at
		)`

	row := cloudProvisionRow{
		ID:                 prov.ID,
		CreatorID:          prov.CreatorID,
		CredentialID:       prov.CredentialID,
		Provider:           string(prov.Provider),
		Status:             string(prov.Status),
		InstanceName:       prov.InstanceName,
		Region:             prov.Region,
		Size:               prov.Size,
		ProviderInstanceID: prov.ProviderInstanceID,
		PublicIP:           prov.PublicIP,
		NodeID:             prov.NodeID,
		SSHKeyID:           prov.SSHKeyID,
		CurrentStep:        prov.CurrentStep,
		ErrorMessage:       prov.ErrorMessage,
		CreatedAt:          prov.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          prov.UpdatedAt.Format(time.RFC3339),
		CompletedAt:        completedAt,
	}

	_, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: cloud_provisions.id") {
			return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ID, "cloud provision with this ID already exists", ErrDuplicateID)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ID, "cloud provision with this name already exists", ErrDuplicateKey)
		}
		return NewStoreError("CreateCloudProvision", "cloud_provision", prov.ID, err.Error(), err)
	}

	return nil
}

// GetCloudProvision retrieves a cloud provision by ID.
func (s *SQLiteStore) GetCloudProvision(ctx context.Context, id string) (*domain.CloudProvision, error) {
	return getCloudProvision(ctx, s.db, id)
}

func getCloudProvision(ctx context.Context, exec executor, id string) (*domain.CloudProvision, error) {
	var row cloudProvisionRow
	query := `
		SELECT id, creator_id, credential_id, provider, status,
			instance_name, region, size,
			provider_instance_id, public_ip, node_id, ssh_key_id,
			current_step, error_message,
			created_at, updated_at, completed_at
		FROM cloud_provisions WHERE id = ?`

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
		WHERE id = :id`

	row := cloudProvisionRow{
		ID:                 prov.ID,
		CreatorID:          prov.CreatorID,
		CredentialID:       prov.CredentialID,
		Provider:           string(prov.Provider),
		Status:             string(prov.Status),
		InstanceName:       prov.InstanceName,
		Region:             prov.Region,
		Size:               prov.Size,
		ProviderInstanceID: prov.ProviderInstanceID,
		PublicIP:           prov.PublicIP,
		NodeID:             prov.NodeID,
		SSHKeyID:           prov.SSHKeyID,
		CurrentStep:        prov.CurrentStep,
		ErrorMessage:       prov.ErrorMessage,
		CreatedAt:          prov.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          prov.UpdatedAt.Format(time.RFC3339),
		CompletedAt:        completedAt,
	}

	result, err := exec.NamedExecContext(ctx, query, row)
	if err != nil {
		return NewStoreError("UpdateCloudProvision", "cloud_provision", prov.ID, err.Error(), err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return NewStoreError("UpdateCloudProvision", "cloud_provision", prov.ID, "cloud provision not found", ErrNotFound)
	}

	return nil
}

// ListCloudProvisionsByCreator lists all cloud provisions for a creator.
func (s *SQLiteStore) ListCloudProvisionsByCreator(ctx context.Context, creatorID string, opts ListOptions) ([]domain.CloudProvision, error) {
	return listCloudProvisionsByCreator(ctx, s.db, creatorID, opts)
}

func listCloudProvisionsByCreator(ctx context.Context, exec executor, creatorID string, opts ListOptions) ([]domain.CloudProvision, error) {
	opts = opts.Normalize()

	query := `
		SELECT id, creator_id, credential_id, provider, status,
			instance_name, region, size,
			provider_instance_id, public_ip, node_id, ssh_key_id,
			current_step, error_message,
			created_at, updated_at, completed_at
		FROM cloud_provisions
		WHERE creator_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`

	var rows []cloudProvisionRow
	if err := exec.SelectContext(ctx, &rows, query, creatorID, opts.Limit, opts.Offset); err != nil {
		return nil, NewStoreError("ListCloudProvisionsByCreator", "cloud_provision", creatorID, err.Error(), err)
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

func listActiveProvisions(ctx context.Context, exec executor) ([]domain.CloudProvision, error) {
	query := `
		SELECT id, creator_id, credential_id, provider, status,
			instance_name, region, size,
			provider_instance_id, public_ip, node_id, ssh_key_id,
			current_step, error_message,
			created_at, updated_at, completed_at
		FROM cloud_provisions
		WHERE status IN ('pending', 'creating', 'configuring')
		ORDER BY created_at ASC`

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
	createdAt, _ := time.Parse(time.RFC3339, row.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, row.UpdatedAt)

	var completedAt *time.Time
	if row.CompletedAt != nil && *row.CompletedAt != "" {
		t, _ := time.Parse(time.RFC3339, *row.CompletedAt)
		completedAt = &t
	}

	return &domain.CloudProvision{
		ID:                 row.ID,
		CreatorID:          row.CreatorID,
		CredentialID:       row.CredentialID,
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
