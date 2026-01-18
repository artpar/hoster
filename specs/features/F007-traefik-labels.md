# F007: Traefik Label Generation

## User Story

As a **deployment orchestrator**, I need to generate Traefik reverse proxy labels for deployed containers, so that services can be accessed externally via auto-generated subdomains with optional TLS.

## Overview

This feature provides pure functions to generate Traefik Docker labels that configure reverse proxy routing for deployments. All functions live in `internal/core/traefik/` and have **no I/O** - they take values in and return values out, compliant with ADR-002 "Values as Boundaries".

Without this feature, deployed containers have no external HTTP/HTTPS access.

## Acceptance Criteria

- [ ] Generate Traefik routing labels for HTTP traffic
- [ ] Generate Traefik routing labels for HTTPS traffic (with TLS)
- [ ] Support custom ports for service routing
- [ ] Use consistent naming for routers and services
- [ ] All functions are pure (no I/O, no side effects)
- [ ] 100% test coverage

## Functions

### Label Generation

```go
// LabelParams contains parameters for generating Traefik labels.
type LabelParams struct {
    DeploymentID string
    ServiceName  string
    Hostname     string
    Port         int
    EnableTLS    bool
}

// GenerateLabels generates Traefik reverse proxy labels for a service.
// Returns a map of Docker labels to apply to the container.
func GenerateLabels(params LabelParams) map[string]string
```

## Generated Labels

### HTTP Only (EnableTLS = false)

| Label | Value |
|-------|-------|
| `traefik.enable` | `"true"` |
| `traefik.http.routers.{name}.rule` | `Host(\`{hostname}\`)` |
| `traefik.http.routers.{name}.entrypoints` | `"web"` |
| `traefik.http.services.{name}.loadbalancer.server.port` | `"{port}"` |

### With TLS (EnableTLS = true)

HTTP labels plus:

| Label | Value |
|-------|-------|
| `traefik.http.routers.{name}-secure.rule` | `Host(\`{hostname}\`)` |
| `traefik.http.routers.{name}-secure.entrypoints` | `"websecure"` |
| `traefik.http.routers.{name}-secure.tls` | `"true"` |
| `traefik.http.routers.{name}-secure.tls.certresolver` | `"letsencrypt"` |

### Router/Service Naming

The `{name}` used in label keys follows this pattern:
```
{deploymentID}-{serviceName}
```

Example: `deploy-abc123-web`

This ensures unique names across all deployments.

## Examples

### Basic HTTP Service

**Input:**
```go
LabelParams{
    DeploymentID: "abc123",
    ServiceName:  "web",
    Hostname:     "myapp-abc123.apps.hoster.io",
    Port:         80,
    EnableTLS:    false,
}
```

**Output:**
```go
map[string]string{
    "traefik.enable": "true",
    "traefik.http.routers.abc123-web.rule": "Host(`myapp-abc123.apps.hoster.io`)",
    "traefik.http.routers.abc123-web.entrypoints": "web",
    "traefik.http.services.abc123-web.loadbalancer.server.port": "80",
}
```

### HTTPS Service with TLS

**Input:**
```go
LabelParams{
    DeploymentID: "def456",
    ServiceName:  "api",
    Hostname:     "api.example.com",
    Port:         3000,
    EnableTLS:    true,
}
```

**Output:**
```go
map[string]string{
    "traefik.enable": "true",
    // HTTP route
    "traefik.http.routers.def456-api.rule": "Host(`api.example.com`)",
    "traefik.http.routers.def456-api.entrypoints": "web",
    // HTTPS route
    "traefik.http.routers.def456-api-secure.rule": "Host(`api.example.com`)",
    "traefik.http.routers.def456-api-secure.entrypoints": "websecure",
    "traefik.http.routers.def456-api-secure.tls": "true",
    "traefik.http.routers.def456-api-secure.tls.certresolver": "letsencrypt",
    // Service (shared by both routes)
    "traefik.http.services.def456-api.loadbalancer.server.port": "3000",
}
```

## Edge Cases

### Port Values
- Port 80 → Standard HTTP port
- Port 443 → Would still use loadbalancer port (container port, not host)
- Port 0 → Passed through (validation at caller level)
- High ports (e.g., 8080, 3000) → Work correctly

### Hostname Values
- Auto-generated: `myapp-abc123.apps.hoster.io`
- Custom domain: `api.example.com`
- Subdomain: `www.example.com`
- Empty hostname → Passed through (validation at caller level)

### Naming Conflicts
- Router names include deployment ID → No conflicts across deployments
- Service names include deployment ID → No conflicts across deployments
- Multiple services in same deployment → Different service names

## Not Supported

| Feature | Reason |
|---------|--------|
| Custom routing rules | Beyond prototype scope |
| Path-based routing | Domain-only routing for simplicity |
| Load balancer weights | Single instance per service |
| Middleware (auth, rate limit) | Beyond prototype scope |
| Custom TLS certificates | Uses Let's Encrypt only |
| HTTP to HTTPS redirect | Can be added later |
| Multiple hostnames per service | Single domain per service |

## Dependencies

- None (pure function, no external dependencies)

## Integration

These labels are added to containers by the deployment planning stage.

```go
// In core/deployment/container.go (future integration)
labels := traefik.GenerateLabels(traefik.LabelParams{
    DeploymentID: deployment.ID,
    ServiceName:  service.Name,
    Hostname:     deployment.Domains[0].Hostname,
    Port:         service.Ports[0].Target,
    EnableTLS:    deployment.Domains[0].SSLEnabled,
})
for k, v := range labels {
    containerPlan.Labels[k] = v
}
```

## Implementation Notes

1. Function must be pure (no I/O, no side effects)
2. Use `fmt.Sprintf` for label value formatting
3. Port must be converted to string in label value
4. Hostname must be wrapped in backticks in Host rule

## Tests

### Test File: `internal/core/traefik/labels_test.go`

| Test | Description |
|------|-------------|
| `TestGenerateLabels_Basic` | HTTP-only service |
| `TestGenerateLabels_WithTLS` | HTTPS service with TLS enabled |
| `TestGenerateLabels_CustomPort` | Non-standard port (e.g., 3000) |
| `TestGenerateLabels_RouterNaming` | Verify unique router names |
| `TestGenerateLabels_ServiceNaming` | Verify unique service names |

**Total: ~5-7 tests**

## Traefik Configuration Requirements

For these labels to work, Traefik must be configured with:

1. **Entrypoints**:
   - `web` on port 80
   - `websecure` on port 443 (if TLS enabled)

2. **Certificate Resolver** (if TLS enabled):
   - Name: `letsencrypt`
   - Provider: ACME

3. **Docker Provider**:
   - Watch: enabled
   - Network: same network as containers

Example Traefik static configuration:
```yaml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@hoster.io
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

providers:
  docker:
    watch: true
    exposedByDefault: false
```
