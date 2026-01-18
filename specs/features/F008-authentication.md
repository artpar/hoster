# F008: Authentication Integration

## Overview

Integrate APIGate's authentication headers into Hoster for user identification and authorization.

## User Stories

### US-1: As a user, I want my requests to be authenticated so that I can manage my own resources

**Acceptance Criteria:**
- Requests with valid `X-User-ID` header are accepted
- User ID is extracted and made available to API handlers
- Missing `X-User-ID` returns 401 Unauthorized for protected endpoints
- Invalid headers are rejected

### US-2: As a user, I want to only access my own templates and deployments

**Acceptance Criteria:**
- Users can only view their own templates (unless published)
- Users can only manage their own deployments
- Users cannot modify other users' resources
- Appropriate 403 Forbidden returned for unauthorized access

### US-3: As a creator, I want my plan limits visible so I know my quotas

**Acceptance Criteria:**
- `X-Plan-Limits` header is parsed and accessible
- Limits are checked before resource creation
- Clear error messages when limits are exceeded

## Technical Specification

### Header Contract (from APIGate)

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `X-User-ID` | UUID | Yes (protected) | Authenticated user's ID |
| `X-Plan-ID` | string | Yes | User's subscription plan ID |
| `X-Plan-Limits` | JSON | Yes | Plan limits object |
| `X-Key-ID` | UUID | No | API key ID (if key auth) |
| `X-Organization-ID` | UUID | No | Organization ID (future) |

### Plan Limits Structure

```json
{
  "max_deployments": 5,
  "max_cpu_cores": 4.0,
  "max_memory_mb": 8192,
  "max_disk_mb": 51200
}
```

### Auth Context (Pure Core)

```go
// internal/core/auth/context.go

type Context struct {
    UserID         string
    PlanID         string
    PlanLimits     PlanLimits
    KeyID          string
    OrganizationID string
    Authenticated  bool
}

type PlanLimits struct {
    MaxDeployments int     `json:"max_deployments"`
    MaxCPUCores    float64 `json:"max_cpu_cores"`
    MaxMemoryMB    int64   `json:"max_memory_mb"`
    MaxDiskMB      int64   `json:"max_disk_mb"`
}

// ExtractContext extracts auth context from HTTP headers
func ExtractContext(headers map[string]string) Context

// ParsePlanLimits parses JSON plan limits string
func ParsePlanLimits(jsonStr string) (PlanLimits, error)
```

### Authorization Functions (Pure Core)

```go
// internal/core/auth/authorization.go

// CanViewTemplate checks if user can view a template
func CanViewTemplate(ctx Context, template domain.Template) bool {
    // Published templates visible to all
    if template.Published {
        return true
    }
    // Otherwise only creator can view
    return ctx.Authenticated && ctx.UserID == template.CreatorID
}

// CanModifyTemplate checks if user can modify a template
func CanModifyTemplate(ctx Context, template domain.Template) bool {
    return ctx.Authenticated && ctx.UserID == template.CreatorID
}

// CanViewDeployment checks if user can view a deployment
func CanViewDeployment(ctx Context, deployment domain.Deployment) bool {
    return ctx.Authenticated && ctx.UserID == deployment.CustomerID
}

// CanManageDeployment checks if user can manage a deployment
func CanManageDeployment(ctx Context, deployment domain.Deployment) bool {
    return ctx.Authenticated && ctx.UserID == deployment.CustomerID
}

// CanCreateDeployment checks if user can create another deployment
func CanCreateDeployment(ctx Context, currentCount int) (bool, string) {
    if !ctx.Authenticated {
        return false, "authentication required"
    }
    if currentCount >= ctx.PlanLimits.MaxDeployments {
        return false, fmt.Sprintf("plan limit reached: max %d deployments", ctx.PlanLimits.MaxDeployments)
    }
    return true, ""
}
```

### Auth Middleware (Shell)

```go
// internal/shell/api/middleware/auth.go

type AuthMiddleware struct {
    trustedHeader string
    requireAuth   bool
    sharedSecret  string // optional defense in depth
}

func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Optional: validate shared secret
        if m.sharedSecret != "" {
            if r.Header.Get("X-APIGate-Secret") != m.sharedSecret {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
        }

        // Extract auth context from headers
        headers := map[string]string{
            "X-User-ID":         r.Header.Get("X-User-ID"),
            "X-Plan-ID":         r.Header.Get("X-Plan-ID"),
            "X-Plan-Limits":     r.Header.Get("X-Plan-Limits"),
            "X-Key-ID":          r.Header.Get("X-Key-ID"),
            "X-Organization-ID": r.Header.Get("X-Organization-ID"),
        }
        ctx := auth.ExtractContext(headers)

        // Require auth if configured
        if m.requireAuth && !ctx.Authenticated {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        // Store in request context
        r = r.WithContext(auth.WithContext(r.Context(), ctx))
        next.ServeHTTP(w, r)
    })
}
```

### Endpoint Protection

| Endpoint | Auth Required | Description |
|----------|---------------|-------------|
| `GET /templates` (published) | No | Browse marketplace |
| `GET /templates/:id` (published) | No | View template details |
| `POST /templates` | Yes | Create template |
| `PATCH /templates/:id` | Yes + Owner | Update template |
| `DELETE /templates/:id` | Yes + Owner | Delete template |
| `GET /deployments` | Yes | List user's deployments |
| `POST /deployments` | Yes + Limits | Create deployment |
| `PATCH /deployments/:id` | Yes + Owner | Update deployment |
| `DELETE /deployments/:id` | Yes + Owner | Delete deployment |
| `POST /deployments/:id/start` | Yes + Owner | Start deployment |
| `POST /deployments/:id/stop` | Yes + Owner | Stop deployment |

### Configuration

```yaml
auth:
  mode: header              # "header" or "none"
  trusted_header: X-User-ID
  require_auth: true        # Require auth for protected endpoints
  shared_secret: ""         # Optional: validate X-APIGate-Secret

# Development mode
auth:
  mode: none               # Skip auth checks
```

## Test Cases

### Unit Tests (internal/core/auth/)

```go
// context_test.go
func TestExtractContext_WithAllHeaders(t *testing.T)
func TestExtractContext_WithMissingHeaders(t *testing.T)
func TestExtractContext_WithInvalidPlanLimits(t *testing.T)
func TestParsePlanLimits_ValidJSON(t *testing.T)
func TestParsePlanLimits_InvalidJSON(t *testing.T)
func TestParsePlanLimits_MissingFields(t *testing.T)

// authorization_test.go
func TestCanViewTemplate_PublishedTemplate(t *testing.T)
func TestCanViewTemplate_UnpublishedOwner(t *testing.T)
func TestCanViewTemplate_UnpublishedOther(t *testing.T)
func TestCanModifyTemplate_Owner(t *testing.T)
func TestCanModifyTemplate_NonOwner(t *testing.T)
func TestCanViewDeployment_Owner(t *testing.T)
func TestCanViewDeployment_NonOwner(t *testing.T)
func TestCanManageDeployment_Owner(t *testing.T)
func TestCanManageDeployment_NonOwner(t *testing.T)
func TestCanCreateDeployment_WithinLimit(t *testing.T)
func TestCanCreateDeployment_AtLimit(t *testing.T)
func TestCanCreateDeployment_Unauthenticated(t *testing.T)
```

### Integration Tests (internal/shell/api/)

```go
// middleware_test.go
func TestAuthMiddleware_ValidHeaders(t *testing.T)
func TestAuthMiddleware_MissingUserID(t *testing.T)
func TestAuthMiddleware_InvalidSecret(t *testing.T)
func TestAuthMiddleware_AuthNotRequired(t *testing.T)

// resources_test.go
func TestTemplateResource_CreateRequiresAuth(t *testing.T)
func TestTemplateResource_UpdateRequiresOwner(t *testing.T)
func TestDeploymentResource_ListOnlyOwn(t *testing.T)
func TestDeploymentResource_CreateChecksLimit(t *testing.T)
```

## Files to Create

- `internal/core/auth/context.go` - Auth context extraction (pure)
- `internal/core/auth/context_test.go` - Tests
- `internal/core/auth/authorization.go` - Permission checks (pure)
- `internal/core/auth/authorization_test.go` - Tests
- `internal/shell/api/middleware/auth.go` - HTTP middleware
- `internal/shell/api/middleware/auth_test.go` - Tests

## Files to Modify

- `internal/shell/api/resources/template.go` - Add auth checks
- `internal/shell/api/resources/deployment.go` - Add auth checks
- `cmd/hoster/config.go` - Add auth config section
- `cmd/hoster/server.go` - Add middleware to router

## NOT Supported

- User registration (handled by APIGate)
- Password reset (handled by APIGate)
- OAuth flows (handled by APIGate)
- API key management (handled by APIGate)
- Session management (handled by APIGate)
- Role-based access control beyond owner checks
- Organization/team sharing
- Fine-grained permissions

## Dependencies

- ADR-003: JSON:API with api2go (middleware integration)
- ADR-005: APIGate Integration (header contract)

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Header spoofing | Network isolation - Hoster not publicly accessible |
| Missing headers | Default to unauthenticated, require auth for protected endpoints |
| Invalid plan limits | Fail safe - reject request if limits can't be parsed |
