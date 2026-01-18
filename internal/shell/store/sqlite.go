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
	ID              string  `db:"id"`
	Name            string  `db:"name"`
	Slug            string  `db:"slug"`
	Description     string  `db:"description"`
	Version         string  `db:"version"`
	ComposeSpec     string  `db:"compose_spec"`
	Variables       *string `db:"variables"`
	ResourcesCPU    float64 `db:"resources_cpu_cores"`
	ResourcesMemory int64   `db:"resources_memory_mb"`
	ResourcesDisk   int64   `db:"resources_disk_mb"`
	PriceMonthly    int64   `db:"price_monthly_cents"`
	Category        string  `db:"category"`
	Tags            *string `db:"tags"`
	Published       bool    `db:"published"`
	CreatorID       string  `db:"creator_id"`
	CreatedAt       string  `db:"created_at"`
	UpdatedAt       string  `db:"updated_at"`
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

func (s *txSQLiteStore) WithTx(ctx context.Context, fn func(Store) error) error {
	// Already in a transaction, just run the function
	return fn(s)
}

func (s *txSQLiteStore) Close() error {
	// No-op for tx store
	return nil
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
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return NewStoreError("CreateTemplate", "template", template.ID, "failed to serialize tags", ErrInvalidData)
	}

	query := `
		INSERT INTO templates (
			id, name, slug, description, version, compose_spec, variables,
			resources_cpu_cores, resources_memory_mb, resources_disk_mb,
			price_monthly_cents, category, tags, published, creator_id,
			created_at, updated_at
		) VALUES (
			:id, :name, :slug, :description, :version, :compose_spec, :variables,
			:resources_cpu_cores, :resources_memory_mb, :resources_disk_mb,
			:price_monthly_cents, :category, :tags, :published, :creator_id,
			:created_at, :updated_at
		)`

	row := map[string]any{
		"id":                   template.ID,
		"name":                 template.Name,
		"slug":                 template.Slug,
		"description":          template.Description,
		"version":              template.Version,
		"compose_spec":         template.ComposeSpec,
		"variables":            string(variablesJSON),
		"resources_cpu_cores":  template.ResourceRequirements.CPUCores,
		"resources_memory_mb":  template.ResourceRequirements.MemoryMB,
		"resources_disk_mb":    template.ResourceRequirements.DiskMB,
		"price_monthly_cents":  template.PriceMonthly,
		"category":             template.Category,
		"tags":                 string(tagsJSON),
		"published":            template.Published,
		"creator_id":           template.CreatorID,
		"created_at":           template.CreatedAt.Format(time.RFC3339),
		"updated_at":           template.UpdatedAt.Format(time.RFC3339),
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
	tagsJSON, err := json.Marshal(template.Tags)
	if err != nil {
		return NewStoreError("UpdateTemplate", "template", template.ID, "failed to serialize tags", ErrInvalidData)
	}

	query := `
		UPDATE templates SET
			name = :name,
			slug = :slug,
			description = :description,
			version = :version,
			compose_spec = :compose_spec,
			variables = :variables,
			resources_cpu_cores = :resources_cpu_cores,
			resources_memory_mb = :resources_memory_mb,
			resources_disk_mb = :resources_disk_mb,
			price_monthly_cents = :price_monthly_cents,
			category = :category,
			tags = :tags,
			published = :published,
			creator_id = :creator_id,
			updated_at = :updated_at
		WHERE id = :id`

	row := map[string]any{
		"id":                   template.ID,
		"name":                 template.Name,
		"slug":                 template.Slug,
		"description":          template.Description,
		"version":              template.Version,
		"compose_spec":         template.ComposeSpec,
		"variables":            string(variablesJSON),
		"resources_cpu_cores":  template.ResourceRequirements.CPUCores,
		"resources_memory_mb":  template.ResourceRequirements.MemoryMB,
		"resources_disk_mb":    template.ResourceRequirements.DiskMB,
		"price_monthly_cents":  template.PriceMonthly,
		"category":             template.Category,
		"tags":                 string(tagsJSON),
		"published":            template.Published,
		"creator_id":           template.CreatorID,
		"updated_at":           template.UpdatedAt.Format(time.RFC3339),
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

	query := `
		INSERT INTO deployments (
			id, name, template_id, template_version, customer_id, node_id,
			status, variables, domains, containers,
			resources_cpu_cores, resources_memory_mb, resources_disk_mb,
			error_message, created_at, updated_at, started_at, stopped_at
		) VALUES (
			:id, :name, :template_id, :template_version, :customer_id, :node_id,
			:status, :variables, :domains, :containers,
			:resources_cpu_cores, :resources_memory_mb, :resources_disk_mb,
			:error_message, :created_at, :updated_at, :started_at, :stopped_at
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

	var tags []string
	if row.Tags != nil && *row.Tags != "" && *row.Tags != "null" {
		if err := json.Unmarshal([]byte(*row.Tags), &tags); err != nil {
			return nil, NewStoreError("rowToTemplate", "template", row.ID, "failed to parse tags", ErrInvalidData)
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
		ResourceRequirements: domain.Resources{
			CPUCores: row.ResourcesCPU,
			MemoryMB: row.ResourcesMemory,
			DiskMB:   row.ResourcesDisk,
		},
		PriceMonthly: row.PriceMonthly,
		Category:     row.Category,
		Tags:         tags,
		Published:    row.Published,
		CreatorID:    row.CreatorID,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
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
		ErrorMessage: row.ErrorMessage,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		StartedAt:    startedAt,
		StoppedAt:    stoppedAt,
	}, nil
}
