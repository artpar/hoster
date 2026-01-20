# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 20, 2026)

### Status: ✅ DEPLOYED TO PRODUCTION - emptychair.dev

**Production Deployment:**
- **URL**: https://emptychair.dev
- **Server**: AWS EC2 (3.82.3.209)
- **APIGate**: Handling TLS via ACME (auto-cert from Let's Encrypt)
- **Hoster**: Running as systemd service

**CI/CD Status:**
- **GitHub Actions CI**: ✅ Passing (test, build, vet jobs)
- **GitHub Releases**: ✅ v0.1.0 released with Linux amd64 binary
- **Deployment method**: Download from GitHub releases (no server-side build needed)

**Implementation Status:**
- MVP complete (core deployment loop works)
- All backend tests passing (500+ tests)
- All frontend components built and working
- **Production deployed with automatic SSL certificates** ✅
- **Billing Integration: WORKING END-TO-END**
- **CI/CD Pipeline: COMPLETE** ✅

**What's Working:**
- User signup/login via APIGate portal
- Template creation and management
- Deployment creation, start, stop
- App Proxy routing (host-based wildcard)
- Direct container access via allocated ports
- **Billing events reported to APIGate /api/v1/meter** ✅
- **Usage event storage and retrieval** ✅
- **ACME auto-cert working** ✅ (Issue #31, #32 fixed)

**APIGate Issues: ALL RESOLVED ✅**
- All 9 issues (#20-#28) have been fixed by the APIGate team
- Issues #29-#32 (TLS/ACME) also resolved
- Full integration is now possible without workarounds
- Auto-registration, service accounts, public routes all working
- ACME automatic certificate management working

### Production Management (emptychair.dev)

**Deployment via Makefile (RECOMMENDED):**
```bash
cd deploy/local
make deploy-release                    # Deploy latest release from GitHub
make deploy-release VERSION=v0.1.0     # Deploy specific version
```

**Server Management via Makefile:**
```bash
cd deploy/local
make status           # Show service status
make logs             # Tail all logs
make logs-apigate     # Tail APIGate logs only
make logs-hoster      # Tail Hoster logs only
make restart          # Restart both services
make shell            # SSH into server
make settings         # Show APIGate settings
make certs            # Show stored certificates
```

**Production Architecture:**
```
Client → emptychair.dev:443 (APIGate/ACME) → Portal/API
                                          → Hoster:8080
                                          → App Proxy:9091 → Containers
```

**Service Files:**
- APIGate: `/etc/systemd/system/apigate.service` (ACME mode, ports 80/443)
- Hoster: `/etc/systemd/system/hoster.service` (port 8080)
- Hoster env: `/etc/hoster/.env`
- APIGate DB: `/var/lib/apigate/apigate.db`
- Hoster DB: `/var/lib/hoster/hoster.db`

---

### E2E Test Environment State (Local)

**Running Services (may need restart):**
```
APIGate:    http://localhost:8082  (data: /tmp/apigate-data/)
Hoster API: http://localhost:8080  (data: ./data/)
App Proxy:  http://localhost:9091  (base: apps.localhost)
Frontend:   http://localhost:5174  (or 5173 if available)
```

**Test Data Created:**
```
User:       testuser@example.com (ID: 85257577-f230-4ebc-8370-f983cea27085)
API Key:    ak_d3df507720aaf9944e5b6248e6d0a8e1cb53aa2946031006bbcf287cb9fd5ed0
Template:   tmpl_2b6ae7fb (Simple Nginx)
Deployment: depl_2fb4c5e7 (running on port 30000)
```

**Database Files:**
- APIGate: `/tmp/apigate-data/apigate.db`
- Hoster: `./data/hoster.db`

### Commands to Restart Environment

```bash
# 1. Download latest APIGate (if needed)
gh run download 21140775848 --repo artpar/apigate --name apigate-darwin-arm64 --dir /tmp/apigate-new
cp /tmp/apigate-new/apigate-darwin-arm64 /tmp/apigate-darwin-arm64
chmod +x /tmp/apigate-darwin-arm64

# 2. Start APIGate (in background) - NOTE: new command syntax
APIGATE_DATABASE_DSN=/tmp/apigate-data/apigate.db \
APIGATE_SERVER_PORT=8082 \
nohup /tmp/apigate-darwin-arm64 serve > /tmp/apigate.log 2>&1 &

# 3. Build and start Hoster (in background)
make build
HOSTER_BILLING_ENABLED=true \
HOSTER_BILLING_APIGATE_URL=http://localhost:8082 \
HOSTER_APIGATE_ENABLED=true \
HOSTER_APIGATE_URL=http://localhost:8082 \
HOSTER_APP_PROXY_ENABLED=true \
HOSTER_APP_PROXY_ADDRESS=0.0.0.0:9091 \
HOSTER_APP_PROXY_BASE_DOMAIN=apps.localhost \
nohup ./bin/hoster > /tmp/hoster.log 2>&1 &

# 4. Start frontend (in foreground or background)
cd web && npm run dev
```

### Previous Implementation Completed:
- Phase -1 (ADR & Spec Updates) COMPLETE
- Phase 0 (API Layer Migration) COMPLETE
- Phase 1 (APIGate Auth Integration) COMPLETE
- Phase 2 (Billing Integration) COMPLETE
- Phase 3 (Monitoring Backend) COMPLETE
- Phase 4 (Frontend Foundation) COMPLETE
- Phase 5 (Frontend Views) COMPLETE
- Phase 6 Integration bug fixes COMPLETE
- Creator Worker Nodes Feature - ALL 7 PHASES COMPLETE
- App Proxy - Built-in HTTP routing COMPLETE
- F014 Phase A (App Proxy) COMPLETE
- F014 Phase B (APIGate Integration) COMPLETE
- Billing Client Updated to JSON:API Format COMPLETE

**Frontend Build Status:**
```
dist/index.html                   0.54 kB
dist/assets/index-*.css          23.47 kB (gzip: 5.02 kB)
dist/assets/index-*.js          383.96 kB (gzip: 114.21 kB)
```

**All UI Components Tested & Working:**
- Marketplace page with search/sort/filter
- Template detail page with pricing
- Deploy dialog for creating deployments
- Deployment detail page with monitoring tabs
- Creator dashboard for template management
- **Nodes tab for worker node management**

**Creator Worker Nodes Feature Progress:**
- Phase 1 (Domain Model & Scheduler): COMPLETE
- Phase 2 (Database Layer): COMPLETE
- Phase 3 (SSH Docker Client via Minion): COMPLETE
- Phase 4 (Scheduler Integration): COMPLETE
- Phase 5 (Node API Resource): COMPLETE
- Phase 6 (Frontend Nodes Tab): COMPLETE
- Phase 7 (Health Checker Worker): COMPLETE

**Creator Worker Nodes Feature: FULLY COMPLETE**

All phases of the Creator Worker Nodes feature are now implemented. The feature includes:
- Node and SSH Key domain models with full validation
- Database layer with encrypted SSH key storage (AES-256-GCM)
- SSH-based Docker client via the minion protocol
- Intelligent scheduler for node selection based on capabilities and capacity
- JSON:API resources for nodes and SSH keys
- Frontend UI for node management in Creator Dashboard
- Background health checker worker for periodic node monitoring

**App Proxy Feature: FULLY COMPLETE**

Built-in HTTP reverse proxy for routing requests to deployed containers. No external dependencies (no Traefik, nginx, etc.). The feature includes:
- Core proxy types (`internal/core/proxy/`): ProxyTarget, HostnameParser, ProxyError, PortRange
- Shell proxy server (`internal/shell/proxy/`): HTTP server using `net/http/httputil.ReverseProxy`
- Error page templates: not_found.html, stopped.html, unavailable.html
- Database migration 006: Added `proxy_port` column to deployments
- Store methods: `GetDeploymentByDomain`, `GetUsedProxyPorts`
- Port allocation: Allocates ports from range 30000-39999 on deployment start
- Container port binding: Binds primary service port to proxy port on localhost
- Configuration: `ProxyConfig` with host, port, base_domain, timeouts
- Default proxy address: `0.0.0.0:9091`

**Proxy Architecture:**
```
User Request → APIGate (8080) → App Proxy (9091) → Container (30000-39999)
                                    ↓
                              DB Lookup by hostname
                              (my-app.apps.localhost → deployment → port)
```

---

## LAST SESSION SUMMARY (January 20, 2026)

### What Was Accomplished: CI/CD Pipeline + APIGate Admin API Fix

This session fixed the APIGate integration bug and set up a complete CI/CD pipeline for Hoster.

**Key Accomplishments:**

1. **Fixed APIGate Admin API endpoint bug** - Hoster was calling `/api/upstreams` but APIGate uses `/admin/upstreams`
   - Updated `internal/shell/apigate/client.go` to use `/admin/` prefix
   - Updated tests in `client_test.go` and `registrar_test.go`

2. **Set up GitHub Actions CI/CD** - Following APIGate's patterns:
   - `.github/workflows/ci.yml` - Test, Build, Vet jobs on push/PR
   - `.github/workflows/release.yml` - Build releases on version tags

3. **Created v0.1.0 release** - First official release with Linux amd64 binary

4. **Updated Makefile** - Added `deploy-release` target to download from GitHub releases

5. **Deployed to production** - v0.1.0 running on emptychair.dev

**Files Modified/Created:**
```
internal/shell/apigate/client.go          # Fixed /api/ → /admin/
internal/shell/apigate/client_test.go     # Updated test expectations
internal/shell/apigate/registrar_test.go  # Updated test mocks
.github/workflows/ci.yml                  # NEW - CI workflow
.github/workflows/release.yml             # NEW - Release workflow
deploy/local/Makefile                     # Added deploy-release target
```

---

## PREVIOUS SESSION SUMMARY (January 19, 2026)

### What Was Accomplished: Billing Integration Now Working!

This session verified that the critical APIGate issues (#27 and #28) were fixed, updated APIGate, and tested the complete billing flow end-to-end.

**Key Accomplishments:**

1. **Downloaded updated APIGate binary** with metering fix (commit da36cc7)
2. **Verified metering endpoint works** - `POST /api/v1/meter` accepts events
3. **Confirmed billing events flowing** - Hoster successfully reports deployment.created and deployment.started events
4. **Events stored in APIGate** - Usage events retrievable via `GET /api/v1/meter?user_id=...`

**Verified Working End-to-End:**
- Deployment creates usage event → Hoster queues event → Reporter sends to APIGate → Event stored → Event retrievable

**Test Data:**
```
Events in APIGate for dev-user:
- evt_20260119190805_aaaaaaaa: deployment.created
- evt_20260119190829_iiiaaaaa: deployment.started
```

**All Components Now Working:**
- APIGate server: 8082 (updated binary with metering fix)
- Hoster API: 8080 (billing reporter active)
- App Proxy: 9091
- Frontend dev: 5173/5174
- User auth via portal
- Template creation/deployment
- Container management
- Host-based wildcard routing
- **Billing events to APIGate** ✅

**APIGate Issues Resolved:**
- Issue #27 (Metering API) - FIXED
- Issue #28 (API key generation) - FIXED

---

## APIGate Issues Reference

Issues filed during E2E testing at https://github.com/artpar/apigate/issues:

| # | Title | Status | Notes |
|---|-------|--------|-------|
| 20 | Admin API auth for service integration | **FIXED** | ✅ Resolved |
| 21 | UI missing host_pattern fields | **FIXED** | ✅ Resolved |
| 22 | Public routes needed | **FIXED** | ✅ Resolved |
| 23 | Env var naming inconsistency | **FIXED** | ✅ Resolved |
| 24 | Hot reload request | **FIXED** | ✅ Already exists (30s interval) |
| 25 | Service accounts | **FIXED** | ✅ Resolved |
| 26 | Portal API endpoints require API key | **FIXED** | ✅ Resolved |
| 27 | Metering API not implemented | **FIXED** | ✅ Billing works! |
| 28 | REST API keys fail validation | **FIXED** | ✅ Resolved |

**All APIGate issues have been resolved!** Full integration is now possible.

**To check latest status:**
```bash
gh issue list --repo artpar/apigate --state all --limit 20
```

---

## PREVIOUS SESSION SUMMARY (January 19, 2026)

### What Was Accomplished: Automated APIGate Integration + Operational Readiness

This session completed the automated APIGate integration (central to Hoster deployment) and verified operational readiness features.

**Key Accomplishment: APIGate is Central, Not Optional**

Implemented automated route registration on Hoster startup - no manual configuration required. When Hoster starts, it automatically registers with APIGate:
- Creates upstream for app proxy (`hoster-app-proxy`)
- Creates wildcard route (`*.apps.localhost` → app proxy)
- Optionally registers Hoster API route

**1. New APIGate Integration Package:**

Created `internal/shell/apigate/` with:
- `client.go` - APIGate admin API client (CRUD for upstreams/routes)
- `registrar.go` - Automatic registration on startup
- `client_test.go` - 13 tests for client
- `registrar_test.go` - 11 tests for registrar

**2. Configuration Updates (`cmd/hoster/config.go`):**

Added centralized `APIGateConfig`:
```go
type APIGateConfig struct {
    URL          string `mapstructure:"url"`           // e.g., "http://localhost:8082"
    AdminKey     string `mapstructure:"admin_key"`     // Admin API key
    AutoRegister bool   `mapstructure:"auto_register"` // Auto-register on startup
}
```

Defaults:
- `apigate.url`: `http://localhost:8082`
- `apigate.auto_register`: `true`

**3. Server Integration (`cmd/hoster/server.go`):**

- Registrar called on startup in `Start()` method
- Billing now uses centralized APIGate URL (with fallback for backward compatibility)
- Non-blocking registration (logs error but continues startup if APIGate unavailable)

**4. Operational Readiness Verified:**

- Health endpoints already exist: `/health`, `/health/live`, `/health/ready`
- Graceful shutdown already implemented with configurable timeout
- Signal handling (SIGINT, SIGTERM) already works

**5. Files Created/Modified:**

```
internal/shell/apigate/client.go         # NEW - APIGate admin client
internal/shell/apigate/registrar.go      # NEW - Auto-registration
internal/shell/apigate/client_test.go    # NEW - 13 tests
internal/shell/apigate/registrar_test.go # NEW - 11 tests
cmd/hoster/config.go                     # Added APIGateConfig
cmd/hoster/server.go                     # Integrated registrar + billing URL
```

**6. All Tests Pass:** 524+ tests (24 new apigate tests)

**7. Environment Variables:**

```bash
# APIGate configuration (central to Hoster)
HOSTER_APIGATE_URL=http://localhost:8082      # APIGate base URL
HOSTER_APIGATE_ADMIN_KEY=your-admin-key       # Admin API key for registration
HOSTER_APIGATE_AUTO_REGISTER=true             # Enable auto-registration (default)

# Billing (uses APIGate URL by default)
HOSTER_BILLING_ENABLED=true
HOSTER_BILLING_API_KEY=your-billing-key       # Or uses APIGATE_ADMIN_KEY as fallback
```

See plan file at `/Users/artpar/.claude/plans/partitioned-juggling-giraffe.md` for full details

---

## SUGGESTED NEXT STEPS

### Priority 1: Test Full User Journey (Billing Now Works!)

The billing flow is now working. Test the complete user journey:
1. New user signs up at `http://localhost:8082/portal/signup`
2. User creates API key in portal
3. User browses marketplace at `http://localhost:5174`
4. User deploys nginx template
5. User accesses app at `http://my-app.apps.localhost:9091`
6. Billing events are recorded in APIGate
7. User sees usage in portal

**Verify billing events:**
```bash
# Check events for a user
curl "http://localhost:8082/api/v1/meter" \
  --get --data-urlencode "user_id=USER_ID_HERE" \
  -H "X-API-Key: ak_d3df507720aaf9944e5b6248e6d0a8e1cb53aa2946031006bbcf287cb9fd5ed0" \
  | jq '.data'
```

### Priority 2: Payment Flow Testing
- Configure APIGate with Stripe test keys
- Test Stripe checkout flow end-to-end
- Test subscription webhook handling
- Verify plan limits enforcement

### Priority 3: Production Auth Mode Testing
Now that all APIGate issues are resolved, test with real APIGate auth:
1. Set `HOSTER_AUTH_MODE=header`
2. Access Hoster through APIGate proxy (not directly)
3. Verify X-User-ID header is properly passed
4. Verify billing events use real user IDs
5. Test auto-registration with service accounts (Issue #25 now fixed)
6. Test public routes for unauthenticated endpoints (Issue #22 now fixed)

### Other Options (Lower Priority)
- **Node Metrics Collection**: CPU/memory/disk usage from nodes
- **Enhanced Scheduling**: Round-robin, least-loaded, affinity policies
- **WebSocket Updates**: Real-time deployment status updates
- **Template Versioning**: Version management for templates
- **Prometheus Metrics Endpoint**: For monitoring integration

### Quick Verification Commands

```bash
# Check if services are running
curl -s http://localhost:8080/health | jq .   # Hoster
curl -s http://localhost:9091/health | jq .   # App Proxy
curl -s http://localhost:8082/portal          # APIGate

# Check logs
tail -20 /tmp/hoster.log
tail -20 /tmp/apigate.log

# Test deployed app
curl http://localhost:30000                    # Direct
curl http://my-app.apps.localhost:9091         # Via proxy
```

---

## PREVIOUS SESSION SUMMARY (January 19, 2026)

### What Was Accomplished: App Proxy STC Alignment + F014 Spec

This session completed the STC (Spec → Test → Code) alignment for the App Proxy feature and added the comprehensive F014 production readiness spec.

**1. App Proxy STC Alignment:**

Identified and implemented missing integration pieces from the App Proxy spec:

- **Port Allocation on Deployment Start** (`internal/shell/api/resources/deployment.go:246-269`):
  - Allocates ports from range 30000-39999 using `proxy.AllocatePort()`
  - Persists proxy port to deployment before starting containers
  - Uses `store.GetUsedProxyPorts()` to avoid collisions

- **Container Port Binding in Orchestrator** (`internal/shell/docker/orchestrator.go:319-337`):
  - Detects "primary service" (first service with exposed ports in topological order)
  - Binds primary service's first exposed port to `deployment.ProxyPort`
  - Binds to `127.0.0.1` only for security (prevents external access)

**2. F014 Production Readiness Spec:**

Added `specs/features/F014-e2e-production-readiness.md`:
- Comprehensive spec for transforming Hoster into a self-service production platform
- **Phase A: App Proxy** - COMPLETE (implemented this and previous sessions)
- **Phase B: APIGate Integration** - Full route configuration, frontend deployment, auth flow
- **Phase C: Billing Integration** - Usage event reporting, plan limits enforcement
- **Phase D: Landing Page & Documentation** - Marketing pages, user docs
- **Phase E: Operational Readiness** - Health endpoints, metrics, structured logging

**Files Created/Modified:**
```
specs/features/F014-e2e-production-readiness.md  # Production readiness spec
internal/shell/docker/orchestrator.go            # Added proxy port binding
internal/shell/api/resources/deployment.go       # Added port allocation on start
```

**All Tests Pass:** 500+ tests across all packages

---

### Architecture Overview:
```
┌──────────────────────────────────────────────────────────────────────────┐
│ PRODUCTION PATH (setup.go → resources/deployment.go)                      │
│                                                                           │
│ DeploymentResource                                                        │
│  ├─ Scheduler (scheduler.Service)                                         │
│  │   ├── store.ListOnlineNodes()                                         │
│  │   ├── corescheduler.Schedule() (pure algorithm)                       │
│  │   └── nodePool.GetClient() or local fallback                          │
│  └─ docker.Orchestrator (created per-request with scheduled client)      │
└──────────────────────────────────────────────────────────────────────────┘
             │
             │ For nodeID="local" → uses local Docker client
             │ For nodeID="node-X" → uses SSHDockerClient from NodePool
             ▼
┌────────────────┐                      ┌────────────────┐
│ Local Docker   │  OR                  │ Remote Node    │
│ daemon         │                      │ via SSH+Minion │
└────────────────┘                      └────────────────┘
```

**Key Insight:** Production uses `setup.go` → `DeploymentResource` NOT `handler.go`.
The handler.go path is primarily for tests and non-api2go endpoints.

### Generic Factory Pattern:
```
┌─────────────────────────────────────────────────────────────────┐
│ createResourceApi<Resource, CreateReq, UpdateReq, CustomActions>│
│   → ResourceApi { list, get, create, update, delete }           │
│   → Custom actions (e.g., publish, enterMaintenance)            │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ createResourceHooks({ resourceName, api })                      │
│   → keys, useList, useGet, useCreate, useUpdate, useDelete      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ createIdActionHook(keys, actionFn)                              │
│   → Custom mutation hook for id-based actions                   │
└─────────────────────────────────────────────────────────────────┘
```

### Testing Environment Notes:
- Backend runs on port 9090: `HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=dev ./bin/hoster`
- Frontend dev server proxies to backend via vite.config.ts
- Dev mode (`HOSTER_AUTH_MODE=dev`) auto-authenticates as `dev-user`
- Auth modes: `header` (production), `dev` (local development), `none` (unauthenticated)

---

## Creator Worker Nodes - Task Status (ALL PHASES COMPLETE)

### Completed (All 7 Phases):
- [x] `specs/domain/node.md` - Node entity specification
- [x] `internal/core/domain/node.go` - Node domain model (51 tests)
- [x] `internal/core/scheduler/scheduler.go` - Pure scheduling algorithm (26 tests)
- [x] `internal/core/crypto/encryption.go` - AES-256-GCM encryption (26 tests)
- [x] `internal/core/auth/context.go` - Added AllowedCapabilities to PlanLimits
- [x] `internal/core/domain/template.go` - Added RequiredCapabilities field
- [x] Database migration 005_nodes (up/down)
- [x] Store interface + SQLite implementation for nodes/SSH keys (20 tests)
- [x] Handler test stubs for new Store interface
- [x] `internal/core/minion/protocol.go` - Minion protocol types (20 tests)
- [x] `cmd/hoster-minion/` - Minion binary with 18 commands
- [x] `internal/shell/docker/ssh_client.go` - SSHDockerClient implementing Client
- [x] `internal/shell/docker/node_pool.go` - Connection pool with lazy init
- [x] `internal/shell/docker/minion_embed.go` - Embedded minion binaries
- [x] Makefile updates for minion build
- [x] `internal/shell/scheduler/service.go` - Scheduling service with I/O
- [x] `internal/shell/scheduler/service_test.go` - 9 tests for scheduling service
- [x] `internal/shell/api/handler.go` - Scheduler integration
- [x] `internal/shell/api/resources/node.go` - Node JSON:API resource
- [x] `internal/shell/api/resources/ssh_key.go` - SSH Key resource
- [x] Authorization checks (CanManageNode, CanViewNode, CanCreateNode)
- [x] SSH key encryption with AES-256-GCM
- [x] Maintenance mode endpoints
- [x] OpenAPI documentation for new resources
- [x] `web/src/api/createResourceApi.ts` - Generic API factory
- [x] `web/src/hooks/createResourceHooks.ts` - Generic hooks factory
- [x] `web/src/api/nodes.ts` - Node API client
- [x] `web/src/api/ssh-keys.ts` - SSH Key API client
- [x] `web/src/hooks/useNodes.ts` - Node query hooks
- [x] `web/src/hooks/useSSHKeys.ts` - SSH Key query hooks
- [x] `web/src/components/nodes/` - NodeCard, AddNodeDialog, AddSSHKeyDialog
- [x] Nodes tab in Creator Dashboard

### Phase 7 - Health Checker Worker (COMPLETE):
- [x] `internal/shell/workers/health_checker.go` - Periodic health check worker (11 tests)
- [x] `internal/shell/store/store.go` - Added ListCheckableNodes interface method
- [x] `internal/shell/store/sqlite.go` - Implemented ListCheckableNodes
- [x] `cmd/hoster/config.go` - Added NodesConfig
- [x] `cmd/hoster/server.go` - Integrated health checker with server lifecycle
- [x] Background goroutine that pings nodes periodically
- [x] Updates node status (online/offline) and last_health_check timestamp
- [x] Records error messages for offline nodes
- [x] Configurable check interval (default: 60s), timeout, and concurrency

### Backend API Endpoints (Already Implemented):
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/nodes` | List creator's nodes |
| POST | `/api/v1/nodes` | Create new node |
| GET | `/api/v1/nodes/:id` | Get node details |
| PATCH | `/api/v1/nodes/:id` | Update node |
| DELETE | `/api/v1/nodes/:id` | Delete node |
| POST | `/api/v1/nodes/:id/maintenance` | Enter maintenance mode |
| DELETE | `/api/v1/nodes/:id/maintenance` | Exit maintenance mode |
| GET | `/api/v1/ssh_keys` | List creator's SSH keys |
| POST | `/api/v1/ssh_keys` | Create SSH key (upload private key) |
| GET | `/api/v1/ssh_keys/:id` | Get SSH key (fingerprint only) |
| DELETE | `/api/v1/ssh_keys/:id` | Delete SSH key |

### Verification Commands:
```bash
# Backend
make test      # All backend tests pass
make build     # Build with embedded minion binaries
HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=dev make run   # Start backend on :9090

# Backend with remote nodes enabled
HOSTER_SERVER_PORT=9090 HOSTER_AUTH_MODE=dev \
HOSTER_NODES_ENABLED=true \
HOSTER_NODES_ENCRYPTION_KEY=<32-byte-secret-key-here> \
make run

# Frontend
cd web
npm install    # Install dependencies
npm run build  # Build for production (should succeed)
npm run dev    # Start dev server on :3000 (proxies to :9090)
```

---

## Phase 1: Context Loading (MANDATORY)

### Step 1: Read Core Documents (in order)

```bash
# Run this to verify project exists
ls -la
```

Read these files in this exact order:

1. **`CLAUDE.md`** - Project memory, decisions, current state
2. **`specs/README.md`** - How specs work
3. **`specs/decisions/ADR-000-stc-methodology.md`** - THE methodology
4. **`specs/decisions/ADR-001-docker-direct.md`** - Architecture decision
5. **`specs/decisions/ADR-002-values-as-boundaries.md`** - Code organization

### Step 2: Read Post-MVP ADRs (for UI/API work)

These are critical for frontend work:

6. **`specs/decisions/ADR-003-jsonapi-api2go.md`** - JSON:API with api2go
7. **`specs/decisions/ADR-004-reflective-openapi.md`** - OpenAPI generation
8. **`specs/decisions/ADR-005-apigate-integration.md`** - Auth/billing via APIGate
9. **`specs/decisions/ADR-006-frontend-architecture.md`** - React + Vite frontend
10. **`specs/decisions/ADR-007-uiux-guidelines.md`** - UI/UX consistency

### Step 3: Read Feature Specs (for implementation)

```
specs/features/F008-authentication.md     - Header-based auth
specs/features/F009-billing-integration.md - Usage tracking
specs/features/F010-monitoring-dashboard.md - Health/logs/stats
specs/features/F011-marketplace-ui.md      - Template browsing (IMPLEMENTED)
specs/features/F012-deployment-management-ui.md - Deployment controls (IMPLEMENTED)
specs/features/F013-creator-dashboard-ui.md - Template management (IMPLEMENTED)
```

### Step 4: Read Domain Specs

```
specs/domain/template.md    - Template entity + JSON:API definition
specs/domain/deployment.md  - Deployment entity + JSON:API definition
specs/domain/monitoring.md  - Health, stats, logs, events types
specs/domain/user-context.md - AuthContext from APIGate headers
specs/domain/node.md        - Node entity for worker nodes
```

### Step 5: Verify Understanding

After reading, you should know:
- [ ] What is Hoster? (Deployment marketplace platform)
- [ ] What is STC? (Spec -> Test -> Code)
- [ ] What is "Values as Boundaries"? (Pure core, thin shell)
- [ ] What is JSON:API? (Standardized API format via api2go)
- [ ] What is APIGate? (External auth/billing, injects X-User-ID headers)
- [ ] Where do specs go? (`specs/` directory)
- [ ] Where does core code go? (`internal/core/`)
- [ ] Where does I/O code go? (`internal/shell/`)
- [ ] What libraries to use? (Listed in CLAUDE.md)
- [ ] How to add new CRUD resources? (Use createResourceApi + createResourceHooks factories)

### Step 6: Verify Tests Pass

```bash
make test
```

If tests fail, something is broken. Fix before proceeding.

---

## Phase 2: Task Understanding

### Step 7: Check Implementation Plan

Read the plan file for detailed implementation phases:
```
/Users/artpar/.claude/plans/wondrous-forging-donut.md
```

**Implementation Phases:**
- Phase -1: ADR & Spec Updates (COMPLETE)
- Phase 0: API Layer Migration (JSON:API + OpenAPI) (COMPLETE)
- Phase 1: APIGate Integration (Backend Auth) (COMPLETE)
- Phase 2: Billing Integration (COMPLETE)
- Phase 3: Monitoring Backend (COMPLETE)
- Phase 4: Frontend Foundation (COMPLETE)
- Phase 5: Frontend Views (COMPLETE)
- Phase 6: Integration & Polish (COMPLETE)

**Creator Worker Nodes Phases:**
- Phase 1: Domain Model & Scheduler (COMPLETE)
- Phase 2: Database Layer (COMPLETE)
- Phase 3: SSH Docker Client via Minion (COMPLETE)
- Phase 4: Scheduler Integration (COMPLETE)
- Phase 5: Node API Resource (COMPLETE)
- Phase 6: Frontend Nodes Tab (COMPLETE)
- Phase 7: Health Checker Worker (COMPLETE)

**All Creator Worker Nodes Phases Complete!** The feature is fully implemented.

### Step 8: Check Current Status

Read `CLAUDE.md` section "Current Implementation Status" to understand:
- What's DONE
- What's TODO
- What's BLOCKED

### Step 9: Check Agile Project

```
Use mcp__agile__workflow_execute with workflow: "backlog_status"
and project_id: "HOSTER" to see current tasks.
```

### Step 10: Understand User's Request

Now you can ask the user what they want to do. Compare against:
- What's already implemented (don't redo)
- What's in the TODO list
- What's explicitly NOT supported (don't implement)

---

## Phase 3: Before Making Changes

### Step 11: Identify Relevant Specs

For ANY change:
1. Find the relevant spec in `specs/`
2. If no spec exists -> WRITE SPEC FIRST
3. If spec exists -> READ IT before changing code

### Step 12: Pre-Flight Checklist

Before writing ANY code, verify:

- [ ] Spec exists for this feature/change
- [ ] I understand the spec's acceptance criteria
- [ ] I know what's "NOT Supported" (don't implement those)
- [ ] Tests exist (or I'll write them first)
- [ ] I know which directory: `internal/core/` or `internal/shell/`
- [ ] I'm using the approved libraries (check CLAUDE.md)
- [ ] For new CRUD resources: use createResourceApi + createResourceHooks factories

---

## Phase 4: Making Changes (STC Flow)

### For New Features

```
1. SPEC   -> Create specs/features/F###-name.md
2. TEST   -> Create internal/core/xxx/feature_test.go (failing tests)
3. CODE   -> Create internal/core/xxx/feature.go (make tests pass)
4. VERIFY -> make test
```

### For New CRUD Resources (Frontend)

```
1. Add types to web/src/api/types.ts
2. Create API client using createResourceApi factory
3. Create hooks using createResourceHooks factory
4. Create UI components
5. VERIFY -> npm run build
```

### For Bug Fixes

```
1. SPEC   -> Update spec if behavior was wrong
2. TEST   -> Add test that demonstrates the bug
3. CODE   -> Fix code to pass test
4. VERIFY -> make test
```

### For Refactoring

```
1. VERIFY -> make test (all pass before)
2. REFACTOR -> Make changes
3. VERIFY -> make test (all pass after)
```

---

## Phase 5: After Making Changes

### Step 13: Verify Sync

After any change:
- [ ] Spec still matches implementation
- [ ] Tests still match spec
- [ ] All tests pass (`make test`)
- [ ] Frontend builds (`cd web && npm run build`)

### Step 14: Update CLAUDE.md

If you:
- Completed a TODO item -> Move to DONE
- Made a new decision -> Document in CLAUDE.md
- Added new spec -> Reference in CLAUDE.md
- Changed architecture -> Update ADR or create new one

### Step 15: Update Agile Project

```
Use mcp__agile__task_transition to update task status.
```

---

## Key Library Changes (Post-MVP)

| Purpose | Old | New |
|---------|-----|-----|
| HTTP router | chi/v5 | gorilla/mux (api2go built-in) |
| API format | custom JSON | JSON:API via api2go |
| OpenAPI | manual | reflective generation |

**Backend Dependencies:**
- `github.com/manyminds/api2go` - JSON:API implementation
- `github.com/gorilla/mux` - Router (api2go support)
- `github.com/getkin/kin-openapi` - OpenAPI 3.0 types
- `golang.org/x/crypto/ssh` - SSH client for remote nodes

**Frontend Dependencies (web/package.json):**
- React 19 + React DOM 19
- React Router DOM 7.1
- TanStack Query 5.62
- Zustand 5.0
- Vite 6.0
- TailwindCSS 3.4
- Lucide React 0.469 (icons)

---

## Frontend File Structure

```
web/
├── src/
│   ├── api/
│   │   ├── client.ts              # JSON:API fetch wrapper
│   │   ├── types.ts               # TypeScript types
│   │   ├── createResourceApi.ts   # Generic API factory ← NEW
│   │   ├── templates.ts           # Template API (uses factory)
│   │   ├── deployments.ts         # Deployment API
│   │   ├── monitoring.ts          # Monitoring API
│   │   ├── nodes.ts               # Node API (uses factory) ← NEW
│   │   └── ssh-keys.ts            # SSH Key API (uses factory) ← NEW
│   ├── components/
│   │   ├── common/
│   │   │   ├── LoadingSpinner.tsx
│   │   │   ├── EmptyState.tsx
│   │   │   └── StatusBadge.tsx    # Includes node statuses
│   │   ├── layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── Layout.tsx
│   │   ├── templates/
│   │   │   ├── TemplateCard.tsx
│   │   │   ├── DeployDialog.tsx
│   │   │   └── CreateTemplateDialog.tsx
│   │   ├── deployments/
│   │   │   └── DeploymentCard.tsx
│   │   ├── nodes/                 # ← NEW
│   │   │   ├── NodeCard.tsx
│   │   │   ├── AddNodeDialog.tsx
│   │   │   ├── AddSSHKeyDialog.tsx
│   │   │   └── index.ts
│   │   └── ui/
│   │       ├── Button.tsx
│   │       ├── Input.tsx
│   │       ├── Label.tsx
│   │       ├── Textarea.tsx
│   │       ├── Select.tsx
│   │       ├── Tabs.tsx
│   │       ├── Dialog.tsx
│   │       ├── Card.tsx
│   │       ├── Badge.tsx
│   │       ├── Skeleton.tsx
│   │       └── index.ts
│   ├── hooks/
│   │   ├── createResourceHooks.ts # Generic hooks factory ← NEW
│   │   ├── useTemplates.ts        # (uses factory)
│   │   ├── useDeployments.ts
│   │   ├── useMonitoring.ts
│   │   ├── useNodes.ts            # ← NEW
│   │   └── useSSHKeys.ts          # ← NEW
│   ├── pages/
│   │   ├── marketplace/
│   │   │   ├── MarketplacePage.tsx
│   │   │   └── TemplateDetailPage.tsx
│   │   ├── deployments/
│   │   │   ├── MyDeploymentsPage.tsx
│   │   │   └── DeploymentDetailPage.tsx
│   │   ├── creator/
│   │   │   └── CreatorDashboardPage.tsx  # Has Nodes tab
│   │   └── NotFoundPage.tsx
│   ├── stores/
│   │   └── authStore.ts           # Zustand store
│   ├── lib/
│   │   └── cn.ts                  # clsx + tailwind-merge
│   ├── App.tsx
│   ├── main.tsx
│   └── index.css
├── package.json
├── vite.config.ts
├── tailwind.config.ts
├── postcss.config.js
└── tsconfig.json
```

---

## Adding New CRUD Resources (Quick Reference)

When adding a new resource type, use the generic factories:

**1. Add types to `web/src/api/types.ts`:**
```typescript
export interface FooAttributes { name: string; /* ... */ }
export type Foo = JsonApiResource<'foos', FooAttributes>;
export interface CreateFooRequest { name: string; /* ... */ }
export interface UpdateFooRequest { name?: string; /* ... */ }
```

**2. Create API client `web/src/api/foos.ts`:**
```typescript
import { createResourceApi } from './createResourceApi';
import type { Foo, CreateFooRequest, UpdateFooRequest } from './types';

export const foosApi = createResourceApi<Foo, CreateFooRequest, UpdateFooRequest>({
  resourceName: 'foos',
  // Optional: customActions, supportsUpdate, supportsDelete
});
```

**3. Create hooks `web/src/hooks/useFoos.ts`:**
```typescript
import { foosApi } from '@/api/foos';
import type { Foo, CreateFooRequest, UpdateFooRequest } from '@/api/types';
import { createResourceHooks } from './createResourceHooks';

const fooHooks = createResourceHooks<Foo, CreateFooRequest, UpdateFooRequest>({
  resourceName: 'foos',
  api: foosApi,
});

export const fooKeys = fooHooks.keys;
export const useFoos = fooHooks.useList;
export const useFoo = fooHooks.useGet;
export const useCreateFoo = fooHooks.useCreate;
export const useUpdateFoo = fooHooks.useUpdate;
export const useDeleteFoo = fooHooks.useDelete;
```

---

## Common Mistakes to Avoid

| Mistake | Why It's Bad | Prevention |
|---------|--------------|------------|
| Writing code without reading specs | You'll implement wrong behavior | Always read spec first |
| Writing code before tests | No safety net for refactoring | Write test first, see it fail |
| Putting I/O in `internal/core/` | Breaks architecture, needs mocks | Keep core pure |
| Using different library | Inconsistency, harder to maintain | Check CLAUDE.md library list |
| Implementing "NOT Supported" items | Scope creep, wasted effort | Read spec's NOT Supported section |
| Skipping `make test` | Broken code goes unnoticed | Run after every change |
| Not updating CLAUDE.md | Next session loses context | Update after significant changes |
| Ignoring ADR-007 UI guidelines | Inconsistent UI | Follow semantic colors, patterns |
| Not using factories for new CRUD | Code duplication | Use createResourceApi + createResourceHooks |

---

## Quick Reference: Where Things Go

| What | Where | Example |
|------|-------|---------|
| Domain specs | `specs/domain/` | `template.md` |
| Feature specs | `specs/features/` | `F008-authentication.md` |
| ADRs | `specs/decisions/` | `ADR-003-jsonapi-api2go.md` |
| Pure logic | `internal/core/` | `domain/template.go` |
| I/O code | `internal/shell/` | `docker/client.go` |
| Unit tests | Same dir as code | `template_test.go` |
| E2E tests | `tests/e2e/` | `deploy_test.go` |
| Sample templates | `examples/` | `wordpress/compose.yml` |
| Frontend | `web/` | React + Vite app |
| UI components | `web/src/components/ui/` | `Button.tsx` |
| Page components | `web/src/pages/` | `MarketplacePage.tsx` |
| API factories | `web/src/api/` | `createResourceApi.ts` |
| Hook factories | `web/src/hooks/` | `createResourceHooks.ts` |

---

## Emergency Recovery

### If Tests Are Failing

```bash
# See what's broken
make test 2>&1 | grep FAIL

# Check recent changes
git log --oneline -10

# Revert if needed
git checkout HEAD~1 -- <file>
```

### If Frontend Won't Build

```bash
cd web
rm -rf node_modules
npm install
npm run build
```

### If You're Lost

1. Re-read `CLAUDE.md` from the beginning
2. Run `make test` to verify baseline
3. Run `cd web && npm run build` to verify frontend
4. Read the specific spec for what you're working on
5. Read the plan file for implementation phases
6. Ask user for clarification

### If Spec Doesn't Exist

DO NOT write code. Instead:

1. Create spec file
2. Write requirements
3. Write "NOT Supported" section
4. Get user confirmation
5. Then proceed with tests and code

---

## Session End Checklist

Before ending a session:

- [ ] All tests pass (`make test`)
- [ ] Frontend builds (`cd web && npm run build`)
- [ ] CLAUDE.md is updated with:
  - [ ] New DONE items
  - [ ] New TODO items
  - [ ] Any new decisions
- [ ] Agile project updated
- [ ] User informed of current state
- [ ] This file updated if project state changed significantly
