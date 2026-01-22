# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (January 22, 2026)

### Status: LOCAL E2E ENVIRONMENT FULLY FUNCTIONAL - MONITORING COMPLETE

**Local E2E Testing Environment - FULLY WORKING:**

- **APIGate** (localhost:8082) - Single entry point for all traffic
- **Hoster Backend** (localhost:8080) - API + embedded frontend
- **App Proxy** (localhost:9091) - Routes deployed apps via subdomain
- **Database**: `/tmp/hoster-e2e-test/hoster.db` (Hoster), `/tmp/hoster-e2e-test/apigate.db` (APIGate)
- **Routes configured**: Frontend (/*), API (/api/*), App Proxy (*.apps.localhost/*)
- **Auto-registration**: Disabled (using manual route configuration)
- **Auth**: Disabled for testing (`auth_required=0` on all routes)

**New Features Completed:**

1. **Deployment Monitoring** (January 22, 2026)
   - ✅ Container event recording (lifecycle tracking)
   - ✅ Stats tab (CPU, memory, network, disk I/O)
   - ✅ Logs tab (container logs with filtering)
   - ✅ Events tab (deployment history timeline)

2. **Default Marketplace Templates** (January 22, 2026)
   - ✅ PostgreSQL Database ($5/month, 512MB RAM, 0.5 CPU)
   - ✅ MySQL Database ($5/month, 512MB RAM, 0.5 CPU)
   - ✅ Redis Cache ($3/month, 256MB RAM, 0.25 CPU)
   - ✅ MongoDB Database ($5/month, 512MB RAM, 0.5 CPU)
   - ✅ Nginx Web Server ($2/month, 64MB RAM, 0.1 CPU)
   - ✅ Node.js Application ($4/month, 256MB RAM, 0.5 CPU)

**Local Development Modes:**

1. **Standalone (Dev Auth Mode)** - Simple local development
   - `HOSTER_AUTH_MODE=dev`
   - In-memory session-based auth
   - Good for quick UI testing

2. **APIGate Integration (Header Auth)** - Production-like setup
   - APIGate on port 8082, Hoster on 8080
   - Header injection: X-User-ID, X-Plan-ID, X-Key-ID
   - Full auth flow with real user UUIDs
   - Verified working January 20, 2026

**Production Deployment:**
- **URL**: https://emptychair.dev
- **Server**: AWS EC2 (ubuntu@emptychair.dev)
- **APIGate**: Handling TLS via ACME (auto-cert from Let's Encrypt)
- **Hoster**: Running as systemd service (v0.1.0 - backend only)

**What's Working Locally:**
- ✅ Dev auth mode with session-based login
- ✅ APIGate integration with header injection
- ✅ Marketplace browsing
- ✅ Template deployment
- ✅ Container orchestration
- ✅ App proxy routing via subdomain
- ✅ All E2E flows verified (both modes)

**What Needs Production Work:**
- CI workflows need verification (npm/rollup issues were being fixed)
- Need release with embedded frontend (v0.2.0)
- Production routing configuration verification

---

## LOCAL DEVELOPMENT SETUP

### Current E2E Test Environment (January 22, 2026)

**Location:** `/tmp/hoster-e2e-test/`

**Running Services:**
```bash
# Check if services are running
ps aux | grep -E "(apigate|hoster)" | grep -v grep
lsof -i :8082  # APIGate
lsof -i :8080  # Hoster
lsof -i :9091  # App Proxy
```

**Start Services:**
```bash
# Terminal 1: Start APIGate
cd /tmp/hoster-e2e-test
apigate serve --config apigate.yaml > apigate.log 2>&1 &

# Terminal 2: Start Hoster (without auto-registration)
cd /Users/artpar/workspace/code/hoster
HOSTER_DATABASE_DSN=/tmp/hoster-e2e-test/hoster.db \
HOSTER_APIGATE_AUTO_REGISTER=false \
./bin/hoster > /tmp/hoster-e2e-test/hoster.log 2>&1 &
```

**Access:**
- **Frontend:** http://localhost:8082/ (via APIGate)
- **Marketplace:** http://localhost:8082/marketplace
- **API:** http://localhost:8082/api/v1/* (via APIGate)
- **Deployed Apps:** http://{name}.apps.localhost:8082/

**Routes Configuration (in APIGate database):**
```sql
-- Check routes
sqlite3 /tmp/hoster-e2e-test/apigate.db "SELECT name, path_pattern, host_pattern, priority, auth_required FROM routes ORDER BY priority DESC;"

-- Expected output:
-- hoster-app-proxy | /* | *.apps.localhost | 100 | 0
-- hoster-api       | /api/* |              | 50  | 0
-- hoster-frontend  | /*     |              | 10  | 0
```

**Logs:**
```bash
tail -f /tmp/hoster-e2e-test/apigate.log   # APIGate logs
tail -f /tmp/hoster-e2e-test/hoster.log    # Hoster logs
```

**Important:** All traffic MUST go through APIGate (localhost:8082). Never access Hoster directly on localhost:8080 during E2E testing.

---

### Option A: Standalone (Dev Auth Mode)

**Terminal 1: Start Hoster**
```bash
cd /Users/artpar/workspace/code/hoster
HOSTER_AUTH_MODE=dev go run ./cmd/hoster
```

**Terminal 2: Start Frontend**
```bash
cd /Users/artpar/workspace/code/hoster/web
npm run dev
```

**Access:**
- Frontend: http://localhost:3000
- API: http://localhost:8080
- App Proxy: http://localhost:9091
- Deployed apps: http://{name}.apps.localhost:9091

**Note:** Add entries to `/etc/hosts` for app subdomains:
```
127.0.0.1 {deployment-name}.apps.localhost
```

### Option B: Full Production-Like Setup (APIGate + Hoster)

This setup mirrors production with APIGate handling user authentication and Hoster receiving user context via headers.

**Terminal 1: Start APIGate**
```bash
cd /Users/artpar/workspace/code/apigate
./apigate --config=/tmp/apigate-temp.yaml
```

Config file (`/tmp/apigate-temp.yaml`):
```yaml
database:
  dsn: "/path/to/apigate.db"
server:
  port: 8082
```

**Terminal 2: Start Hoster**
```bash
cd /Users/artpar/workspace/code/hoster
HOSTER_AUTH_MODE=header HOSTER_SERVER_PORT=8080 \
  HOSTER_APP_PROXY_ENABLED=true HOSTER_APP_PROXY_ADDRESS=0.0.0.0:9091 \
  HOSTER_APP_PROXY_BASE_DOMAIN=apps.localhost ./hoster
```

**APIGate Configuration (via admin UI at http://localhost:8082):**

1. **Create Admin User:**
   - Complete setup wizard at http://localhost:8082
   - Create admin account

2. **Create Upstream:**
   - Name: `hoster-api`
   - URL: `http://localhost:8080`

3. **Create Route with Header Injection:**
   - Name: `Default Route`
   - Path Pattern: `/*`
   - Match Type: Prefix
   - Target Upstream: `hoster-api`
   - Set Headers (Request Transform):
     ```
     X-User-ID=userID
     X-Plan-ID=planID
     X-Key-ID=keyID
     ```

4. **Create Test User via Portal:**
   - Go to http://localhost:8082/portal/
   - Sign up with email/password
   - Create an API key

**Access:**
- APIGate Portal: http://localhost:8082/portal
- APIGate Admin: http://localhost:8082/dashboard
- Hoster API (via APIGate): http://localhost:8082/api/v1/*
- App Proxy: http://localhost:9091
- Deployed apps: http://{name}.apps.localhost:9091

### E2E Test Flow (Standalone - Dev Auth)

1. Open http://localhost:3000
2. Navigate to http://localhost:3000/login
3. Login with any email/password (dev auth accepts anything)
4. Browse marketplace at /marketplace
5. Click on a template, click "Deploy Now"
6. After deployment created, start it via API or UI
7. Access deployed app at http://{name}.apps.localhost:9091

### E2E Test Flow (APIGate Integration - Verified Working January 20, 2026)

**Setup Summary:**
- APIGate running on port 8082
- Hoster running on port 8080 (header auth mode)
- Routes configured to inject X-User-ID from APIGate auth context

**API Testing (requires API key):**
```javascript
// From browser console at http://localhost:8082/portal/dashboard
const apiKey = 'ak_your_api_key_here';

// List templates
const resp = await fetch('http://localhost:8082/api/v1/templates', {
  headers: { 'X-API-Key': apiKey, 'Accept': 'application/vnd.api+json' }
});
console.log(await resp.json());

// Create deployment
const deploy = await fetch('http://localhost:8082/api/v1/deployments', {
  method: 'POST',
  headers: {
    'X-API-Key': apiKey,
    'Content-Type': 'application/vnd.api+json',
    'Accept': 'application/vnd.api+json'
  },
  body: JSON.stringify({
    data: { type: 'deployments', attributes: { name: 'my-app', template_id: 'tmpl_xxx' }}
  })
});
console.log(await deploy.json());

// Start deployment
const start = await fetch('http://localhost:8082/api/v1/deployments/depl_xxx/start', {
  method: 'POST',
  headers: { 'X-API-Key': apiKey }
});
console.log(await start.json());
```

**Verified Flow (January 20, 2026):**
1. ✅ User signup via APIGate portal
2. ✅ API key creation
3. ✅ Header injection working (X-User-ID = real user UUID)
4. ✅ Deployment creation with correct customer_id
5. ✅ Deployment start
6. ✅ App accessible via subdomain proxy (http://my-nginx-app.apps.localhost:9091)

**Limitation:** Browser access to Hoster frontend through APIGate requires API key. APIGate doesn't currently support public routes. Options:
- Access Hoster frontend directly (port 8080) in dev mode
- Use API key in all fetch requests from frontend
- File feature request for APIGate public routes

---

## IMMEDIATE NEXT STEPS (Priority Order)

### 1. Complete APIGate Auto-Registration Fix

**Current Issue:** Auto-registration fails with 401 when accessing `/admin/upstreams` through APIGate proxy.

**Root Cause:** Hoster frontend route (`/*`, priority 10) catches all requests including `/admin/*`, proxying them to Hoster instead of APIGate's built-in admin endpoints.

**Solution Options:**

**Option A: Higher Priority Admin Route (RECOMMENDED)**
- Add admin route to APIGate with higher priority (priority 5)
- Path: `/admin/*`, upstream: `apigate-internal` (direct to APIGate)
- This prevents frontend route from catching admin requests

**Option B: Exclude Pattern in Frontend Route**
- Modify Hoster frontend route to exclude `/admin/*`
- Requires APIGate route pattern support (check if available)

**Option C: Keep Manual Configuration (CURRENT)**
- Continue using manually configured routes
- Disable auto-registration (`HOSTER_APIGATE_AUTO_REGISTER=false`)
- Simple and working for testing

### 2. Verify CI Workflow is Fixed

Check if the npm/rollup issue is resolved:
```bash
gh run list --repo artpar/hoster --limit 3
```

If still failing, see troubleshooting section below.

### 3. Create New Release with Embedded Frontend

Once CI passes:
```bash
# Commit any remaining changes
git add -A
git commit -m "feat: Complete monitoring and default templates"

# Delete old failed tag if exists
git tag -d v0.2.0 2>/dev/null
git push origin :refs/tags/v0.2.0 2>/dev/null

# Create fresh release
git tag v0.2.2
git push origin v0.2.2
```

### 4. Deploy to Production

```bash
cd deploy/local
make deploy-release VERSION=v0.2.2
```

### 5. Test Production E2E

1. Navigate to https://emptychair.dev
2. Should see Hoster marketplace (not APIGate portal)
3. Sign up / Log in via APIGate
4. Browse templates (should see 7 templates with pricing)
5. Deploy a template
6. Monitor deployment (Events, Stats, Logs tabs)
7. Access deployed app at https://{name}.apps.emptychair.dev

---

## Files Changed This Session (Session 3)

**Added dev auth mode:**
- `internal/shell/api/dev_auth.go` - NEW - In-memory session auth endpoints
- `internal/shell/api/setup.go` - Register dev auth routes when auth.mode=dev

**Fixed app proxy:**
- `internal/shell/proxy/server.go` - Strip port from Host header for domain matching

**Updated frontend proxy:**
- `web/vite.config.ts` - Proxy /auth and /api to Hoster for local dev

**Commit:** `feat: Add dev auth mode for local E2E testing`

---

## Architecture

### Local Development Mode

```
┌─────────────────────────────────────────────────────────────┐
│  Local Development Stack (Dev Auth Mode)                    │
│                                                             │
│  Frontend (localhost:3000)                                  │
│    │                                                        │
│    ├─ /api/* ──► Hoster API (localhost:8080)               │
│    │                                                        │
│    └─ /auth/* ──► Hoster Dev Auth (localhost:8080)         │
│                                                             │
│  Hoster (localhost:8080)                                    │
│    ├─ /api/v1/* - Deployment API                           │
│    └─ /auth/* - Dev auth endpoints (session-based)         │
│                                                             │
│  App Proxy (localhost:9091)                                 │
│    └─ *.apps.localhost → Deployed containers               │
└─────────────────────────────────────────────────────────────┘
```

### Production Mode (APIGate Integration)

```
┌─────────────────────────────────────────────────────────────┐
│  Production Stack (APIGate Auth)                            │
│                                                             │
│  Internet ───► APIGate (TLS termination)                   │
│                    │                                        │
│                    ├─ /portal/* → APIGate portal           │
│                    ├─ /api/* → Hoster API (with X-User-ID) │
│                    └─ /* → Hoster static files             │
│                                                             │
│  Hoster (auth.mode=header)                                 │
│    └─ Trusts X-User-ID headers from APIGate                │
│                                                             │
│  App Proxy                                                  │
│    └─ *.apps.emptychair.dev → Deployed containers          │
└─────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting

### CI npm/rollup Issue

If CI fails with:
```
Error: Cannot find module @rollup/rollup-linux-x64-gnu
```

**Option A: Clean install (already attempted)**
```yaml
- run: rm -rf node_modules package-lock.json && npm install
```

**Option B: Pin rollup version**
```json
// In web/package.json, add:
"overrides": {
  "rollup": "4.9.6"
}
```

**Option C: Use different npm version**
```yaml
- uses: actions/setup-node@v4
  with:
    node-version: '20'
```

### App Proxy Not Finding Apps

1. Check deployment has a domain assigned:
   ```bash
   curl http://localhost:8080/api/v1/deployments/{id} | jq '.data.attributes.domains'
   ```

2. Ensure /etc/hosts has the subdomain entry

3. Verify app proxy is running on port 9091

### Dev Auth Session Issues

Dev auth stores sessions in memory. Restarting Hoster clears all sessions.
Just log in again after restart.

---

## Production Management

**Deployment via Makefile:**
```bash
cd deploy/local
make deploy-release                    # Deploy latest release
make deploy-release VERSION=v0.2.0     # Deploy specific version
```

**Server Management:**
```bash
cd deploy/local
make status           # Show service status
make logs             # Tail all logs
make logs-hoster      # Tail Hoster logs only
make restart          # Restart both services
make shell            # SSH into server
```

---

## Session History

### Session 5 (January 22, 2026) - CURRENT SESSION

**Goal:** Complete monitoring features and prepare for production deployment

**Accomplished:**

1. **Deployment Monitoring Implementation:**
   - Added container event recording to orchestrator (`internal/shell/docker/orchestrator.go`)
   - Created StoreInterface to avoid circular dependencies
   - Added `recordEvent()` method to track lifecycle events
   - Updated all NewOrchestrator() call sites (9 locations) to pass store parameter
   - Events recorded: container_created, container_started, container_stopped, container_restarted
   - Verified Events tab showing deployment history in UI

2. **Monitoring Tabs Testing:**
   - Stats tab: CPU, memory, network, disk I/O metrics working
   - Logs tab: Container logs with timestamps and filtering working
   - Events tab: Deployment lifecycle events timeline working

3. **Default Marketplace Templates:**
   - Created migration `007_default_templates.up.sql` with 6 templates
   - Templates include resource limits (CPU/RAM/disk) and pricing
   - Published PostgreSQL template to marketplace
   - Verified 7 templates total visible in marketplace (6 default + 1 test)

4. **Local E2E Environment Setup:**
   - Created `/tmp/hoster-e2e-test/` test environment
   - Set up APIGate (localhost:8082) and Hoster (localhost:8080)
   - Manually configured routes in APIGate database:
     - `hoster-frontend`: Priority 10, path `/*`, auth_required=0
     - `hoster-api`: Priority 50, path `/api/*`, auth_required=0
     - `hoster-app-proxy`: Priority 100, path `/*`, host `*.apps.localhost`, auth_required=0
   - Disabled auto-registration to use existing routes
   - Verified full E2E flow through APIGate

5. **Browser-Based E2E Testing:**
   - Used Chrome DevTools MCP for testing
   - Frontend accessible at `http://localhost:8082/`
   - API working at `http://localhost:8082/api/v1/templates`
   - Marketplace showing all 7 templates
   - All network requests going through APIGate (verified)

**Files Changed:**
- `internal/shell/docker/orchestrator.go` - Added event recording infrastructure
- `internal/shell/api/handler.go` - Updated NewOrchestrator calls (5 locations)
- `internal/shell/api/resources/deployment.go` - Updated NewOrchestrator calls (4 locations)
- `internal/shell/store/migrations/007_default_templates.up.sql` - NEW - Default templates
- `internal/shell/store/migrations/007_default_templates.down.sql` - NEW - Rollback migration

**Not Completed:**
- APIGate auto-registration (401 errors when accessing admin endpoints through proxy)
- Production deployment (needs CI fixes first)
- APIGate per-route auth feature request (currently all routes require auth unless manually disabled)

**Known Issues:**
- APIGate admin endpoints (`/admin/*`) are being caught by Hoster frontend route
- Auto-registration requires direct access to APIGate admin API (not through proxy)
- Solution: Disabled auto-registration, using manually configured routes

**Testing Architecture Verified:**
```
Browser → localhost:8082 (APIGate ONLY)
              ↓
  ┌───────────┴────────────┐
  ▼                        ▼
Frontend/*            API/api/*           App Proxy/*.apps.localhost/*
Priority 10           Priority 50         Priority 100
auth_required=0       auth_required=0     auth_required=0
  ↓                        ↓                   ↓
localhost:8080        localhost:8080      localhost:9091
(Hoster)              (Hoster API)        (App Proxy)
```

---

### Session 4 (January 20, 2026)

**Goal:** Full APIGate + Hoster E2E integration testing

**Accomplished:**
1. Created Dockerfile for Hoster containerization
2. Created docker-compose.local.yml for full E2E stack (APIGate + Hoster)
3. Added RequestTransform support to APIGate client for header injection
4. Updated registrar to configure X-User-ID/X-Plan-ID/X-Key-ID headers
5. Added Makefile targets for local E2E management
6. Created local-e2e-setup.sh and test-local-e2e.sh automation scripts
7. Verified full APIGate integration E2E flow:
   - User signup via APIGate portal ✅
   - API key creation ✅
   - Header injection working (customer_id = real user UUID) ✅
   - Deployment creation via APIGate proxy ✅
   - Deployment start ✅
   - App accessible via subdomain proxy (my-nginx-app.apps.localhost:9091) ✅
8. Updated documentation with comprehensive APIGate E2E instructions

**Files Added/Changed:**
- `Dockerfile` - NEW - Hoster container build
- `deploy/docker-compose.local.yml` - NEW - Full E2E stack
- `deploy/env.local` - NEW - Environment template
- `internal/shell/apigate/client.go` - RequestTransform type added
- `internal/shell/apigate/registrar.go` - Header injection config
- `Makefile` - Local E2E targets
- `scripts/local-e2e-setup.sh` - NEW - Setup automation
- `scripts/test-local-e2e.sh` - NEW - E2E test automation
- `docs/local-e2e-development.md` - NEW - Developer guide
- `docs/screenshots/e2e-nginx-deployed.png` - NEW - E2E verification

**Not Completed:**
- Production deployment (CI needs verification first)
- Production E2E testing

**Known Limitation:**
APIGate requires API key for ALL proxied routes. Browser access to Hoster frontend through APIGate needs API key in fetch requests.

---

### Session 3 (January 20, 2026)

**Goal:** Make Hoster E2E usable locally

**Accomplished:**
1. Created dev auth mode (`internal/shell/api/dev_auth.go`)
   - Session-based auth with in-memory storage
   - Endpoints: /auth/login, /auth/register, /auth/me, /auth/logout
2. Fixed app proxy port stripping bug
   - Browsers send `Host: name.apps.localhost:9091` with port
   - DB stores hostname without port
   - Fixed by stripping port before domain lookup
3. Verified full E2E flow locally:
   - Login ✅
   - Browse marketplace ✅
   - Deploy template ✅
   - Start deployment ✅
   - Access via subdomain ✅

**Not Completed:**
- Production deployment (CI needs verification)
- Production E2E testing

### Session 2 (January 20, 2026)

**Accomplished:**
- Fixed APIGate admin API bug (/api/ → /admin/)
- Set up GitHub Actions CI/CD workflows
- Created v0.1.0 release (backend only)
- Deployed v0.1.0 to production
- Created embedded frontend handler

**Issues:**
- CI workflows failing (npm/rollup issues)

### Session 1 (Earlier)

- Initial Hoster development
- Backend implementation complete

---

## Quick Reference

**GitHub:**
- Repo: https://github.com/artpar/hoster
- Releases: https://github.com/artpar/hoster/releases
- Actions: https://github.com/artpar/hoster/actions

**Production:**
- URL: https://emptychair.dev
- Server: ubuntu@emptychair.dev

**Local Dev Ports:**
- Hoster API: 8080
- App Proxy: 9091
- Frontend dev: 3000
- APIGate (optional): 8082

---

## Onboarding Checklist for New Session

1. [ ] Read CLAUDE.md completely
2. [ ] Read this SESSION-HANDOFF.md
3. [ ] Check CI status: `gh run list --repo artpar/hoster --limit 3`
4. [ ] If CI failing, fix the issue (see troubleshooting)
5. [ ] If CI passing, create release and deploy
6. [ ] Test end-to-end on production

**For local development:**
1. [ ] Start Hoster: `HOSTER_AUTH_MODE=dev go run ./cmd/hoster`
2. [ ] Start frontend: `cd web && npm run dev`
3. [ ] Open http://localhost:3000
4. [ ] Test E2E flow (login → browse → deploy → access)
