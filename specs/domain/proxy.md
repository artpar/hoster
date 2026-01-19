# Domain Spec: App Proxy

## Overview

The App Proxy is Hoster's built-in reverse proxy that routes incoming HTTP requests to deployed containers based on hostname. It is a **core feature** of Hoster, not a third-party dependency.

## Design Principles

1. **Built-in**: No external dependencies (Traefik, nginx, etc.)
2. **Simple**: Use Go's standard `net/http/httputil.ReverseProxy`
3. **Fast**: O(1) hostname lookup via database index
4. **Secure**: Only route to running deployments owned by valid users
5. **Observable**: Metrics and logging for all requests

## Architecture

```
                    ┌─────────────────────────────────────────────────┐
                    │                   APIGate                        │
                    │                  (Port 8080)                     │
                    │                                                  │
                    │  Routes *.apps.domain → upstream localhost:9091  │
                    └─────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            HOSTER APP PROXY                                      │
│                              (Port 9091)                                         │
│                                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Request    │    │   Hostname   │    │   Route      │    │   Response   │  │
│  │   Handler    │───▶│   Resolver   │───▶│   Proxy      │───▶│   Handler    │  │
│  └──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                   │                   │                   │           │
│         │                   ▼                   ▼                   │           │
│         │           ┌──────────────┐    ┌──────────────┐           │           │
│         │           │   Database   │    │   Container  │           │           │
│         │           │   (domains)  │    │   (ports)    │           │           │
│         │           └──────────────┘    └──────────────┘           │           │
│         │                                                           │           │
│         └───────────────────────────────────────────────────────────┘           │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            DOCKER CONTAINERS                                     │
│                                                                                  │
│   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                │
│   │ my-blog         │  │ my-shop         │  │ gitea           │                │
│   │ Port: 30001     │  │ Port: 30002     │  │ Port: 30003     │                │
│   │ Status: running │  │ Status: running │  │ Status: stopped │                │
│   └─────────────────┘  └─────────────────┘  └─────────────────┘                │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Request Flow

```
1. User requests: https://my-blog.apps.hoster.io/posts/1
2. DNS resolves *.apps.hoster.io → APIGate IP
3. APIGate receives request, matches *.apps.* route
4. APIGate proxies to Hoster App Proxy (localhost:9091)
5. App Proxy extracts hostname: "my-blog.apps.hoster.io"
6. App Proxy queries database: SELECT * FROM deployments WHERE domain = ?
7. Found: deployment "depl_xyz", node "local", port 30001, status "running"
8. App Proxy creates reverse proxy to http://127.0.0.1:30001
9. Request proxied to container
10. Response returned to user
```

## Core Types

### ProxyTarget (Pure)

```go
// internal/core/proxy/target.go

// ProxyTarget represents the destination for a proxied request.
// This is a pure data type with no I/O.
type ProxyTarget struct {
    // DeploymentID is the deployment this target belongs to
    DeploymentID string

    // NodeID is the node where the container runs ("local" or node ID)
    NodeID string

    // Port is the host port the container is bound to
    Port int

    // Status is the deployment status (running, stopped, etc.)
    Status string

    // CustomerID is the owner of the deployment
    CustomerID string
}

// CanRoute returns true if the target can accept traffic.
func (t ProxyTarget) CanRoute() bool {
    return t.Status == "running" && t.Port > 0
}

// IsLocal returns true if the target is on the local node.
func (t ProxyTarget) IsLocal() bool {
    return t.NodeID == "" || t.NodeID == "local"
}

// Address returns the target address for local containers.
// For remote containers, use NodePool to get tunneled address.
func (t ProxyTarget) LocalAddress() string {
    return fmt.Sprintf("127.0.0.1:%d", t.Port)
}
```

### HostnameResolver (Pure)

```go
// internal/core/proxy/resolver.go

// ResolveResult represents the result of hostname resolution.
type ResolveResult struct {
    Found   bool
    Target  ProxyTarget
    Error   error
}

// HostnameParser extracts deployment info from hostname.
// Pure function - no I/O.
type HostnameParser struct {
    BaseDomain string // e.g., "apps.hoster.io"
}

// Parse extracts the deployment slug from a hostname.
// "my-blog.apps.hoster.io" → "my-blog"
// "my-blog.apps.hoster.io:8080" → "my-blog"
func (p HostnameParser) Parse(hostname string) (slug string, ok bool) {
    // Strip port if present
    host := hostname
    if idx := strings.LastIndex(hostname, ":"); idx != -1 {
        host = hostname[:idx]
    }

    // Check if hostname ends with base domain
    suffix := "." + p.BaseDomain
    if !strings.HasSuffix(host, suffix) {
        return "", false
    }

    // Extract slug (everything before the suffix)
    slug = strings.TrimSuffix(host, suffix)
    if slug == "" {
        return "", false
    }

    return slug, true
}
```

### ProxyError (Pure)

```go
// internal/core/proxy/errors.go

// ProxyErrorType defines the type of proxy error.
type ProxyErrorType int

const (
    ErrorNotFound ProxyErrorType = iota
    ErrorStopped
    ErrorUnavailable
    ErrorUpstreamTimeout
    ErrorUpstreamError
)

// ProxyError represents an error during proxying.
type ProxyError struct {
    Type       ProxyErrorType
    Hostname   string
    Message    string
    StatusCode int
}

func (e ProxyError) Error() string {
    return e.Message
}

// NewNotFoundError creates an error for unknown hostname.
func NewNotFoundError(hostname string) ProxyError {
    return ProxyError{
        Type:       ErrorNotFound,
        Hostname:   hostname,
        Message:    fmt.Sprintf("app not found: %s", hostname),
        StatusCode: 404,
    }
}

// NewStoppedError creates an error for stopped deployment.
func NewStoppedError(hostname string) ProxyError {
    return ProxyError{
        Type:       ErrorStopped,
        Hostname:   hostname,
        Message:    fmt.Sprintf("app is stopped: %s", hostname),
        StatusCode: 503,
    }
}

// NewUnavailableError creates an error for unreachable container.
func NewUnavailableError(hostname string) ProxyError {
    return ProxyError{
        Type:       ErrorUnavailable,
        Hostname:   hostname,
        Message:    fmt.Sprintf("app unavailable: %s", hostname),
        StatusCode: 503,
    }
}
```

## Shell Components

### ProxyServer (Shell)

```go
// internal/shell/proxy/server.go

// ProxyServer is the HTTP server that handles app routing.
type ProxyServer struct {
    store      store.Store
    nodePool   *docker.NodePool  // For remote node tunnels
    parser     proxy.HostnameParser
    httpClient *http.Client
    logger     *slog.Logger

    // Configuration
    config ProxyConfig
}

type ProxyConfig struct {
    Address      string        // e.g., "0.0.0.0:9091"
    BaseDomain   string        // e.g., "apps.hoster.io"
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
}

// ServeHTTP implements http.Handler.
func (s *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    hostname := r.Host

    // 1. Parse hostname to extract slug
    slug, ok := s.parser.Parse(hostname)
    if !ok {
        s.serveError(w, r, proxy.NewNotFoundError(hostname))
        return
    }

    // 2. Resolve target from database
    target, err := s.resolveTarget(ctx, slug, hostname)
    if err != nil {
        s.serveError(w, r, err)
        return
    }

    // 3. Check if routable
    if !target.CanRoute() {
        s.serveError(w, r, proxy.NewStoppedError(hostname))
        return
    }

    // 4. Get upstream URL
    upstreamURL, err := s.getUpstreamURL(ctx, target)
    if err != nil {
        s.serveError(w, r, proxy.NewUnavailableError(hostname))
        return
    }

    // 5. Proxy the request
    s.proxyRequest(w, r, upstreamURL, target)
}

func (s *ProxyServer) resolveTarget(ctx context.Context, slug, hostname string) (proxy.ProxyTarget, error) {
    // Query database for deployment by domain hostname
    deployment, err := s.store.GetDeploymentByDomain(ctx, hostname)
    if err != nil {
        if errors.Is(err, store.ErrNotFound) {
            return proxy.ProxyTarget{}, proxy.NewNotFoundError(hostname)
        }
        return proxy.ProxyTarget{}, err
    }

    return proxy.ProxyTarget{
        DeploymentID: deployment.ID,
        NodeID:       deployment.NodeID,
        Port:         deployment.ProxyPort,
        Status:       string(deployment.Status),
        CustomerID:   deployment.CustomerID,
    }, nil
}

func (s *ProxyServer) getUpstreamURL(ctx context.Context, target proxy.ProxyTarget) (*url.URL, error) {
    if target.IsLocal() {
        return url.Parse("http://" + target.LocalAddress())
    }

    // For remote nodes, get SSH tunnel
    tunnel, err := s.nodePool.GetTunnel(ctx, target.NodeID, target.Port)
    if err != nil {
        return nil, err
    }

    return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", tunnel.LocalPort))
}

func (s *ProxyServer) proxyRequest(w http.ResponseWriter, r *http.Request, upstream *url.URL, target proxy.ProxyTarget) {
    proxy := httputil.NewSingleHostReverseProxy(upstream)

    // Customize director to set proper headers
    originalDirector := proxy.Director
    proxy.Director = func(req *http.Request) {
        originalDirector(req)
        req.Header.Set("X-Forwarded-Host", r.Host)
        req.Header.Set("X-Real-IP", getRealIP(r))
        req.Header.Set("X-Deployment-ID", target.DeploymentID)
    }

    // Handle errors
    proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
        s.logger.Error("proxy error",
            "hostname", r.Host,
            "deployment", target.DeploymentID,
            "error", err,
        )
        s.serveError(w, r, proxy.NewUnavailableError(r.Host))
    }

    proxy.ServeHTTP(w, r)
}

func (s *ProxyServer) serveError(w http.ResponseWriter, r *http.Request, err error) {
    var proxyErr proxy.ProxyError
    if errors.As(err, &proxyErr) {
        s.renderErrorPage(w, proxyErr)
        return
    }

    // Unknown error
    http.Error(w, "Internal server error", 500)
}

func (s *ProxyServer) renderErrorPage(w http.ResponseWriter, err proxy.ProxyError) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(err.StatusCode)

    // Render friendly error page
    tmpl := s.getErrorTemplate(err.Type)
    tmpl.Execute(w, map[string]interface{}{
        "Hostname": err.Hostname,
        "Message":  err.Message,
    })
}
```

### Store Interface Addition

```go
// internal/shell/store/store.go (addition)

type Store interface {
    // ... existing methods

    // GetDeploymentByDomain finds a deployment by its domain hostname.
    // Returns ErrNotFound if no deployment matches.
    GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error)

    // GetUsedProxyPorts returns all proxy ports in use on a node.
    GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error)
}
```

### SQLite Implementation

```go
// internal/shell/store/sqlite.go (addition)

func (s *SQLiteStore) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
    query := `
        SELECT id, name, template_id, template_version, customer_id, node_id,
               status, proxy_port, domains, containers, resources,
               created_at, updated_at, started_at, stopped_at
        FROM deployments
        WHERE json_extract(domains, '$[0].hostname') = ?
           OR json_extract(domains, '$[1].hostname') = ?
        LIMIT 1
    `

    var d deploymentRow
    if err := s.db.GetContext(ctx, &d, query, hostname, hostname); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("get deployment by domain: %w", err)
    }

    return d.toDomain()
}

func (s *SQLiteStore) GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error) {
    query := `
        SELECT proxy_port FROM deployments
        WHERE node_id = ? AND proxy_port IS NOT NULL AND status != 'deleted'
    `

    var ports []int
    if err := s.db.SelectContext(ctx, &ports, query, nodeID); err != nil {
        return nil, fmt.Errorf("get used proxy ports: %w", err)
    }

    return ports, nil
}
```

## Port Management

### Port Allocation (Pure)

```go
// internal/core/deployment/ports.go

// PortRange defines the available port range for deployments.
type PortRange struct {
    Start int // Inclusive, e.g., 30000
    End   int // Inclusive, e.g., 39999
}

// DefaultPortRange returns the default port range.
func DefaultPortRange() PortRange {
    return PortRange{Start: 30000, End: 39999}
}

// AllocatePort finds the first available port in the range.
// Pure function - takes used ports as input.
func AllocatePort(usedPorts []int, portRange PortRange) (int, error) {
    used := make(map[int]bool, len(usedPorts))
    for _, p := range usedPorts {
        used[p] = true
    }

    for port := portRange.Start; port <= portRange.End; port++ {
        if !used[port] {
            return port, nil
        }
    }

    return 0, errors.New("no available ports in range")
}

// ValidatePort checks if a port is within the allowed range.
func ValidatePort(port int, portRange PortRange) bool {
    return port >= portRange.Start && port <= portRange.End
}
```

### Container Port Binding

When creating a container, bind the service port to the allocated proxy port:

```go
// internal/shell/docker/orchestrator.go (modification)

func (o *Orchestrator) createContainer(ctx context.Context, spec ContainerSpec) error {
    // ... existing code

    // Bind container port to host proxy port
    portBindings := nat.PortMap{}
    if spec.ProxyPort > 0 && spec.ServicePort > 0 {
        containerPort := nat.Port(fmt.Sprintf("%d/tcp", spec.ServicePort))
        portBindings[containerPort] = []nat.PortBinding{
            {
                HostIP:   "127.0.0.1", // Only bind to localhost for security
                HostPort: fmt.Sprintf("%d", spec.ProxyPort),
            },
        }
    }

    hostConfig := &container.HostConfig{
        PortBindings: portBindings,
        // ... other config
    }

    // ... create container
}
```

## Database Migration

```sql
-- migrations/006_proxy_port.up.sql

-- Add proxy_port column to deployments
ALTER TABLE deployments ADD COLUMN proxy_port INTEGER;

-- Create index for domain lookup
-- SQLite doesn't support functional indexes directly, so we use a generated column
-- or query with json_extract (used in the query above)

-- For better performance, consider a separate domains table in the future
```

```sql
-- migrations/006_proxy_port.down.sql

ALTER TABLE deployments DROP COLUMN proxy_port;
```

## WebSocket Support

```go
// internal/shell/proxy/websocket.go

// isWebSocketRequest checks if the request is a WebSocket upgrade.
func isWebSocketRequest(r *http.Request) bool {
    return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// The standard httputil.ReverseProxy handles WebSocket upgrades automatically
// when the backend supports it. We just need to ensure:
// 1. Proper headers are preserved
// 2. Timeouts are appropriate for long-lived connections
```

## Error Pages

```html
<!-- internal/shell/proxy/templates/not_found.html -->
<!DOCTYPE html>
<html>
<head>
    <title>App Not Found</title>
    <style>
        body { font-family: system-ui; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #e74c3c; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>App Not Found</h1>
    <p>The app at <code>{{.Hostname}}</code> does not exist.</p>
    <p>If you just created this deployment, it may take a few moments to become available.</p>
    <p><a href="/">Return to homepage</a></p>
</body>
</html>
```

```html
<!-- internal/shell/proxy/templates/stopped.html -->
<!DOCTYPE html>
<html>
<head>
    <title>App Stopped</title>
    <style>
        body { font-family: system-ui; max-width: 600px; margin: 100px auto; padding: 20px; }
        h1 { color: #f39c12; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>App Stopped</h1>
    <p>The app at <code>{{.Hostname}}</code> is currently stopped.</p>
    <p>The owner can start it from their dashboard.</p>
    <p><a href="/">Return to homepage</a></p>
</body>
</html>
```

## Configuration

```yaml
# config.yaml
proxy:
  enabled: true
  address: "0.0.0.0:9091"
  base_domain: "apps.hoster.io"
  read_timeout: 30s
  write_timeout: 60s
  idle_timeout: 120s

  # Port range for container binding
  port_range:
    start: 30000
    end: 39999
```

## Metrics

```go
// internal/shell/proxy/metrics.go

var (
    proxyRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "hoster_proxy_requests_total",
            Help: "Total number of proxy requests",
        },
        []string{"status", "deployment_id"},
    )

    proxyRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "hoster_proxy_request_duration_seconds",
            Help:    "Proxy request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"deployment_id"},
    )

    proxyActiveConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "hoster_proxy_active_connections",
            Help: "Number of active proxy connections",
        },
    )
)
```

## Test Cases

### Unit Tests (internal/core/proxy/)

```go
// hostname_parser_test.go
func TestHostnameParser_Parse(t *testing.T) {
    parser := HostnameParser{BaseDomain: "apps.hoster.io"}

    tests := []struct {
        name     string
        hostname string
        wantSlug string
        wantOK   bool
    }{
        {"valid", "my-blog.apps.hoster.io", "my-blog", true},
        {"with port", "my-blog.apps.hoster.io:8080", "my-blog", true},
        {"nested subdomain", "api.my-blog.apps.hoster.io", "api.my-blog", true},
        {"wrong domain", "my-blog.other.io", "", false},
        {"base domain only", "apps.hoster.io", "", false},
        {"empty", "", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            slug, ok := parser.Parse(tt.hostname)
            assert.Equal(t, tt.wantSlug, slug)
            assert.Equal(t, tt.wantOK, ok)
        })
    }
}

// target_test.go
func TestProxyTarget_CanRoute(t *testing.T) {
    tests := []struct {
        name   string
        target ProxyTarget
        want   bool
    }{
        {"running with port", ProxyTarget{Status: "running", Port: 30001}, true},
        {"stopped", ProxyTarget{Status: "stopped", Port: 30001}, false},
        {"running no port", ProxyTarget{Status: "running", Port: 0}, false},
        {"pending", ProxyTarget{Status: "pending", Port: 30001}, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.want, tt.target.CanRoute())
        })
    }
}

// ports_test.go
func TestAllocatePort(t *testing.T) {
    portRange := PortRange{Start: 30000, End: 30005}

    tests := []struct {
        name      string
        usedPorts []int
        wantPort  int
        wantErr   bool
    }{
        {"empty", nil, 30000, false},
        {"some used", []int{30000, 30001}, 30002, false},
        {"all used", []int{30000, 30001, 30002, 30003, 30004, 30005}, 0, true},
        {"gaps", []int{30000, 30002}, 30001, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            port, err := AllocatePort(tt.usedPorts, portRange)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.wantPort, port)
            }
        })
    }
}
```

### Integration Tests (internal/shell/proxy/)

```go
// server_test.go
func TestProxyServer_ServeHTTP(t *testing.T) {
    // Setup mock store with test deployment
    store := &mockStore{
        deployments: map[string]*domain.Deployment{
            "my-blog.apps.test.io": {
                ID:        "depl_123",
                NodeID:    "local",
                ProxyPort: 30001,
                Status:    domain.DeploymentStatusRunning,
            },
        },
    }

    // Start a test backend server
    backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello from backend"))
    }))
    defer backend.Close()

    // The mock store should return the backend's port
    // ... configure appropriately

    server := NewProxyServer(ProxyConfig{
        BaseDomain: "apps.test.io",
    }, store, nil, slog.Default())

    // Test successful proxy
    t.Run("routes to running deployment", func(t *testing.T) {
        req := httptest.NewRequest("GET", "http://my-blog.apps.test.io/", nil)
        rec := httptest.NewRecorder()

        server.ServeHTTP(rec, req)

        assert.Equal(t, 200, rec.Code)
        assert.Contains(t, rec.Body.String(), "Hello from backend")
    })

    // Test not found
    t.Run("returns 404 for unknown hostname", func(t *testing.T) {
        req := httptest.NewRequest("GET", "http://unknown.apps.test.io/", nil)
        rec := httptest.NewRecorder()

        server.ServeHTTP(rec, req)

        assert.Equal(t, 404, rec.Code)
    })
}
```

## NOT Supported

- Custom domains (user brings their own domain)
- SSL termination (handled by APIGate)
- Load balancing (single container per deployment)
- Path-based routing (hostname only)
- Request/response modification
- Authentication at proxy level (handled by APIGate)

## Files to Create

```
internal/core/proxy/
├── target.go        # ProxyTarget type
├── target_test.go
├── resolver.go      # HostnameParser
├── resolver_test.go
├── errors.go        # ProxyError types
└── errors_test.go

internal/core/deployment/
├── ports.go         # Port allocation logic
└── ports_test.go

internal/shell/proxy/
├── server.go        # ProxyServer
├── server_test.go
├── middleware.go    # Logging, metrics
├── websocket.go     # WebSocket notes
├── metrics.go       # Prometheus metrics
└── templates/       # Error page templates
    ├── not_found.html
    ├── stopped.html
    └── unavailable.html

internal/shell/store/
├── store.go         # Add interface methods
└── sqlite.go        # Add implementations

migrations/
└── 006_proxy_port.up.sql
└── 006_proxy_port.down.sql
```

## Files to Modify

```
internal/shell/docker/orchestrator.go  # Port binding
internal/shell/api/resources/deployment.go  # Port allocation on create
cmd/hoster/config.go  # Proxy config
cmd/hoster/server.go  # Start proxy server
```
