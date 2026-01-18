# ADR-005: APIGate Integration for Authentication and Billing

## Status
Accepted

## Context

Hoster needs:
- User authentication (registration, login, sessions/tokens)
- User management (accounts, profiles, API keys)
- Billing/subscriptions (pricing tiers, payment processing, usage tracking)
- Rate limiting (per-user, per-plan)

Building these from scratch would require significant effort:
- OAuth/JWT implementation
- Password hashing, reset flows
- Payment provider integration (Stripe, etc.)
- Subscription management
- Admin dashboard for user management

APIGate (https://github.com/artpar/apigate) is an existing project that provides all of these features as a standalone gateway.

## Decision

We will integrate **APIGate** as a reverse proxy in front of Hoster's API. APIGate handles authentication and billing; Hoster trusts APIGate's injected headers.

### Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Client    │────►│     APIGate      │────►│     Hoster      │
│  (Browser)  │     │   (Port 8080)    │     │  (Port 9090)    │
└─────────────┘     └──────────────────┘     └─────────────────┘
                           │                         │
                           │                         │
                    Authenticates              Trusts headers:
                    Rate limits                X-User-ID
                    Bills usage                X-Plan-ID
                                               X-Plan-Limits
```

### Header Contract

APIGate injects these headers for authenticated requests:

| Header | Type | Description |
|--------|------|-------------|
| `X-User-ID` | UUID | Authenticated user's ID |
| `X-Plan-ID` | string | User's subscription plan ID |
| `X-Plan-Limits` | JSON | Plan limits `{"max_deployments": 5, ...}` |
| `X-Key-ID` | UUID | API key ID (if API key auth) |
| `X-Organization-ID` | UUID | Organization ID (future) |

### Trust Model

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

### Security Enhancement (Defense in Depth)

Optionally add shared secret validation:

```yaml
# Hoster config
auth:
  mode: header
  trusted_header: X-User-ID
  shared_secret: ${APIGATE_SECRET}  # Validate X-APIGate-Secret header
```

## Implementation

### Auth Middleware

```go
// internal/shell/api/middleware/auth.go

type AuthMiddleware struct {
    trustedHeader string
    requireAuth   bool
    sharedSecret  string // optional
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

        // Extract auth context
        ctx := auth.ExtractContext(r, m.trustedHeader)

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
    MaxDeployments int   `json:"max_deployments"`
    MaxCPUCores    float64 `json:"max_cpu_cores"`
    MaxMemoryMB    int64   `json:"max_memory_mb"`
    MaxDiskMB      int64   `json:"max_disk_mb"`
}

// ExtractContext extracts auth context from HTTP headers
func ExtractContext(r *http.Request, trustedHeader string) Context
```

### Authorization (Pure Core)

```go
// internal/core/auth/authorization.go

// CanModifyTemplate checks if user can modify a template
func CanModifyTemplate(ctx Context, template domain.Template) bool {
    return ctx.Authenticated && ctx.UserID == template.CreatorID
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

### Resource Changes

Remove client-provided IDs from requests:

```go
// Before
type CreateTemplateRequest struct {
    Name      string `json:"name"`
    CreatorID string `json:"creator_id"`  // Client provides
}

// After
type CreateTemplateRequest struct {
    Name string `json:"name"`
    // CreatorID comes from X-User-ID header
}
```

### Billing Integration

Report usage events to APIGate:

```go
// internal/shell/billing/client.go

type Client interface {
    // MeterUsage reports a usage event to APIGate
    MeterUsage(ctx context.Context, event MeterEvent) error
}

type MeterEvent struct {
    UserID       string            `json:"user_id"`
    EventType    string            `json:"event_type"`
    ResourceID   string            `json:"resource_id"`
    ResourceType string            `json:"resource_type"`
    Metadata     map[string]string `json:"metadata"`
    Timestamp    time.Time         `json:"timestamp"`
}
```

Events to report:
- `deployment_created` - When deployment is created
- `deployment_started` - When deployment starts running
- `deployment_stopped` - When deployment stops
- `deployment_deleted` - When deployment is deleted

## Consequences

### Positive
- **No auth code**: APIGate handles registration, login, sessions
- **No billing code**: APIGate handles Stripe/Paddle integration
- **No admin UI**: APIGate provides user management portal
- **Faster development**: Focus on deployment features
- **Proven infrastructure**: Reuse working auth/billing system

### Negative
- **Dependency**: Hoster requires APIGate to run
- **Network setup**: Must ensure proper isolation
- **Coordination**: Changes to auth/billing require APIGate updates
- **Development complexity**: Need both services running locally

### Neutral
- Hoster can still run standalone (no auth) for development
- Header-based auth is a standard pattern (like API gateways)

## Alternatives Considered

### Build Auth from Scratch
- **Rejected because**: Significant effort, not core to Hoster's value
- **Would reconsider if**: APIGate doesn't meet our needs

### Use Auth0/Clerk/Supabase Auth
- **Rejected because**: External dependency, cost, vendor lock-in
- **Would reconsider if**: We need features APIGate doesn't provide

### Embed Auth in Hoster
- **Rejected because**: Duplicates APIGate functionality
- **Would reconsider if**: We need simpler deployment (single binary)

## Configuration

```yaml
# config.yaml

auth:
  mode: header           # "header" or "none"
  trusted_header: X-User-ID
  require_auth: true     # Require auth for protected endpoints
  shared_secret: ""      # Optional: validate X-APIGate-Secret

billing:
  enabled: true
  apigate_url: http://apigate:8080
  report_interval: 60s   # Batch event reporting
```

## Development Mode

For local development without APIGate:

```yaml
auth:
  mode: none  # Skip auth checks
```

Or inject headers manually in requests:
```bash
curl -H "X-User-ID: dev-user-123" http://localhost:9090/api/v1/templates
```

## Files to Create

- `internal/core/auth/context.go` - Auth context extraction (pure)
- `internal/core/auth/authorization.go` - Permission checks (pure)
- `internal/core/limits/validation.go` - Plan limit validation (pure)
- `internal/shell/api/middleware/auth.go` - Auth middleware
- `internal/shell/billing/client.go` - Billing client

## Files to Modify

- `internal/shell/api/resources/*.go` - Use auth context
- `cmd/hoster/config.go` - Add auth/billing config
- `cmd/hoster/server.go` - Add middleware, initialize billing

## References

- APIGate: https://github.com/artpar/apigate
- API Gateway Pattern: https://microservices.io/patterns/apigateway.html
- Header-based Auth: https://auth0.com/docs/secure/tokens/access-tokens/validate-access-tokens
