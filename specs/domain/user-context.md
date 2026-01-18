# User Context

## Overview

User Context represents the authenticated user information extracted from APIGate headers. This is not a stored entity but a request-scoped context object used for authorization and plan limit enforcement.

Per ADR-005, Hoster trusts headers injected by APIGate and does not manage users directly.

## Types

### AuthContext

The authentication and authorization context for a request.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `user_id` | UUID | Yes (if authenticated) | Unique user identifier |
| `plan_id` | string | Yes (if authenticated) | User's subscription plan ID |
| `plan_limits` | PlanLimits | Yes (if authenticated) | Resource limits from plan |
| `key_id` | UUID | No | API key ID (if key auth used) |
| `organization_id` | UUID | No | Organization ID (future use) |
| `authenticated` | bool | Yes | Whether request is authenticated |

### PlanLimits

Resource limits defined by the user's subscription plan.

| Field | Type | Description |
|-------|------|-------------|
| `max_deployments` | int | Maximum number of active deployments |
| `max_cpu_cores` | float64 | Maximum total CPU cores across deployments |
| `max_memory_mb` | int64 | Maximum total memory in MB |
| `max_disk_mb` | int64 | Maximum total disk space in MB |

## Header Contract

APIGate injects these headers for authenticated requests:

| Header | Type | Description |
|--------|------|-------------|
| `X-User-ID` | UUID | Authenticated user's ID |
| `X-Plan-ID` | string | User's subscription plan ID |
| `X-Plan-Limits` | JSON | Plan limits as JSON object |
| `X-Key-ID` | UUID | API key ID (if API key auth) |
| `X-Organization-ID` | UUID | Organization ID (future) |

### X-Plan-Limits Format

```json
{
  "max_deployments": 5,
  "max_cpu_cores": 4.0,
  "max_memory_mb": 8192,
  "max_disk_mb": 51200
}
```

## Behaviors

### Context Extraction

Auth context is extracted from HTTP headers:

```go
func ExtractContext(headers map[string]string) Context {
    userID := headers["X-User-ID"]

    if userID == "" {
        return Context{Authenticated: false}
    }

    limits, _ := ParsePlanLimits(headers["X-Plan-Limits"])

    return Context{
        UserID:         userID,
        PlanID:         headers["X-Plan-ID"],
        PlanLimits:     limits,
        KeyID:          headers["X-Key-ID"],
        OrganizationID: headers["X-Organization-ID"],
        Authenticated:  true,
    }
}
```

### Plan Limit Parsing

```go
func ParsePlanLimits(jsonStr string) (PlanLimits, error) {
    if jsonStr == "" {
        return DefaultPlanLimits(), nil
    }

    var limits PlanLimits
    if err := json.Unmarshal([]byte(jsonStr), &limits); err != nil {
        return PlanLimits{}, fmt.Errorf("invalid plan limits: %w", err)
    }

    return limits, nil
}

func DefaultPlanLimits() PlanLimits {
    return PlanLimits{
        MaxDeployments: 1,
        MaxCPUCores:    1.0,
        MaxMemoryMB:    1024,
        MaxDiskMB:      5120,
    }
}
```

### Context Storage

Auth context is stored in request context:

```go
type contextKey string

const authContextKey contextKey = "auth"

func WithContext(ctx context.Context, authCtx Context) context.Context {
    return context.WithValue(ctx, authContextKey, authCtx)
}

func FromContext(ctx context.Context) Context {
    if authCtx, ok := ctx.Value(authContextKey).(Context); ok {
        return authCtx
    }
    return Context{Authenticated: false}
}
```

## Authorization Functions

Pure functions for authorization checks (no I/O):

### Resource Ownership

```go
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
```

### Plan Limits

```go
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

// ValidateResourceLimits checks if requested resources are within plan limits
func ValidateResourceLimits(ctx Context, currentUsage, requested Resources) (bool, string) {
    total := Resources{
        CPUCores: currentUsage.CPUCores + requested.CPUCores,
        MemoryMB: currentUsage.MemoryMB + requested.MemoryMB,
        DiskMB:   currentUsage.DiskMB + requested.DiskMB,
    }

    if total.CPUCores > ctx.PlanLimits.MaxCPUCores {
        return false, fmt.Sprintf("CPU limit exceeded: %.1f/%.1f cores", total.CPUCores, ctx.PlanLimits.MaxCPUCores)
    }
    if total.MemoryMB > ctx.PlanLimits.MaxMemoryMB {
        return false, fmt.Sprintf("memory limit exceeded: %dMB/%dMB", total.MemoryMB, ctx.PlanLimits.MaxMemoryMB)
    }
    if total.DiskMB > ctx.PlanLimits.MaxDiskMB {
        return false, fmt.Sprintf("disk limit exceeded: %dMB/%dMB", total.DiskMB, ctx.PlanLimits.MaxDiskMB)
    }

    return true, ""
}
```

## Trust Model

**Problem**: How does Hoster know headers are legitimate?

**Solution**: Network isolation. Hoster is NOT publicly accessible:

```
Internet ──► APIGate (public:8080) ──► Hoster (internal:9090)
                                           │
                                    Not directly accessible
```

Implementation options:
1. **Docker network**: Hoster only listens on Docker bridge network
2. **Firewall**: iptables blocks external access to port 9090
3. **Bind address**: Hoster binds to `127.0.0.1` or Docker network IP only

### Defense in Depth (Optional)

Add shared secret validation:

```go
// Optional: validate shared secret in middleware
if m.sharedSecret != "" {
    if r.Header.Get("X-APIGate-Secret") != m.sharedSecret {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
}
```

## Configuration

```yaml
auth:
  mode: header           # "header" or "none"
  trusted_header: X-User-ID
  require_auth: true     # Require auth for protected endpoints
  shared_secret: ""      # Optional: validate X-APIGate-Secret
```

### Development Mode

For local development without APIGate:

```yaml
auth:
  mode: none  # Skip auth checks
```

Or inject headers manually:
```bash
curl -H "X-User-ID: dev-user-123" http://localhost:9090/api/v1/templates
```

## Not Supported

1. **User management**: Handled by APIGate
   - Registration, login, password reset
   - Profile management
   - Email verification

2. **API key management**: Handled by APIGate
   - Key creation, rotation, revocation
   - Key scopes and permissions

3. **Role-based access control**: Beyond owner checks
   - Admin roles
   - Team permissions
   - Fine-grained permissions

4. **Organization sharing**: Single-user ownership only
   - Team templates
   - Shared deployments

5. **Session management**: Handled by APIGate
   - Token refresh
   - Session invalidation
   - Multi-device sessions

## Tests

- `internal/core/auth/context_test.go` - Context extraction tests
- `internal/core/auth/authorization_test.go` - Authorization function tests
- `internal/core/limits/validation_test.go` - Plan limit validation tests
- `internal/shell/api/middleware/auth_test.go` - Middleware integration tests
