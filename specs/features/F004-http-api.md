# F004: HTTP API Specification

## Overview

REST API for managing templates and deployments. Provides endpoints for CRUD operations on templates, deployment lifecycle management, and health checks.

## Dependencies

- F002: SQLite Store (for persistence)
- F003: Docker Client (for deployment operations)

## Package

`internal/shell/api/`

## Files

```
internal/shell/api/
├── handler.go           # Handler struct, Routes(), middleware
├── template_handlers.go # Template CRUD endpoints
├── deployment_handlers.go # Deployment lifecycle endpoints
├── types.go             # Request/Response types
├── helpers.go           # JSON helpers, error responses
└── handler_test.go      # ~50 tests
```

---

## API Design

### Base URL

```
/api/v1
```

### Content Type

All requests and responses use `application/json`.

### Error Response Format

```json
{
  "error": "human readable message",
  "code": "machine_readable_code"
}
```

Error Codes:
- `validation_error` - Invalid request data
- `template_not_found` - Template doesn't exist
- `deployment_not_found` - Deployment doesn't exist
- `template_not_published` - Template is not published
- `invalid_transition` - State transition not allowed
- `internal_error` - Internal server error

---

## Endpoints

### Health Endpoints

#### GET /health

Returns basic health status.

**Response: 200 OK**
```json
{
  "status": "healthy"
}
```

#### GET /ready

Returns readiness status (checks database and Docker).

**Response: 200 OK**
```json
{
  "status": "ready",
  "checks": {
    "database": "ok",
    "docker": "ok"
  }
}
```

**Response: 503 Service Unavailable**
```json
{
  "status": "not_ready",
  "checks": {
    "database": "ok",
    "docker": "failed"
  }
}
```

---

### Template Endpoints

#### POST /api/v1/templates

Create a new template.

**Request:**
```json
{
  "name": "WordPress",
  "version": "1.0.0",
  "compose_spec": "services:\n  wordpress:\n    image: wordpress",
  "creator_id": "user-123",
  "description": "WordPress with MySQL",
  "category": "cms",
  "tags": ["wordpress", "blog"],
  "price_monthly_cents": 500,
  "variables": [
    {
      "name": "MYSQL_PASSWORD",
      "description": "MySQL root password",
      "type": "string",
      "required": true,
      "secret": true
    }
  ]
}
```

**Response: 201 Created**
```json
{
  "id": "tmpl_abc123",
  "name": "WordPress",
  "slug": "wordpress",
  "version": "1.0.0",
  "compose_spec": "services:\n  wordpress:\n    image: wordpress",
  "creator_id": "user-123",
  "description": "WordPress with MySQL",
  "category": "cms",
  "tags": ["wordpress", "blog"],
  "price_monthly_cents": 500,
  "published": false,
  "resource_requirements": {
    "cpu_cores": 1,
    "memory_mb": 512,
    "disk_mb": 1024
  },
  "variables": [...],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Response: 400 Bad Request**
```json
{
  "error": "name is required",
  "code": "validation_error"
}
```

#### GET /api/v1/templates

List all templates with pagination.

**Query Parameters:**
- `limit` (optional, default: 20, max: 100)
- `offset` (optional, default: 0)
- `published` (optional, boolean)

**Response: 200 OK**
```json
{
  "templates": [...],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

#### GET /api/v1/templates/:id

Get a single template.

**Response: 200 OK**
```json
{
  "id": "tmpl_abc123",
  ...
}
```

**Response: 404 Not Found**
```json
{
  "error": "template not found",
  "code": "template_not_found"
}
```

#### PUT /api/v1/templates/:id

Update a template (only unpublished templates can be updated).

**Request:**
```json
{
  "name": "WordPress Pro",
  "description": "Updated description"
}
```

**Response: 200 OK**
```json
{
  "id": "tmpl_abc123",
  ...
}
```

**Response: 409 Conflict**
```json
{
  "error": "published templates cannot be modified",
  "code": "template_published"
}
```

#### DELETE /api/v1/templates/:id

Delete a template (only if no deployments exist).

**Response: 204 No Content**

**Response: 409 Conflict**
```json
{
  "error": "template has active deployments",
  "code": "template_in_use"
}
```

#### POST /api/v1/templates/:id/publish

Publish a template (makes it available for deployments).

**Response: 200 OK**
```json
{
  "id": "tmpl_abc123",
  "published": true,
  ...
}
```

**Response: 409 Conflict**
```json
{
  "error": "template is already published",
  "code": "already_published"
}
```

---

### Deployment Endpoints

#### POST /api/v1/deployments

Create a new deployment from a template.

**Request:**
```json
{
  "template_id": "tmpl_abc123",
  "customer_id": "cust-456",
  "name": "My WordPress Site",
  "variables": {
    "MYSQL_PASSWORD": "secret123"
  }
}
```

**Response: 201 Created**
```json
{
  "id": "depl_xyz789",
  "name": "My WordPress Site",
  "template_id": "tmpl_abc123",
  "template_version": "1.0.0",
  "customer_id": "cust-456",
  "status": "pending",
  "variables": {...},
  "domains": [],
  "containers": [],
  "resources": {...},
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Response: 400 Bad Request**
```json
{
  "error": "template_id is required",
  "code": "validation_error"
}
```

**Response: 404 Not Found**
```json
{
  "error": "template not found",
  "code": "template_not_found"
}
```

**Response: 409 Conflict**
```json
{
  "error": "template is not published",
  "code": "template_not_published"
}
```

#### GET /api/v1/deployments

List all deployments with pagination.

**Query Parameters:**
- `limit` (optional, default: 20, max: 100)
- `offset` (optional, default: 0)
- `template_id` (optional)
- `customer_id` (optional)
- `status` (optional)

**Response: 200 OK**
```json
{
  "deployments": [...],
  "total": 15,
  "limit": 20,
  "offset": 0
}
```

#### GET /api/v1/deployments/:id

Get a single deployment.

**Response: 200 OK**
```json
{
  "id": "depl_xyz789",
  ...
}
```

**Response: 404 Not Found**
```json
{
  "error": "deployment not found",
  "code": "deployment_not_found"
}
```

#### DELETE /api/v1/deployments/:id

Delete a deployment (stops containers first if running).

**Response: 204 No Content**

**Response: 404 Not Found**
```json
{
  "error": "deployment not found",
  "code": "deployment_not_found"
}
```

#### POST /api/v1/deployments/:id/start

Start a deployment (creates and starts Docker containers).

**Response: 200 OK**
```json
{
  "id": "depl_xyz789",
  "status": "running",
  "containers": [
    {
      "service_name": "wordpress",
      "container_id": "abc123...",
      "status": "running"
    }
  ],
  ...
}
```

**Response: 409 Conflict**
```json
{
  "error": "deployment is already running",
  "code": "invalid_transition"
}
```

#### POST /api/v1/deployments/:id/stop

Stop a running deployment.

**Response: 200 OK**
```json
{
  "id": "depl_xyz789",
  "status": "stopped",
  ...
}
```

**Response: 409 Conflict**
```json
{
  "error": "deployment is not running",
  "code": "invalid_transition"
}
```

---

## Handler Interface

```go
// Handler provides HTTP handlers for the API.
type Handler struct {
    store  store.Store
    docker docker.Client
    logger *slog.Logger
}

// NewHandler creates a new API handler.
func NewHandler(s store.Store, d docker.Client, l *slog.Logger) *Handler

// Routes returns the router with all routes configured.
func (h *Handler) Routes() http.Handler
```

---

## Middleware

### Request Logging

All requests are logged with:
- Method
- Path
- Status code
- Duration
- Request ID

### Request ID

Each request gets a unique ID via `X-Request-ID` header.

### JSON Content-Type

All responses set `Content-Type: application/json`.

### Error Recovery

Panics are recovered and return 500 Internal Server Error.

---

## Request/Response Types

```go
// CreateTemplateRequest is the request body for creating a template.
type CreateTemplateRequest struct {
    Name             string            `json:"name"`
    Version          string            `json:"version"`
    ComposeSpec      string            `json:"compose_spec"`
    CreatorID        string            `json:"creator_id"`
    Description      string            `json:"description,omitempty"`
    Category         string            `json:"category,omitempty"`
    Tags             []string          `json:"tags,omitempty"`
    PriceMonthly     int               `json:"price_monthly_cents,omitempty"`
    Variables        []VariableRequest `json:"variables,omitempty"`
}

// UpdateTemplateRequest is the request body for updating a template.
type UpdateTemplateRequest struct {
    Name         string   `json:"name,omitempty"`
    Description  string   `json:"description,omitempty"`
    Category     string   `json:"category,omitempty"`
    Tags         []string `json:"tags,omitempty"`
    PriceMonthly int      `json:"price_monthly_cents,omitempty"`
}

// CreateDeploymentRequest is the request body for creating a deployment.
type CreateDeploymentRequest struct {
    TemplateID string            `json:"template_id"`
    CustomerID string            `json:"customer_id"`
    Name       string            `json:"name,omitempty"`
    Variables  map[string]string `json:"variables,omitempty"`
}

// TemplateResponse is the response for template operations.
type TemplateResponse struct {
    ID                   string              `json:"id"`
    Name                 string              `json:"name"`
    Slug                 string              `json:"slug"`
    Description          string              `json:"description"`
    Version              string              `json:"version"`
    ComposeSpec          string              `json:"compose_spec"`
    Variables            []VariableResponse  `json:"variables"`
    ResourceRequirements ResourcesResponse   `json:"resource_requirements"`
    PriceMonthly         int                 `json:"price_monthly_cents"`
    Category             string              `json:"category"`
    Tags                 []string            `json:"tags"`
    Published            bool                `json:"published"`
    CreatorID            string              `json:"creator_id"`
    CreatedAt            time.Time           `json:"created_at"`
    UpdatedAt            time.Time           `json:"updated_at"`
}

// DeploymentResponse is the response for deployment operations.
type DeploymentResponse struct {
    ID              string               `json:"id"`
    Name            string               `json:"name"`
    TemplateID      string               `json:"template_id"`
    TemplateVersion string               `json:"template_version"`
    CustomerID      string               `json:"customer_id"`
    Status          string               `json:"status"`
    Variables       map[string]string    `json:"variables"`
    Domains         []DomainResponse     `json:"domains"`
    Containers      []ContainerResponse  `json:"containers"`
    Resources       ResourcesResponse    `json:"resources"`
    ErrorMessage    string               `json:"error_message,omitempty"`
    CreatedAt       time.Time            `json:"created_at"`
    UpdatedAt       time.Time            `json:"updated_at"`
    StartedAt       *time.Time           `json:"started_at,omitempty"`
    StoppedAt       *time.Time           `json:"stopped_at,omitempty"`
}

// ListTemplatesResponse is the response for listing templates.
type ListTemplatesResponse struct {
    Templates []TemplateResponse `json:"templates"`
    Total     int                `json:"total"`
    Limit     int                `json:"limit"`
    Offset    int                `json:"offset"`
}

// ListDeploymentsResponse is the response for listing deployments.
type ListDeploymentsResponse struct {
    Deployments []DeploymentResponse `json:"deployments"`
    Total       int                  `json:"total"`
    Limit       int                  `json:"limit"`
    Offset      int                  `json:"offset"`
}

// ErrorResponse is the error response format.
type ErrorResponse struct {
    Error string `json:"error"`
    Code  string `json:"code"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
    Status string `json:"status"`
}

// ReadyResponse is the readiness check response.
type ReadyResponse struct {
    Status string            `json:"status"`
    Checks map[string]string `json:"checks"`
}
```

---

## Test Categories (~50 tests)

### Template Endpoints (20 tests)
1. CreateTemplate_Success
2. CreateTemplate_MissingName
3. CreateTemplate_MissingVersion
4. CreateTemplate_MissingComposeSpec
5. CreateTemplate_MissingCreatorID
6. CreateTemplate_InvalidJSON
7. GetTemplate_Success
8. GetTemplate_NotFound
9. GetTemplateBySlug_Success
10. ListTemplates_Success
11. ListTemplates_Empty
12. ListTemplates_Pagination
13. ListTemplates_FilterPublished
14. UpdateTemplate_Success
15. UpdateTemplate_NotFound
16. UpdateTemplate_Published
17. DeleteTemplate_Success
18. DeleteTemplate_NotFound
19. PublishTemplate_Success
20. PublishTemplate_AlreadyPublished

### Deployment Endpoints (20 tests)
1. CreateDeployment_Success
2. CreateDeployment_MissingTemplateID
3. CreateDeployment_MissingCustomerID
4. CreateDeployment_TemplateNotFound
5. CreateDeployment_TemplateNotPublished
6. CreateDeployment_InvalidJSON
7. GetDeployment_Success
8. GetDeployment_NotFound
9. ListDeployments_Success
10. ListDeployments_Empty
11. ListDeployments_Pagination
12. ListDeployments_FilterByTemplate
13. ListDeployments_FilterByCustomer
14. ListDeployments_FilterByStatus
15. DeleteDeployment_Success
16. DeleteDeployment_NotFound
17. StartDeployment_Success
18. StartDeployment_NotFound
19. StartDeployment_AlreadyRunning
20. StopDeployment_Success

### Health Endpoints (5 tests)
1. Health_Success
2. Ready_AllHealthy
3. Ready_DatabaseFailed
4. Ready_DockerFailed
5. Ready_Unhealthy

### Middleware/Helpers (5 tests)
1. RequestID_Generated
2. ContentType_JSON
3. ErrorResponse_Format
4. Panic_Recovery
5. InvalidMethod_405

---

## Error Handling

### Validation Errors (400)
- Missing required fields
- Invalid field values
- Invalid JSON

### Not Found Errors (404)
- Template not found
- Deployment not found

### Conflict Errors (409)
- Template already published
- Template in use (has deployments)
- Deployment already running
- Invalid state transition

### Internal Errors (500)
- Database errors
- Docker errors
- Unexpected panics

---

## Security Considerations

### Input Validation
- Validate all request bodies
- Sanitize strings
- Validate IDs format

### No Authentication
- MVP has no authentication
- All endpoints are public
- Add authentication in future

### Rate Limiting
- Not implemented in MVP
- Add in future iterations

---

## Implementation Notes

1. Use chi router for routing
2. Use slog for structured logging
3. Store handles all database operations
4. Docker client handles container operations
5. All operations are synchronous in MVP
6. No background workers in MVP
