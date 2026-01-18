# F002: SQLite Store

## User Story

As a **Hoster system**, I need to persist templates and deployments so that data survives restarts and can be queried efficiently.

## Overview

This feature provides a SQLite-based persistence layer for Template and Deployment entities. It follows the Imperative Shell pattern (I/O lives in `internal/shell/`), using `sqlx` for database access and `golang-migrate` for schema migrations.

## Acceptance Criteria

- [ ] Store interface with CRUD operations for Templates and Deployments
- [ ] SQLite implementation with proper connection management
- [ ] Database migrations for schema setup
- [ ] JSON serialization for complex fields (Variables, Containers)
- [ ] Transaction support with commit/rollback
- [ ] Pagination for list operations
- [ ] Context cancellation support
- [ ] Proper error types for all failure modes
- [ ] ~35 tests with real SQLite (no mocks)

## File Structure

```
internal/shell/store/
├── store.go         # Store interface
├── sqlite.go        # SQLiteStore implementation
├── errors.go        # Error types
├── migrations/
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
└── sqlite_test.go   # ~35 tests (real SQLite)
```

## Interface

```go
package store

import (
    "context"
    "github.com/artpar/hoster/internal/core/domain"
)

// Store defines the persistence interface for Hoster entities.
type Store interface {
    // Template operations
    CreateTemplate(ctx context.Context, template *domain.Template) error
    GetTemplate(ctx context.Context, id string) (*domain.Template, error)
    GetTemplateBySlug(ctx context.Context, slug string) (*domain.Template, error)
    UpdateTemplate(ctx context.Context, template *domain.Template) error
    DeleteTemplate(ctx context.Context, id string) error
    ListTemplates(ctx context.Context, opts ListOptions) ([]domain.Template, error)

    // Deployment operations
    CreateDeployment(ctx context.Context, deployment *domain.Deployment) error
    GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
    UpdateDeployment(ctx context.Context, deployment *domain.Deployment) error
    DeleteDeployment(ctx context.Context, id string) error
    ListDeployments(ctx context.Context, opts ListOptions) ([]domain.Deployment, error)
    ListDeploymentsByTemplate(ctx context.Context, templateID string, opts ListOptions) ([]domain.Deployment, error)
    ListDeploymentsByCustomer(ctx context.Context, customerID string, opts ListOptions) ([]domain.Deployment, error)

    // Transaction support
    WithTx(ctx context.Context, fn func(Store) error) error

    // Lifecycle
    Close() error
}

// ListOptions defines pagination and filtering options.
type ListOptions struct {
    Limit  int
    Offset int
}
```

## Schema

### Templates Table

```sql
CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    version TEXT NOT NULL,
    compose_spec TEXT NOT NULL,
    variables TEXT,  -- JSON array
    resources_cpu_cores REAL NOT NULL DEFAULT 0,
    resources_memory_mb INTEGER NOT NULL DEFAULT 0,
    resources_disk_mb INTEGER NOT NULL DEFAULT 0,
    price_monthly_cents INTEGER DEFAULT 0,
    published INTEGER DEFAULT 0,  -- 0 = false, 1 = true
    creator_id TEXT NOT NULL,
    created_at TEXT NOT NULL,     -- RFC3339 timestamp
    updated_at TEXT NOT NULL      -- RFC3339 timestamp
);

CREATE INDEX idx_templates_slug ON templates(slug);
CREATE INDEX idx_templates_creator ON templates(creator_id);
CREATE INDEX idx_templates_published ON templates(published);
```

### Deployments Table

```sql
CREATE TABLE deployments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    status TEXT NOT NULL,         -- pending, scheduled, starting, running, etc.
    variables TEXT,               -- JSON object
    containers TEXT,              -- JSON array
    created_at TEXT NOT NULL,     -- RFC3339 timestamp
    updated_at TEXT NOT NULL,     -- RFC3339 timestamp
    FOREIGN KEY (template_id) REFERENCES templates(id)
);

CREATE INDEX idx_deployments_template ON deployments(template_id);
CREATE INDEX idx_deployments_customer ON deployments(customer_id);
CREATE INDEX idx_deployments_status ON deployments(status);
```

## Error Types

| Error | Condition |
|-------|-----------|
| ErrNotFound | Entity with given ID does not exist |
| ErrDuplicateID | Entity with same ID already exists |
| ErrDuplicateSlug | Template with same slug already exists |
| ErrForeignKey | Referenced template does not exist |
| ErrConnectionFailed | Failed to connect to database |
| ErrMigrationFailed | Database migration failed |
| ErrInvalidData | JSON serialization/deserialization failed |
| ErrTxFailed | Transaction commit/rollback failed |

## Constructor

```go
// NewSQLiteStore creates a new SQLite-backed store.
// dsn is the database connection string (e.g., "./data/hoster.db" or ":memory:")
// Runs migrations automatically on startup.
func NewSQLiteStore(dsn string) (*SQLiteStore, error)
```

## JSON Serialization

### Variables

Template variables are stored as JSON array:
```json
["DB_PASSWORD", "API_KEY", "SECRET_TOKEN"]
```

Deployment variables are stored as JSON object:
```json
{"DB_PASSWORD": "secret123", "API_KEY": "abc123"}
```

### Containers

Deployment containers are stored as JSON array:
```json
[
  {"id": "abc123", "name": "web", "status": "running"},
  {"id": "def456", "name": "db", "status": "running"}
]
```

## Implementation Notes

1. Use `sqlx` for ergonomic SQL operations
2. Use `golang-migrate/migrate/v4` for schema migrations
3. Store timestamps in RFC3339 format
4. Use `encoding/json` for JSON serialization
5. Support `:memory:` DSN for testing
6. Wrap SQLite errors with appropriate custom errors
7. Use prepared statements where beneficial
8. Transaction must use same connection via `sqlx.Tx`

## Test Categories (~35 tests)

### Template CRUD (~8 tests)
- CreateTemplate success
- CreateTemplate duplicate ID error
- CreateTemplate duplicate slug error
- GetTemplate success
- GetTemplate not found
- GetTemplateBySlug success
- UpdateTemplate success
- DeleteTemplate success

### Deployment CRUD (~8 tests)
- CreateDeployment success
- CreateDeployment foreign key error
- GetDeployment success
- GetDeployment not found
- UpdateDeployment success
- DeleteDeployment success
- ListDeploymentsByTemplate success
- ListDeploymentsByCustomer success

### JSON Serialization (~4 tests)
- Variables serialization/deserialization
- Containers serialization/deserialization
- Empty arrays handled correctly
- Null/nil handled correctly

### List Operations (~5 tests)
- ListTemplates with pagination
- ListDeployments with pagination
- Empty result set
- Offset beyond data
- Default limit applied

### Transactions (~4 tests)
- Transaction commit success
- Transaction rollback on error
- Nested transaction not allowed
- Context cancellation during transaction

### Error Handling (~4 tests)
- Connection failed error
- Context deadline exceeded
- Invalid JSON data
- Schema migration failure

### Edge Cases (~2 tests)
- Unicode in text fields
- Very long compose spec

## Dependencies

```go
"github.com/jmoiron/sqlx"
"github.com/mattn/go-sqlite3"
"github.com/golang-migrate/migrate/v4"
"github.com/golang-migrate/migrate/v4/database/sqlite3"
"github.com/golang-migrate/migrate/v4/source/iofs"
```

## Not Supported (By Design)

- Connection pooling (SQLite is single-writer)
- Read replicas
- Async writes
- Full-text search
- Complex queries beyond CRUD
