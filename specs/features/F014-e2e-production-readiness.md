# F014: End-to-End Production Readiness

## Status
Draft - Requires Review

## Executive Summary

This spec defines the complete work required to transform Hoster from a working backend into a **fully self-service production platform** where non-technical users can:

1. **Discover** - Find the platform and understand what it offers
2. **Onboard** - Sign up, verify account, set up billing
3. **Deploy** - Browse marketplace, deploy apps with one click
4. **Manage** - Access running apps, view logs, start/stop
5. **Pay** - Be billed accurately for usage

**Critical Design Principle**:
- APIGate is the SINGLE entry point for ALL traffic
- NO third-party dependencies for core features (no Traefik, nginx, etc.)
- Routing to deployed apps is a CORE feature built into Hoster

---

## Current State Analysis

### What Exists ✅

| Component | Status | Notes |
|-----------|--------|-------|
| Backend API | ✅ Complete | 500+ tests, full CRUD |
| Docker orchestration | ✅ Complete | Create/start/stop/delete containers |
| Auth middleware | ✅ Complete | Header-based auth from APIGate |
| APIGate integration | ✅ Complete | X-User-ID header injection working |
| Frontend UI | ✅ Complete | React app, marketplace, dashboards |
| Domain generation | ✅ Complete | Auto-generates `{name}.apps.{domain}` |
| Worker nodes | ✅ Complete | Remote deployment support |
| Health checking | ✅ Complete | Background node monitoring |

### What's Missing ❌

| Component | Status | Impact |
|-----------|--------|--------|
| **App Proxy (routing)** | ❌ Missing | Deployed apps unreachable |
| **SSL/TLS termination** | ❌ Missing | No HTTPS, security issue |
| Frontend via APIGate | ❌ Missing | Frontend not integrated |
| User signup flow | ❌ Missing | Users can't self-register |
| Billing events | ❌ Partial | Usage not reported to APIGate |
| Landing page | ❌ Missing | No marketing/onboarding page |
| Documentation | ❌ Missing | Users don't know how to use |
| Plan limits enforcement | ❌ Partial | Limits received but not enforced |

---

## Architecture Overview

### Target Production Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              INTERNET (Users)                                     │
└─────────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         DNS PROVIDER (Cloudflare / Route53)                      │
│                                                                                   │
│   hoster.io              → APIGate IP                                            │
│   api.hoster.io          → APIGate IP                                            │
│   app.hoster.io          → APIGate IP                                            │
│   *.apps.hoster.io       → APIGate IP (or Hoster App Proxy)                     │
│                                                                                   │
└─────────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              APIGate (Port 8080)                                 │
│                                                                                   │
│  UNIFIED GATEWAY - ALL requests come through here                               │
│                                                                                   │
│  Routes:                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │  /                    → Landing page (static)                            │    │
│  │  /app/*               → Frontend SPA (static)                            │    │
│  │  /api/v1/*            → Hoster API (upstream: localhost:9090)           │    │
│  │  *.apps.hoster.io/*   → Hoster App Proxy (upstream: localhost:9091)     │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                                                                   │
│  Features:                                                                       │
│  • Authentication (API keys, sessions)                                          │
│  • Rate limiting                                                                 │
│  • Billing/metering                                                             │
│  • Header injection (X-User-ID, X-Plan-ID)                                     │
│  • SSL termination (HTTPS)                                                      │
│  • Static file serving                                                          │
│                                                                                   │
└─────────────────────────────────────────────────────────────────────────────────┘
         │                          │                          │
         │ API requests             │ App proxy requests       │
         ▼                          ▼                          │
┌────────────────────┐    ┌────────────────────┐              │
│  Hoster API        │    │  Hoster App Proxy  │              │
│  (Port 9090)       │    │  (Port 9091)       │              │
│                    │    │                    │              │
│  • Template CRUD   │    │  • Hostname lookup │              │
│  • Deployment CRUD │    │  • Route to container             │
│  • Orchestration   │    │  • WebSocket support              │
│  • Monitoring      │    │  • Health checks   │              │
│                    │    │                    │              │
└────────────────────┘    └────────────────────┘              │
         │                          │                          │
         └──────────────────────────┼──────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           DOCKER CONTAINERS                                       │
│                                                                                   │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│   │ User App 1  │  │ User App 2  │  │ User App 3  │  │ User App N  │           │
│   │             │  │             │  │             │  │             │           │
│   │ my-blog.    │  │ my-shop.    │  │ gitea.      │  │ ...         │           │
│   │ apps.       │  │ apps.       │  │ apps.       │  │             │           │
│   │ hoster.io   │  │ hoster.io   │  │ hoster.io   │  │             │           │
│   │             │  │             │  │             │  │             │           │
│   │ Port: 32001 │  │ Port: 32002 │  │ Port: 32003 │  │ Port: 3200N │           │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘           │
│                                                                                   │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Request Flows

**Flow 1: User visits landing page**
```
User → hoster.io → APIGate → Static HTML (landing page)
```

**Flow 2: User logs in / signs up**
```
User → hoster.io/login → APIGate → Built-in auth → Session cookie
```

**Flow 3: User browses marketplace (authenticated)**
```
User → app.hoster.io/marketplace → APIGate → Static SPA
SPA  → api.hoster.io/api/v1/templates → APIGate → (+X-User-ID) → Hoster API → JSON
```

**Flow 4: User deploys app**
```
User → api.hoster.io/api/v1/deployments → APIGate → Hoster API → Docker → Container
        + X-User-ID injected
        + Usage event recorded
        + Container gets assigned port (e.g., 32001)
        + Domain registered in database (my-blog.apps.hoster.io → 32001)
```

**Flow 5: User accesses deployed app** (CRITICAL NEW FEATURE)
```
User → my-blog.apps.hoster.io → APIGate → Hoster App Proxy
        │
        ▼
    App Proxy looks up "my-blog.apps.hoster.io" in database
        │
        ▼
    Finds: deployment "depl_xyz", container port 32001
        │
        ▼
    Proxies request to localhost:32001 (or remote node)
        │
        ▼
    Response returned to user
```

---

## Implementation Phases

### Phase A: Hoster App Proxy (CRITICAL PATH)

**Goal**: Built-in reverse proxy that routes requests to deployed containers.

#### A.1: App Proxy Server

**What**: New HTTP server in Hoster that handles app routing.

**Design**:
```go
// internal/shell/proxy/server.go

type AppProxyServer struct {
    store       store.Store
    docker      docker.Client
    nodePool    *docker.NodePool
    httpClient  *http.Client
    baseDomain  string  // e.g., "apps.hoster.io"
}

// ServeHTTP handles incoming app requests
func (s *AppProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Extract hostname from request
    hostname := r.Host  // e.g., "my-blog.apps.hoster.io"

    // 2. Look up deployment by domain
    deployment, err := s.store.GetDeploymentByDomain(ctx, hostname)
    if err != nil {
        http.Error(w, "App not found", 404)
        return
    }

    // 3. Check deployment is running
    if deployment.Status != "running" {
        // Serve friendly "app stopped" page
        s.serveStoppedPage(w, deployment)
        return
    }

    // 4. Get container address
    addr, err := s.getContainerAddress(deployment)
    if err != nil {
        http.Error(w, "App unavailable", 503)
        return
    }

    // 5. Proxy request to container
    proxy := httputil.NewSingleHostReverseProxy(addr)
    proxy.ServeHTTP(w, r)
}
```

**Port Assignment Strategy**:
```go
// Option 1: Dynamic port mapping (Docker assigns)
// Container runs on random port like 32768
// Hoster reads port from container inspection

// Option 2: Fixed port range (Hoster assigns)
// Hoster allocates ports from range 30000-39999
// Stores port in deployment record

// Recommendation: Option 2 for predictability
```

**Database Schema Addition**:
```sql
-- Add to deployments table
ALTER TABLE deployments ADD COLUMN proxy_port INTEGER;
ALTER TABLE deployments ADD COLUMN proxy_address TEXT;  -- For remote nodes

-- Index for fast domain lookup
CREATE INDEX idx_deployments_domain ON deployments(
    (json_extract(domains, '$[0].hostname'))
);
```

**Files to Create**:
- `internal/shell/proxy/server.go` - Main proxy server
- `internal/shell/proxy/server_test.go` - Tests
- `internal/shell/proxy/middleware.go` - Logging, metrics
- `internal/shell/proxy/websocket.go` - WebSocket support
- `cmd/hoster/proxy.go` - Proxy startup

**Configuration**:
```yaml
proxy:
  enabled: true
  address: "0.0.0.0:9091"
  base_domain: "apps.hoster.io"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  websocket_enabled: true
```

**Acceptance Criteria**:
- [ ] Proxy server starts on configured port
- [ ] Routes requests based on hostname
- [ ] Handles multiple concurrent connections
- [ ] Supports WebSocket upgrade
- [ ] Returns friendly error for stopped/missing apps
- [ ] Works with local Docker
- [ ] Works with remote nodes (via SSH tunnel or direct)

#### A.2: Port Management

**What**: Allocate and track ports for deployed containers.

**Design**:
```go
// internal/core/deployment/ports.go

// PortRange defines the range of ports available for deployments
type PortRange struct {
    Start int  // e.g., 30000
    End   int  // e.g., 39999
}

// AllocatePort finds the next available port
func AllocatePort(usedPorts []int, portRange PortRange) (int, error) {
    used := make(map[int]bool)
    for _, p := range usedPorts {
        used[p] = true
    }

    for port := portRange.Start; port <= portRange.End; port++ {
        if !used[port] {
            return port, nil
        }
    }

    return 0, errors.New("no available ports")
}
```

**Store Interface Addition**:
```go
type Store interface {
    // ... existing methods

    // Port management
    GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error)
    GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error)
}
```

**Files**:
- `internal/core/deployment/ports.go` - Port allocation (pure)
- `internal/core/deployment/ports_test.go` - Tests

**Acceptance Criteria**:
- [ ] Ports allocated from configured range
- [ ] No port collisions
- [ ] Ports released when deployment deleted

#### A.3: Container Port Exposure

**What**: Ensure deployed containers expose their service port.

**Current State**: Containers run with internal networking, ports not exposed.

**Change**: Expose primary service port to host.

**Orchestrator Changes**:
```go
// internal/shell/docker/orchestrator.go

func (o *Orchestrator) createContainer(...) error {
    // Get allocated proxy port
    proxyPort := deployment.ProxyPort

    // Container config with port binding
    hostConfig := &container.HostConfig{
        PortBindings: nat.PortMap{
            nat.Port(fmt.Sprintf("%d/tcp", servicePort)): []nat.PortBinding{
                {HostIP: "127.0.0.1", HostPort: fmt.Sprintf("%d", proxyPort)},
            },
        },
    }

    // ...
}
```

**Files to Modify**:
- `internal/shell/docker/orchestrator.go` - Port binding
- `internal/shell/api/resources/deployment.go` - Port allocation

**Acceptance Criteria**:
- [ ] Primary service port exposed to host
- [ ] Port bound to localhost only (security)
- [ ] Port stored in deployment record

#### A.4: Remote Node Routing

**What**: Route to containers on remote worker nodes.

**Options**:

**Option A: SSH Tunnel (Secure)**
```
App Proxy → SSH tunnel to node → localhost:port on node
```

**Option B: Direct Connection (Faster)**
```
App Proxy → http://node-ip:port
Requires: Node firewall allows connection from Hoster
```

**Option C: Hybrid**
- Local node: Direct to localhost
- Remote node: SSH tunnel

**Recommendation**: Option C (Hybrid)

**Implementation**:
```go
func (s *AppProxyServer) getContainerAddress(depl *domain.Deployment) (*url.URL, error) {
    if depl.NodeID == "local" {
        return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", depl.ProxyPort))
    }

    // Get SSH tunnel for remote node
    tunnel, err := s.nodePool.GetTunnel(depl.NodeID, depl.ProxyPort)
    if err != nil {
        return nil, err
    }

    return url.Parse(fmt.Sprintf("http://127.0.0.1:%d", tunnel.LocalPort))
}
```

**Files**:
- `internal/shell/docker/tunnel.go` - SSH tunnel management
- `internal/shell/docker/node_pool.go` - Add tunnel support

**Acceptance Criteria**:
- [ ] Local containers accessible via proxy
- [ ] Remote containers accessible via SSH tunnel
- [ ] Tunnel established on-demand
- [ ] Tunnel cleanup on idle

---

### Phase B: APIGate Integration (Full)

**Goal**: APIGate as the single entry point, serving everything.

#### B.1: Route Configuration

**What**: Configure APIGate routes for all traffic.

**Required Routes**:
```
1. Landing page:     GET /           → static files
2. Frontend SPA:     GET /app/*      → static files (SPA mode)
3. API:              ALL /api/v1/*   → upstream localhost:9090
4. App Proxy:        ALL *.apps.*    → upstream localhost:9091
5. Auth:             ALL /auth/*     → APIGate built-in
6. Portal:           ALL /portal/*   → APIGate built-in
```

**APIGate Route Configuration**:
```json
{
  "routes": [
    {
      "id": "landing",
      "name": "Landing Page",
      "path_pattern": "/",
      "match_type": "exact",
      "type": "static",
      "static_root": "/var/www/hoster/public"
    },
    {
      "id": "frontend",
      "name": "Frontend SPA",
      "path_pattern": "/app/*",
      "match_type": "prefix",
      "type": "static",
      "static_root": "/var/www/hoster/app",
      "spa_mode": true
    },
    {
      "id": "api",
      "name": "Hoster API",
      "path_pattern": "/api/v1/*",
      "match_type": "prefix",
      "upstream_id": "hoster-api",
      "request_transform": {
        "set_headers": {
          "X-User-ID": "userID",
          "X-Plan-ID": "planID",
          "X-Key-ID": "keyID"
        }
      }
    },
    {
      "id": "app-proxy",
      "name": "App Proxy",
      "path_pattern": "/*",
      "match_type": "prefix",
      "host_pattern": "*.apps.hoster.io",
      "upstream_id": "hoster-proxy"
    }
  ],
  "upstreams": [
    {
      "id": "hoster-api",
      "name": "Hoster API",
      "base_url": "http://localhost:9090"
    },
    {
      "id": "hoster-proxy",
      "name": "Hoster App Proxy",
      "base_url": "http://localhost:9091"
    }
  ]
}
```

**Question for User**: Does APIGate support:
1. Static file serving with SPA mode?
2. Host-based routing (*.apps.domain)?
3. If not, what's the alternative?

**Acceptance Criteria**:
- [ ] Landing page served at root
- [ ] SPA served at /app/*
- [ ] API proxied with auth headers
- [ ] App proxy reached via subdomains

#### B.2: Frontend Build & Deployment

**What**: Build frontend for production, deploy to APIGate.

**Build Configuration**:
```typescript
// web/vite.config.ts
export default defineConfig({
  base: '/app/',  // Served under /app/
  build: {
    outDir: 'dist',
  },
  define: {
    'import.meta.env.VITE_API_URL': '"/api/v1"',
  },
});
```

**Deployment Script**:
```bash
#!/bin/bash
# scripts/deploy-frontend.sh

cd web
npm ci
npm run build

# Copy to APIGate static directory
cp -r dist/* /var/www/hoster/app/
```

**Files**:
- `web/vite.config.ts` - Update for production
- `scripts/deploy-frontend.sh` - Deployment script

**Acceptance Criteria**:
- [ ] Frontend builds successfully
- [ ] Works when served from /app/
- [ ] API calls work via /api/v1
- [ ] SPA routing works

#### B.3: Authentication Flow

**What**: Full auth flow using APIGate's built-in auth.

**Pages to Create**:
- `LoginPage` - Email/password login
- `SignupPage` - Registration form
- `ForgotPasswordPage` - Password reset request
- `ResetPasswordPage` - Set new password

**Auth Store Update**:
```typescript
// web/src/stores/authStore.ts

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  // Actions
  checkAuth: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  signup: (data: SignupData) => Promise<void>;
  logout: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,

  checkAuth: async () => {
    try {
      const response = await fetch('/auth/me', { credentials: 'include' });
      if (response.ok) {
        const user = await response.json();
        set({ user, isAuthenticated: true, isLoading: false });
      } else {
        set({ user: null, isAuthenticated: false, isLoading: false });
      }
    } catch {
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },

  login: async (email, password) => {
    const response = await fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
      credentials: 'include',
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message);
    }

    await get().checkAuth();
  },

  // ... signup, logout similar
}));
```

**Protected Route Component**:
```typescript
// web/src/components/auth/ProtectedRoute.tsx

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuthStore();
  const location = useLocation();

  if (isLoading) {
    return <LoadingSpinner />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/app/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}
```

**Files**:
- `web/src/pages/auth/LoginPage.tsx`
- `web/src/pages/auth/SignupPage.tsx`
- `web/src/pages/auth/ForgotPasswordPage.tsx`
- `web/src/pages/auth/ResetPasswordPage.tsx`
- `web/src/stores/authStore.ts` - Update
- `web/src/components/auth/ProtectedRoute.tsx`
- `web/src/components/auth/AuthProvider.tsx`

**APIGate Auth Endpoints** (verify these exist):
```
POST   /auth/register     { email, password, name }
POST   /auth/login        { email, password }
POST   /auth/logout
GET    /auth/me           → { id, email, name, plan_id }
POST   /auth/forgot       { email }
POST   /auth/reset        { token, password }
```

**Acceptance Criteria**:
- [ ] User can sign up
- [ ] User can log in
- [ ] Session persists across page refresh
- [ ] Protected routes redirect to login
- [ ] Logout clears session
- [ ] Password reset flow works

---

### Phase C: Billing Integration

**Goal**: Users are billed for actual usage.

#### C.1: Usage Event Reporting

**What**: Report deployment events to APIGate for billing.

**Events**:
| Event | Trigger | Billing Impact |
|-------|---------|----------------|
| `deployment.created` | Deployment created | Subscription start |
| `deployment.started` | Start command | Compute billing start |
| `deployment.stopped` | Stop command | Compute billing pause |
| `deployment.deleted` | Deletion | Subscription end |

**Implementation**:
```go
// internal/shell/billing/client.go

type APIGateBillingClient struct {
    baseURL    string
    httpClient *http.Client
    serviceKey string  // Hoster's API key for APIGate
}

func (c *APIGateBillingClient) MeterUsage(ctx context.Context, event MeterEvent) error {
    body, _ := json.Marshal(event)

    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/meter", bytes.NewReader(body))
    req.Header.Set("X-API-Key", c.serviceKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    // ... handle response
}
```

**Deployment Resource Integration**:
```go
// internal/shell/api/resources/deployment.go

func (r *DeploymentResource) Create(...) {
    // ... create deployment

    // Record usage event
    event := domain.MeterEvent{
        ID:           uuid.New().String(),
        UserID:       authCtx.UserID,
        EventType:    "deployment.created",
        ResourceID:   deployment.ID,
        ResourceType: "deployment",
        Metadata: map[string]string{
            "template_id": deployment.TemplateID,
            "plan_id":     authCtx.PlanID,
        },
        Timestamp: time.Now(),
    }

    if err := r.billingClient.MeterUsage(ctx, event); err != nil {
        slog.Error("failed to report usage", "error", err)
        // Don't fail the request - usage is best-effort
    }
}
```

**Files**:
- `internal/shell/billing/client.go`
- `internal/shell/billing/client_test.go`
- `internal/core/domain/usage.go` - MeterEvent type

**Acceptance Criteria**:
- [ ] Events sent on deployment create/start/stop/delete
- [ ] Events include user ID, resource ID, metadata
- [ ] Failed events don't break deployment operations

#### C.2: Plan Limits Enforcement

**What**: Prevent exceeding plan limits.

**Validation**:
```go
// internal/core/limits/validation.go

func ValidateDeploymentCreation(
    limits auth.PlanLimits,
    currentCount int,
) ValidationResult {
    if currentCount >= limits.MaxDeployments {
        return ValidationResult{
            Allowed: false,
            Reason:  fmt.Sprintf("Deployment limit reached (%d/%d). Upgrade your plan.",
                currentCount, limits.MaxDeployments),
        }
    }
    return ValidationResult{Allowed: true}
}
```

**Integration**:
```go
func (r *DeploymentResource) Create(...) {
    authCtx := auth.FromContext(ctx)

    // Get current deployment count
    count, _ := r.store.CountDeploymentsByCustomer(ctx, authCtx.UserID)

    // Validate
    result := limits.ValidateDeploymentCreation(authCtx.PlanLimits, count)
    if !result.Allowed {
        return nil, api2go.NewHTTPError(nil, result.Reason, http.StatusForbidden)
    }

    // ... proceed with creation
}
```

**Files**:
- `internal/core/limits/validation.go`
- `internal/core/limits/validation_test.go`

**Acceptance Criteria**:
- [ ] Can't create deployment over limit
- [ ] Clear error message with current/max
- [ ] Suggests upgrade

---

### Phase D: Landing Page & Documentation

**Goal**: Users can discover and learn the platform.

#### D.1: Landing Page

**What**: Marketing page for unauthenticated visitors.

**Sections**:
1. **Hero**: "Deploy apps in seconds" + CTA
2. **Features**: One-click deploy, auto-SSL, monitoring
3. **How it works**: 3-step visual
4. **Pricing**: Plan comparison
5. **Testimonials**: (if available)
6. **Footer**: Links, legal

**Implementation**: Static HTML + CSS (no framework needed)

**Files**:
- `web/public/index.html` - Landing page
- `web/public/pricing.html` - Pricing page
- `web/public/css/landing.css` - Styles
- `web/public/images/` - Assets

**Acceptance Criteria**:
- [ ] Professional appearance
- [ ] Mobile responsive
- [ ] CTA links to signup
- [ ] Fast load time (<2s)

#### D.2: User Documentation

**What**: Self-service documentation.

**Structure**:
```
docs/
├── index.md                    # Welcome + Quick start
├── getting-started/
│   ├── signup.md              # Account creation
│   ├── first-deployment.md    # Deploy first app
│   └── managing-apps.md       # Start/stop/delete
├── marketplace/
│   ├── browsing.md            # Finding templates
│   └── deploying.md           # Deployment process
├── creating-templates/
│   ├── basics.md              # Docker compose intro
│   ├── variables.md           # Configuration options
│   └── publishing.md          # Submit to marketplace
├── billing/
│   ├── how-it-works.md        # Usage-based pricing
│   └── plans.md               # Plan comparison
└── api/
    └── reference.md           # Link to OpenAPI docs
```

**Implementation Options**:
1. **Static Markdown** → HTML via build script
2. **Docusaurus** (overkill for now)
3. **MkDocs** (Python, simple)

**Recommendation**: Static build script for now.

**Files**:
- `docs/**/*.md` - Documentation content
- `scripts/build-docs.sh` - Build script

**Acceptance Criteria**:
- [ ] Accessible at /docs
- [ ] Searchable (basic)
- [ ] Mobile friendly
- [ ] Quick start < 5 min read

---

### Phase E: Operational Readiness

**Goal**: Platform can be operated reliably.

#### E.1: Health Endpoints

**What**: Health check endpoints for monitoring.

**Endpoints**:
```
GET /health          → {"status": "ok", "version": "1.0.0"}
GET /health/ready    → {"status": "ready"} (dependencies checked)
GET /health/live     → {"status": "live"} (process alive)
```

**Files**:
- `internal/shell/api/health.go`

#### E.2: Metrics Endpoint

**What**: Prometheus metrics for observability.

**Metrics**:
- `hoster_deployments_total` - Counter by status
- `hoster_api_requests_total` - Counter by endpoint, status
- `hoster_api_request_duration_seconds` - Histogram
- `hoster_proxy_requests_total` - Counter
- `hoster_proxy_request_duration_seconds` - Histogram

**Files**:
- `internal/shell/api/metrics.go`
- `internal/shell/proxy/metrics.go`

#### E.3: Structured Logging

**What**: JSON logs for aggregation.

**Already Done**: Using `log/slog` with JSON output.

**Enhancement**: Add request ID correlation.

---

## Implementation Priority

### Critical Path (Must Have for MVP)

```
1. Phase A.1: App Proxy Server        ← Deployed apps accessible
2. Phase A.2: Port Management         ← Containers get ports
3. Phase A.3: Container Port Exposure ← Ports bound
4. Phase B.1: APIGate Route Config    ← Everything through gateway
5. Phase B.3: Auth Flow               ← Users can sign up/login
```

### High Priority (Should Have)

```
6. Phase B.2: Frontend Deployment     ← SPA served properly
7. Phase C.1: Usage Events            ← Billing works
8. Phase D.1: Landing Page            ← Users can discover
```

### Medium Priority (Nice to Have)

```
9.  Phase A.4: Remote Node Routing    ← Scale beyond single node
10. Phase C.2: Plan Limits            ← Prevent abuse
11. Phase D.2: Documentation          ← Self-service support
12. Phase E.*: Operations             ← Production monitoring
```

---

## Open Questions for User

1. **APIGate Static Files**: Does APIGate support serving static files with SPA mode? If not, do we need nginx/caddy sidecar?

2. **APIGate Host Routing**: Does APIGate support routing based on `Host` header (for `*.apps.domain`)? Or does app proxy need separate port/domain?

3. **APIGate Auth Flow**: Confirm the auth endpoints exist:
   - POST /auth/register
   - POST /auth/login
   - GET /auth/me
   - POST /auth/logout

4. **SSL Termination**: Does APIGate handle SSL? Or do we need separate cert management?

5. **Domain**: What's the actual domain? `hoster.io`? Need to configure accordingly.

6. **Billing API**: What's the exact APIGate endpoint for metering usage?

---

## NOT Supported (Initial Release)

- Custom domains for deployments
- Multi-region deployment
- Auto-scaling
- Database backups
- SSH access to containers
- Log export/download
- Multi-user teams/organizations
- Template versioning
- Deployment rollback
- Blue-green deployments

---

## Files Summary

### New Files

**Backend**:
- `internal/shell/proxy/server.go` - App proxy server
- `internal/shell/proxy/server_test.go`
- `internal/shell/proxy/middleware.go`
- `internal/shell/proxy/websocket.go`
- `internal/core/deployment/ports.go` - Port allocation
- `internal/core/deployment/ports_test.go`
- `internal/shell/billing/client.go` - APIGate billing
- `internal/shell/billing/client_test.go`
- `internal/core/limits/validation.go` - Plan limits
- `internal/core/limits/validation_test.go`
- `cmd/hoster/proxy.go` - Proxy startup

**Frontend**:
- `web/src/pages/auth/LoginPage.tsx`
- `web/src/pages/auth/SignupPage.tsx`
- `web/src/pages/auth/ForgotPasswordPage.tsx`
- `web/src/pages/auth/ResetPasswordPage.tsx`
- `web/src/components/auth/ProtectedRoute.tsx`
- `web/src/components/auth/AuthProvider.tsx`
- `web/public/index.html` - Landing page
- `web/public/pricing.html`
- `web/public/css/landing.css`

**Documentation**:
- `docs/**/*.md`

**Configuration**:
- `config/production.yml.example`
- `scripts/deploy-frontend.sh`
- `scripts/build-docs.sh`

### Modified Files

- `internal/shell/docker/orchestrator.go` - Port binding
- `internal/shell/store/store.go` - New interface methods
- `internal/shell/store/sqlite.go` - Implement new methods
- `internal/shell/api/resources/deployment.go` - Port allocation, billing
- `web/src/stores/authStore.ts` - Full auth flow
- `web/vite.config.ts` - Production config
- `cmd/hoster/server.go` - Start proxy server
- `cmd/hoster/config.go` - Add proxy config
