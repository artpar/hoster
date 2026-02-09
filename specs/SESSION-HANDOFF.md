# Session Handoff Protocol

> This document is for Claude (AI assistant) when starting a new session with zero memory.
> Follow this protocol EXACTLY before doing any work.

---

## CURRENT PROJECT STATE (February 6, 2026)

### Status: SSH KEYS PAGE + WEB-UI TEMPLATES + E2E DEPLOYMENT VERIFIED

**Local E2E Testing Environment - FULLY WORKING:**

- **APIGate** (localhost:8082) - Single entry point for all traffic
- **Hoster Backend** (localhost:8080) - API + embedded frontend
- **App Proxy** (localhost:9091) - Routes deployed apps via subdomain
- **Database**: `/tmp/hoster-e2e-test/hoster.db` (Hoster), `/tmp/hoster-e2e-test/apigate.db` (APIGate)
- **Routes configured**: Frontend (/*), API (/api/*), App Proxy (*.apps.localhost/*)
- **Auto-registration**: Removed (routes configured manually in APIGate)
- **Auth**: Disabled for testing (`auth_required=0` on all routes)
- **Subdomain Routing**: ✅ WORKING (requires APIGate commit 5d72804 or later)

**IMPORTANT - APIGate Version Requirement:**
App proxy subdomain routing requires APIGate with fix from issue #41 (commit `5d72804`).
This fix addresses:
1. Host header extraction from `r.Host` (Go stores it separately from `r.Header`)
2. Host header preservation when proxying to upstream
3. Public route detection before API key check
4. Priority route middleware for host-based patterns

To update APIGate:
```bash
cd /Users/artpar/workspace/code/apigate
git pull  # Must include commit 5d72804 or later
go build -o bin/apigate ./cmd/apigate
```

**New Features Completed:**

1. **Dedicated SSH Keys Page** (February 6, 2026)
   - ✅ New `/ssh-keys` route with standalone page
   - ✅ Table view: Name, Fingerprint, Used By (node badges), Created, Actions
   - ✅ Cross-references nodes using each key
   - ✅ Delete warning shows which nodes use the key
   - ✅ Sidebar nav item with KeyRound icon
   - ✅ `/nodes` "Manage SSH Keys" links to `/ssh-keys`

2. **Web-UI App Templates** (February 6, 2026)
   - ✅ Migration 008: 6 new templates with real web UIs
   - ✅ WordPress ($8/mo) - CMS with MySQL, multi-service compose
   - ✅ Uptime Kuma ($4/mo) - Monitoring dashboard
   - ✅ Gitea ($5/mo) - Self-hosted Git service
   - ✅ n8n ($6/mo) - Workflow automation
   - ✅ IT Tools ($2/mo) - Developer utility collection
   - ✅ Metabase ($7/mo) - Business intelligence
   - ✅ Marketplace now shows 12 templates total

3. **Real Web-UI App Deployment E2E** (February 6, 2026)
   - ✅ Deployed Uptime Kuma from marketplace via web UI
   - ✅ Container pulled, created, started (~20 seconds)
   - ✅ Accessed running app on localhost:30003 - full web UI loaded
   - ✅ Completed Uptime Kuma admin setup, reached monitoring dashboard
   - ✅ Hoster monitoring tabs verified: Overview, Logs, Stats, Events
   - ✅ Plan limits correctly enforced (blocked 2nd deployment on free tier)

4. **Remote Node E2E Testing** (January 23, 2026)
   - ✅ AWS EC2 instance at 98.82.190.29 with Docker
   - ✅ SSH key management via web UI
   - ✅ Node registration via web UI (aws-test-node)
   - ✅ Deployment scheduling to remote nodes
   - ✅ Container creation/start on remote Docker host
   - ✅ Full deployment lifecycle verified on remote node

2. **UI Improvements** (January 23, 2026)
   - ✅ Replaced all native dialogs (confirm/alert) with React components
   - ✅ Created ConfirmDialog component for destructive actions
   - ✅ Created AlertDialog component for notifications
   - ✅ Fixed Start button visibility for pending deployments

3. **Deployment Monitoring** (January 22, 2026)
   - ✅ Container event recording (lifecycle tracking)
   - ✅ Stats tab (CPU, memory, network, disk I/O)
   - ✅ Logs tab (container logs with filtering)
   - ✅ Events tab (deployment history timeline)

4. **Default Marketplace Templates** (January 22, 2026)
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
- ✅ App proxy routing via subdomain (through APIGate - requires commit 5d72804+)
- ✅ Deployment monitoring (Events, Stats, Logs)
- ✅ All E2E flows verified (both modes)

**What Needs Production Work:**
- ✅ Local E2E fully functional (including real web-UI app deployments)
- ✅ Remote node deployment working (verified on AWS EC2)
- ✅ Web-UI templates working (Uptime Kuma verified end-to-end)
- ✅ Auth moved to token-based (X-Auth-Token header) — session cookie issue (#54) no longer relevant
- ✅ Critical privacy bug FIXED - deployment list now filtered by authenticated user
- ✅ Auth UX improvements complete - user profile in header, better error messages
- ✅ SSH Keys promoted to first-class page with node cross-references
- Need release v0.2.5+ after APIGate fix is deployed
- Production manual testing required (ALL journeys in specs/user-journeys.md)
- CI workflows need verification (npm/rollup issues were being fixed)

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

# Terminal 2: Start Hoster
cd /Users/artpar/workspace/code/hoster
HOSTER_DATABASE_DSN=/tmp/hoster-e2e-test/hoster.db \
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

-- Expected output (priority order):
-- hoster-app-proxy | /*                    | *.apps.localhost | 100 | 0
-- hoster-billing   | /api/v1/deployments*  |                  | 55  | 1
-- hoster-api       | /api/*                |                  | 50  | 0
-- hoster-frontend  | /*                    |                  | 10  | 0
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

### 1. Commit Changes and Create Release

```bash
cd /Users/artpar/workspace/code/hoster
git add -A
git status
# Commit SSH Keys page, web-UI templates, and related changes
git commit -m "feat: Add dedicated SSH Keys page, web-UI app templates (WordPress, Uptime Kuma, etc.)"
git push origin main
```

### 2. Build Frontend and Create Release

```bash
# Build frontend
cd web && npm install && npm run build
cp -r dist ../internal/shell/api/webui/

# Build Go binary with embedded frontend
cd ..
go build -o bin/hoster ./cmd/hoster

# Tag release
git tag v0.2.6
git push origin v0.2.6
```

### 3. Deploy to Production

```bash
cd deploy/local
make deploy-release VERSION=v0.2.6
```

### 4. Test Production E2E

1. Navigate to https://emptychair.dev
2. Browse marketplace (should see 12 templates — infra + web-UI apps)
3. Sign up / Log in (token-based auth via X-Auth-Token)
4. Test SSH Keys page at /ssh-keys
5. Deploy Uptime Kuma or IT Tools
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

### Subdomain Routing Through APIGate Not Working

**Symptom:** Requests to `http://{name}.apps.localhost:8082/` return Hoster homepage instead of deployed app.

**Cause:** APIGate version is missing host-based routing fix.

**Solution:** Update APIGate to commit `5d72804` or later:
```bash
cd /Users/artpar/workspace/code/apigate
git pull
go build -o bin/apigate ./cmd/apigate
# Restart APIGate
```

**Verification:**
```bash
# Should return nginx welcome page (or your deployed app)
curl -H "Host: nginx-web-server-mkpnwv4e.apps.localhost" http://localhost:8082/
```

**Reference:** GitHub issue https://github.com/artpar/apigate/issues/41

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

### Session 8 (February 6, 2026) - CURRENT SESSION

**Goal:** Create dedicated SSH Keys page, add web-UI app templates, E2E deploy a real app

**Accomplished:**

1. **Dedicated SSH Keys Page (`/ssh-keys`)**
   - Created `web/src/pages/ssh-keys/SSHKeysPage.tsx`
   - Table with columns: Name, Fingerprint, Used By (node badges), Created, Actions
   - Cross-references nodes via `ssh_key_id` — shows which nodes use each key
   - Delete confirmation warns if key is in use by nodes
   - Empty state with "Add SSH Key" action
   - Added route in `App.tsx` as protected route
   - Added `KeyRound` icon nav item in `Sidebar.tsx` (after "My Nodes")
   - Removed inline SSH keys summary card from `MyNodesPage.tsx`
   - "Manage SSH Keys" button replaced with `<Link to="/ssh-keys">`

2. **Web-UI App Templates (migration 008)**
   - Created `008_webui_templates.up.sql` with 6 templates
   - WordPress: Multi-service (wordpress + mysql), port 80, $8/mo
   - Uptime Kuma: Single container, monitoring dashboard, port 3001, $4/mo
   - Gitea: Self-hosted Git, port 3000, $5/mo
   - n8n: Workflow automation, port 5678, $6/mo
   - IT Tools: Developer utilities, port 80, $2/mo
   - Metabase: Business intelligence, port 3000, $7/mo
   - Applied to running database for immediate testing

3. **Full E2E Browser Testing (Chrome DevTools MCP)**
   - Logged in via Vite dev server (localhost:3000) proxying to APIGate (8082)
   - Verified marketplace shows all 12 templates
   - Verified SSH Keys page: table, node cross-references, sidebar nav, link from /nodes
   - Deployed Uptime Kuma from marketplace dialog
   - Container created and started in ~20 seconds
   - Accessed Uptime Kuma web UI at localhost:30003 — setup page loaded
   - Completed admin account setup — reached monitoring dashboard
   - Verified Hoster deployment detail: Overview (CPU/memory), Logs (100+ lines), Events (created/started)
   - Plan limit correctly blocked 2nd deployment ("max 1 deployments")

**Files Changed:**
- `web/src/pages/ssh-keys/SSHKeysPage.tsx` - NEW - Standalone SSH Keys page
- `web/src/App.tsx` - Added /ssh-keys route + import
- `web/src/components/layout/Sidebar.tsx` - Added SSH Keys nav item
- `web/src/pages/nodes/MyNodesPage.tsx` - Removed inline SSH key management, added link
- `internal/shell/store/migrations/008_webui_templates.up.sql` - NEW - 6 web-UI templates
- `internal/shell/store/migrations/008_webui_templates.down.sql` - NEW - Rollback
- `CLAUDE.md` - Updated status, template counts, implementation status

**Key Insights:**
- Vite dev server (port 3000) must be used for testing frontend code changes (hot-reload). APIGate (8082) serves old embedded frontend binary.
- Vite proxies `/api` to APIGate on 8082, so API calls work correctly.
- Plan limits are enforced by APIGate on `auth_required=1` routes (deployment CRUD).
- Uptime Kuma healthcheck shows "Unhealthy" initially because it uses a complex node.js health check — the app itself is fully functional.

**Next Steps:**
1. Build and embed frontend with docs registry + SSH Keys page
2. Create release v0.2.7 with docs registry + web-UI templates + SSH Keys page
3. Deploy to production and test ALL user journeys
4. Test real app deployment on production (Uptime Kuma via `*.apps.emptychair.dev`)

---

### Session 7 (January 27, 2026)

**Goal:** Fix production authentication & UX issues discovered via manual testing

**Context:** Manual testing on production (https://emptychair.dev) revealed critical issues that unit tests didn't catch:

**Issues Found:**
1. **APIGate Session Cookie Bug (BLOCKING)** - Filed: https://github.com/artpar/apigate/issues/54
   - User signs up successfully but no `Set-Cookie` header in response
   - Next `/auth/me` call returns 401 "Valid session or API key required"
   - User appears logged out immediately after signup
   - **EXTERNAL DEPENDENCY - WAITING FOR FIX**

2. **Privacy Bug (CRITICAL)** - Deployment list not filtered by user
   - `handleListDeployments()` accepted optional `customer_id` parameter
   - If not provided, returned ALL deployments (privacy violation)
   - Users could see other users' deployments
   - **FIXED IN THIS SESSION**

3. **Poor Auth UX** - User doesn't know if logged in
   - Header always showed "Sign in via APIGate" (no user profile)
   - No indication of authentication state
   - "Sign In Required" dialog not helpful
   - **FIXED IN THIS SESSION**

**Accomplished:**

1. **Created User Journeys Documentation:**
   - File: `specs/user-journeys.md`
   - 10 comprehensive user journeys covering all critical flows
   - Testing protocol with Chrome DevTools MCP automation
   - Test report template for production testing
   - Emphasis on manual testing before deployment

2. **Fixed Critical Privacy Bug:**
   - File: `internal/shell/api/handler.go` lines 555-598
   - Updated `handleListDeployments()` to ALWAYS filter by authenticated user
   - Added authentication check - 401 for unauthenticated requests
   - Template filtering now works in-memory after privacy enforcement
   - **CRITICAL:** Users can now ONLY see their own deployments

3. **Improved Auth UX - Header Component:**
   - File: `web/src/components/layout/Header.tsx`
   - Shows user profile (name/email) with user icon when authenticated
   - Added "Sign Out" button with logout functionality
   - "Sign In" button for unauthenticated users
   - Clear visual indication of auth state

4. **Improved Auth UX - Better Error Messages:**
   - File: `web/src/pages/marketplace/TemplateDetailPage.tsx`
   - Changed dialog title from "Sign In Required" to "Authentication Required"
   - Added context: "Your session may have expired. Please sign in to continue."
   - "Sign In" button navigates to login page
   - File: `web/src/components/ui/AlertDialog.tsx`
   - Added `onConfirm` callback support for custom actions

5. **Added Session Recovery:**
   - File: `web/src/stores/authStore.ts`
   - Window focus event listener checks auth state
   - Automatically calls `checkAuth()` when window regains focus (if unauthenticated)
   - Helps recover sessions that may have been restored by APIGate

6. **Updated CLAUDE.md with Testing Requirements:**
   - Added "Production Testing (MANDATORY)" section
   - Documents requirement to test ALL user journeys before deployment
   - Added "No-Bypass Policy (CRITICAL)" section
   - Lists forbidden actions (workarounds, skipping tests, etc.)
   - Lists required actions (fix root cause, wait for proper fixes)
   - **Bottom line:** If it's not production-ready, don't deploy it

**Files Changed:**
- `specs/user-journeys.md` - NEW - Comprehensive user journey documentation
- `internal/shell/api/handler.go` - CRITICAL - Privacy enforcement in deployment list
- `web/src/components/layout/Header.tsx` - User profile display + sign out
- `web/src/components/ui/AlertDialog.tsx` - Added onConfirm support
- `web/src/pages/marketplace/TemplateDetailPage.tsx` - Better auth error messages
- `web/src/stores/authStore.ts` - Session recovery on window focus
- `CLAUDE.md` - Production testing requirements + no-bypass policy

**Key Insights:**
- **Unit tests are necessary but not sufficient** - Must test as actual user
- Manual testing on production caught issues that unit tests missed
- Privacy bugs are critical - enforce filtering at API level, not client level
- UX issues (like "always looks logged out") break user trust
- Never bypass broken components - wait for proper fixes

**Blocked By:**
- APIGate issue #54 - Session cookies not set after signup/login
- Cannot complete full E2E auth testing until APIGate fix is deployed

**Next Steps:**
1. Wait for APIGate issue #54 fix
2. Test signup/login flow with session cookies
3. Complete all user journeys from `specs/user-journeys.md`
4. Create v0.2.5 release with privacy fix + auth UX improvements
5. Deploy to production and test ALL journeys manually

**DO NOT:**
- Deploy to production before APIGate fix
- Bypass auth issues with workarounds
- Skip manual testing "just this once"

---

### Session 6 (January 23, 2026)

**Goal:** Complete E2E testing with remote AWS EC2 node via web UI

**Accomplished:**

1. **Removed Native Dialogs from Codebase:**
   - Created `web/src/components/ui/ConfirmDialog.tsx` - Reusable confirmation dialog
   - Created `web/src/components/ui/AlertDialog.tsx` - Simple alert dialog
   - Updated `TemplateCard.tsx` - Delete template confirmation
   - Updated `DeploymentDetailPage.tsx` - Delete deployment confirmation
   - Updated `TemplateDetailPage.tsx` - Sign-in required alert
   - Updated `CreatorDashboardPage.tsx` - Delete SSH key and node confirmations
   - Verified with grep: no native confirm()/alert() calls remain

2. **Remote Node E2E Testing:**
   - AWS EC2 instance already running at 98.82.190.29
   - Re-added SSH key via web UI (encryption key: 12345678901234567890123456789012)
   - Registered node "aws-test-node" via Creator Dashboard
   - Node came online successfully (health check passed)
   - Fixed encryption key mismatch issue by cleaning up old nodes/keys

3. **Deployment to Remote Node:**
   - Created deployment from "Test App" template
   - Fixed `canStart` condition to include 'pending' status in DeploymentDetailPage
   - Started deployment via web UI - scheduled to aws-test-node
   - Container `nginx:alpine` pulled and started on remote EC2
   - Events recorded: container_created, container_started
   - Deployment status: **running** on node 98.82.190.29

**Deployment Details:**
- Deployment ID: `depl_075f7cdb`
- Name: `test-app-mkqrkyep`
- Node: `aws-test-node` (98.82.190.29)
- Domain: `test-app-mkqrkyep.apps.localhost`
- Proxy Port: 30000

**Files Changed:**
- `web/src/components/ui/ConfirmDialog.tsx` - NEW
- `web/src/components/ui/AlertDialog.tsx` - NEW
- `web/src/components/templates/TemplateCard.tsx` - Replaced confirm() with ConfirmDialog
- `web/src/pages/deployments/DeploymentDetailPage.tsx` - Added ConfirmDialog + fixed canStart
- `web/src/pages/marketplace/TemplateDetailPage.tsx` - Replaced alert() with AlertDialog
- `web/src/pages/creator/CreatorDashboardPage.tsx` - Added ConfirmDialogs for SSH key/node deletion

**Key Technical Insights:**
- Encryption key must be exactly 32 bytes for AES-256-GCM
- Scheduler assigns deployments to nodes owned by the **template creator**, not deployment customer
- Deployment state machine: pending → scheduled → starting → running
- `canStart` condition needed to include 'pending' status for new deployments

**E2E Test Complete:**
| Step | Status |
|------|--------|
| EC2 instance with Docker | ✅ Verified at 98.82.190.29 |
| SSH key via web UI | ✅ Added and encrypted |
| Node registration via web UI | ✅ aws-test-node online |
| Deploy template to remote | ✅ Scheduled and started |
| Container running on remote | ✅ nginx:alpine running |

---

### Session 5 (January 22, 2026)

**Goal:** Complete monitoring features, fix subdomain routing, and prepare for production deployment

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

6. **APIGate Subdomain Routing Fix (Issue #41):**
   - Identified 4 bugs preventing host-based routing in APIGate
   - Filed comprehensive GitHub issue: https://github.com/artpar/apigate/issues/41
   - Issues identified:
     1. Host header not extracted from `r.Host` (Go stores it separately)
     2. Chi router routes taking precedence over database routes with host patterns
     3. Host header not preserved when proxying to upstream
     4. Public route detection happening after API key check
   - Fix merged in APIGate commit `5d72804`
   - Verified subdomain routing working: `curl -H "Host: nginx-web-server-mkpnwv4e.apps.localhost" http://localhost:8082/` returns nginx welcome page

**Files Changed:**
- `internal/shell/docker/orchestrator.go` - Added event recording infrastructure
- `internal/shell/api/handler.go` - Updated NewOrchestrator calls (5 locations)
- `internal/shell/api/resources/deployment.go` - Updated NewOrchestrator calls (4 locations)
- `internal/shell/store/migrations/007_default_templates.up.sql` - NEW - Default templates
- `internal/shell/store/migrations/007_default_templates.down.sql` - NEW - Rollback migration

**APIGate Changes (in artpar/apigate repo, NOT hoster):**
- `adapters/http/handler.go` - Host header extraction + priority middleware
- `adapters/http/upstream.go` - Host header preservation in all Forward methods
- `app/proxy.go` - Public route detection improvements

**E2E Test Results - ALL PASSING:**
- ✅ Frontend access through APIGate (redirects to /login)
- ✅ API access through APIGate (returns 7 templates without auth)
- ✅ App proxy subdomain routing (nginx welcome page via `*.apps.localhost`)
- ✅ Billing disabled (by design for local testing)

**Not Completed:**
- APIGate auto-registration (401 errors when accessing admin endpoints through proxy)
- Production deployment (needs CI fixes first)

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

## Remote Node Infrastructure

**AWS EC2 Test Node:**
- **IP Address:** 98.82.190.29
- **Node Name:** aws-test-node
- **SSH User:** deploy
- **Docker:** Installed and running
- **Status:** Online (verified January 23, 2026)

**Encryption Key for SSH Keys:**
```
HOSTER_ENCRYPTION_KEY=12345678901234567890123456789012
```
Note: This is the 32-byte key used to encrypt SSH private keys in the database. If changed, existing encrypted keys become unreadable.

**Testing Remote Deployments:**
1. Ensure node is online in Creator Dashboard
2. Create deployment from a template owned by the node's creator
3. Start deployment - scheduler will assign to available node
4. Verify via Events tab: container_created, container_started

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

**For remote node testing:**
1. [ ] Ensure encryption key matches: `HOSTER_ENCRYPTION_KEY=12345678901234567890123456789012`
2. [ ] Start Hoster with database: `HOSTER_DATABASE_DSN=/tmp/hoster-e2e-test/hoster.db`
3. [ ] Verify node is online in Creator Dashboard
4. [ ] Deploy template and verify on remote node
