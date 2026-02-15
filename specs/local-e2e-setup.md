# Local E2E Testing Environment

## Quick Reference

| Service | Port | Purpose |
|---------|------|---------|
| APIGate | 8082 | Front-facing: auth + billing + routing |
| Hoster | 8080 | Backend: API + embedded SPA frontend |
| Vite | 3000 | Dev hot-reload only (NOT for testing) |
| App Proxy | 9091 | Deployment routing (`*.apps.localhost`) |

**ALL access goes through APIGate (localhost:8082). NEVER access Hoster (8080) or Vite (3000) directly.**

## APIGate Admin Credentials

| Field | Value |
|-------|-------|
| Email | `admin@hoster.local` |
| Password | `Admin1234secure` |
| Admin URL | `http://localhost:8082/admin` |

## Directory

Everything lives in `/tmp/hoster-e2e-test/`:

| File | Purpose |
|------|---------|
| `hoster` | Hoster binary (built from source with embedded frontend) |
| `apigate-darwin-arm64` | APIGate binary (downloaded from GitHub releases) |
| `hoster.db` | Hoster SQLite database |
| `apigate.db` | APIGate SQLite database |
| `hoster.log` | Hoster stdout/stderr log |
| `apigate.log` | APIGate stdout/stderr log |
| `start.sh` | Hoster launch script (env vars + exec) |
| `.e2e-rebuilt` | Marker file from rebuild hook |

## Routes Configuration

| Name | Path Pattern | Upstream | auth_required | Priority | Metering | Purpose |
|------|-------------|----------|---------------|----------|----------|---------|
| `hoster-billing` | `/api/v1/deployments*` | `http://localhost:8080` | 1 (billing) | 55 | Per Request (expr: `1`) | Deployment CRUD — billable + metered |
| `hoster-api` | `/api/*` | `http://localhost:8080` | 1 (auth) | 50 | Custom (expr: `0`) | All other APIs — auth only, NO metering |
| `hoster-front` | `/*` | `http://localhost:8080` | 0 (public) | 10 | Custom (expr: `0`) | SPA frontend — public |

**CRITICAL**: `hoster-api` metering expression MUST be `0`. If set to `1` (default), every API call counts against the plan quota and you'll hit "Monthly request quota exceeded" quickly.

**NOTE**: The `hoster-api` route MUST exist for auth enforcement on API paths. Without it, API calls fall through to `hoster-front` (auth_required=0) which strips the Authorization header.

### Auth Behavior by Route Type

- **auth_required=1**: APIGate validates JWT, injects `X-User-ID`/`X-Plan-ID`/`X-Plan-Limits` headers. On billing routes (metering expr > 0), also runs quota check and records usage.
- **auth_required=0**: APIGate **strips** Authorization header. Request forwarded without auth context. Used only for public routes (SPA shell).

## Setup From Scratch

### Prerequisites

- Go installed
- Node.js + npm installed
- `gh` CLI installed and authenticated
- DigitalOcean API key (MUST set `TEST_DO_API_KEY` environment variable — no hardcoded fallback)

### Step 1: Download APIGate Binary

```bash
mkdir -p /tmp/hoster-e2e-test
cd /tmp/hoster-e2e-test

# Download latest APIGate release (currently v0.3.8)
gh release download v0.3.8 --repo artpar/apigate --pattern 'apigate-darwin-arm64*'
# If downloaded as tar.gz:
tar xzf apigate-darwin-arm64.tar.gz
chmod +x apigate-darwin-arm64
```

### Step 2: Build Frontend + Hoster Binary

```bash
# Build React frontend
cd /Users/artpar/workspace/code/hoster/web
npm run build

# Copy built frontend to embed directory
# CRITICAL: go:embed reads from webui/dist/ — NOT webui/ directly
rm -rf ../internal/engine/webui/dist
cp -r dist ../internal/engine/webui/dist

# Build Hoster binary with embedded frontend
cd /Users/artpar/workspace/code/hoster
go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster
```

**WARNING**: The copy command MUST be `cp -r dist ../internal/engine/webui/dist` (creates `webui/dist/` directory). NOT `cp -r dist/* ../internal/engine/webui/` (copies into `webui/` level). The Go embed directive is `//go:embed all:webui/dist` and reads from `internal/engine/webui/dist/`. Getting this wrong causes the old embedded frontend to be served, leading to stale routes (e.g., `/login` instead of `/sign-in`).

### Step 3: Create Hoster Start Script

Create `/tmp/hoster-e2e-test/start.sh` — needed because env vars don't reliably pass to backgrounded processes:

```bash
cat > /tmp/hoster-e2e-test/start.sh << 'EOF'
#!/bin/bash
export HOSTER_DATA_DIR=/tmp/hoster-e2e-test
export HOSTER_NODES_ENCRYPTION_KEY='e2e-test-encryption-key-32bytes!'
export HOSTER_BILLING_API_KEY='<your-apigate-api-key>'
cd /tmp/hoster-e2e-test
exec ./hoster >> hoster.log 2>&1
EOF
chmod +x /tmp/hoster-e2e-test/start.sh
```

**Note**: The encryption key is exactly 32 bytes (required for AES-256-GCM). Using `env VAR=val ./binary &` in some shells causes the env var to not propagate to the background process — hence the script.

**Note**: `HOSTER_BILLING_API_KEY` is set after Step 7 (Create Billing API Key) below. Leave it as a placeholder on first run and update after creating the key.

### Step 4: Start Services

```bash
# Start Hoster (via start script)
/tmp/hoster-e2e-test/start.sh &

# Wait a moment for Hoster to bind to :8080
sleep 2

# Start APIGate
cd /tmp/hoster-e2e-test
APIGATE_DATABASE_DSN=/tmp/hoster-e2e-test/apigate.db \
APIGATE_SERVER_PORT=8082 \
./apigate-darwin-arm64 serve >> apigate.log 2>&1 &
```

Verify both are running:
```bash
lsof -i :8080  # Hoster
lsof -i :8082  # APIGate
```

### Step 5: APIGate Setup Wizard (first run only)

Navigate to `http://localhost:8082/setup` in a browser.

1. **Step 1 — Connect API**: Enter `http://localhost:8080`
2. **Step 2 — Create Account**:
   - Name: `Admin`
   - Email: `admin@hoster.local`
   - Password: `Admin1234secure`
3. **Step 3 — Pricing**:
   - Plan name: `free`
   - Monthly price: `$0`
   - Requests/min: `600`
   - Monthly limit: `100000` (high enough for testing; `0` should mean unlimited per docs but has a bug)
4. **Step 4**: Done

**Post-setup fix**: The setup wizard assigns `plan_id='admin'` to the admin user, but no 'admin' plan exists. Fix via admin UI:
- Go to `http://localhost:8082/admin` → Users section
- Find `admin@hoster.local` → Edit → Change plan to `free`
- Save

### Step 6: Configure Routes

Navigate to `http://localhost:8082/admin` → Routes section.

The setup wizard creates a default `/*` route. You need to add two more routes and configure metering:

1. **hoster-billing** (highest priority — catches deployment CRUD first):
   - Name: `hoster-billing`
   - Path: `/api/v1/deployments*`
   - Upstream: `http://localhost:8080`
   - auth_required: `1`
   - Priority: `55`
   - Metering: Per Request, expression: `1`

2. **hoster-api** (catches all other API calls):
   - Name: `hoster-api`
   - Path: `/api/*`
   - Upstream: `http://localhost:8080`
   - auth_required: `1`
   - Priority: `50`
   - Metering: Custom, expression: **`0`** (CRITICAL — must be zero!)

3. **hoster-front** (catch-all for SPA — already exists from wizard as "Default Route"):
   - Name: `hoster-front`
   - Path: `/*`
   - Upstream: `http://localhost:8080`
   - auth_required: `0`
   - Priority: `10`
   - Metering: Custom, expression: `0`

**CRITICAL**: The `hoster-api` metering expression MUST be `0`. If set to `1` (the default for auth_required=1 routes), every single API call (list templates, get deployments, check stats, etc.) counts against the plan's monthly request quota. With a 100000 limit, you'll hit "Monthly request quota exceeded" within minutes of testing.

Only `hoster-billing` should meter (expression: `1`) — this counts deployment create/start/stop/delete operations against the plan quota.

### Step 7: Create Billing API Key

Hoster's billing reporter sends usage events to APIGate's metering endpoint. It authenticates with an API key.

1. Go to `http://localhost:8082/admin` → Keys section
2. Create a new API key for the admin user
3. Copy the key (e.g., `ak_6aa3f08c...`)
4. Update `/tmp/hoster-e2e-test/start.sh` — set `HOSTER_BILLING_API_KEY` to the key value
5. Restart Hoster to pick up the key

### Step 8: Configure Meter Path (APIGate v0.3.8+)

APIGate's metering endpoint is configurable via the `routes.meter_base_path` setting. The default is `/api/v1/meter`, but this can conflict with the `hoster-api` route (which catches `/api/*`). Change it to `/_internal/meter`:

1. Go to `http://localhost:8082/admin` → Settings
2. Find or add `routes.meter_base_path`
3. Set value to `/_internal/meter`
4. Restart APIGate

Hoster's billing client defaults to `/_internal/meter` as of v0.3.49+.

## Running Playwright E2E Tests

### Test Architecture

```
Global Setup (provisions real DO droplet, ~2 min)
    ↓
UJ1 → UJ2 → UJ3 → UJ4 → UJ5 → UJ6 → UJ7 → UJ8  (sequential, 1 worker)
    ↓
Global Teardown (destroys DO droplet)
```

- **Global setup** (`web/e2e/global-setup.ts`): Signs up a test user, creates a cloud credential with a real DigitalOcean API key, provisions a real droplet (sfo3, s-1vcpu-1gb), creates and publishes a test template (nginx:alpine), writes state to `web/e2e/.e2e-infra.json`
- **Global teardown** (`web/e2e/global-teardown.ts`): Destroys the droplet via UI, deletes deployments/template/credential, then **verifies 0 leaked droplets via DO API** (force-destroys any found and fails the test)
- **Tests share one droplet**: All 8 user journeys share the same DO droplet to minimize cost (~$0.005 per run)
- **UJ4 additionally provisions its own second droplet** to test the cloud provisioning UI flow

### Prerequisites for Running Tests

1. **Hoster + APIGate must be running** (Steps 4-6 above)
2. **Frontend must be rebuilt and embedded** (Step 2 above)
3. **Playwright browsers installed**: `cd web && npx playwright install chromium`

### Running Tests

```bash
# ALWAYS run from the web/ directory — NOT from the repo root
cd /Users/artpar/workspace/code/hoster/web

# Set the DigitalOcean API key (required for real infrastructure tests)
export TEST_DO_API_KEY='dop_v1_your_key_here'

# Run all tests (provisions real droplet, ~10-15 min)
npx playwright test

# Run with visible browser
npx playwright test --headed

# Run a single journey
npx playwright test e2e/uj1-discovery.spec.ts
npx playwright test e2e/uj2-first-deployment.spec.ts
npx playwright test e2e/uj3-day2-operations.spec.ts
npx playwright test e2e/uj4-infrastructure-scaling.spec.ts
npx playwright test e2e/uj5-creator-monetization.spec.ts
npx playwright test e2e/uj6-billing-cycle.spec.ts
npx playwright test e2e/uj7-session-recovery.spec.ts
npx playwright test e2e/uj8-teardown-cleanup.spec.ts

# View test report
npx playwright show-report
```

**WARNING**: Running `npx playwright test` from the repo root may pick up a different Playwright version and fail. Always `cd web` first.

### Test Configuration (`web/playwright.config.ts`)

- `baseURL: http://localhost:8082` (all tests go through APIGate)
- `workers: 1` (sequential — tests share state and plan limits)
- `retries: 1` (flaky tests get one retry)
- `timeout: 60_000` default (individual tests override with `test.setTimeout()` for long operations)
- `globalSetup/globalTeardown` provisions and destroys real infrastructure

### Test Suites (8 User Journeys)

| File | Journey | Tests | Real Infra? | Duration |
|------|---------|-------|-------------|----------|
| `uj1-discovery.spec.ts` | Template browsing | 9 | No (UI only) | ~15s |
| `uj2-first-deployment.spec.ts` | First deployment | 10 | Yes (deploys to shared droplet) | ~1 min |
| `uj3-day2-operations.spec.ts` | Day-2 operations | 10 | Yes (logs, stats, stop/start) | ~5 min |
| `uj4-infrastructure-scaling.spec.ts` | Infrastructure scaling | 8 | Yes (provisions own droplet) | ~3 min |
| `uj5-creator-monetization.spec.ts` | Template CRUD | 8 | No (UI only) | ~15s |
| `uj6-billing-cycle.spec.ts` | Billing page | 7 | No (UI only) | ~15s |
| `uj7-session-recovery.spec.ts` | Auth flows | 6 | No (UI only) | ~15s |
| `uj8-teardown-cleanup.spec.ts` | Teardown & cleanup | 7 | Yes (stop/delete lifecycle) | ~3 min |

**Total**: 65 tests, ~10-15 minutes with real infrastructure.

### Plan Limits and Test Isolation

The free plan allows **1 deployment max**. Tests that create deployments clean up existing ones in `beforeAll`:

```typescript
// Clean up any existing deployments from earlier test suites (plan limit = 1)
const existing = await apiListDeployments(token);
for (const d of existing) {
  const status = d.attributes.status as string;
  if (status === 'running') await apiStopDeployment(token, d.id).catch(() => {});
  if (status !== 'deleted') {
    await new Promise(r => setTimeout(r, 2000));
    await apiDeleteDeployment(token, d.id).catch(() => {});
  }
}
```

This is critical because UJ2 creates a deployment that may still exist when UJ3/UJ8 run.

### Infrastructure State File

Global setup writes `web/e2e/.e2e-infra.json` (gitignored):

```json
{
  "token": "jwt...",
  "email": "e2e-xxx@test.local",
  "nodeId": "node_xxx",
  "templateId": "tmpl_xxx",
  "provisionId": "prov_xxx",
  "credentialId": "cred_xxx",
  "sshKeyId": "sshkey_xxx",
  "dropletIp": "x.x.x.x"
}
```

Tests read this via `readInfraState()` from `web/e2e/fixtures/test-data.ts`.

## Rebuilding After Code Changes

### Frontend changes only:
```bash
cd /Users/artpar/workspace/code/hoster/web && npm run build
rm -rf ../internal/engine/webui/dist && cp -r dist ../internal/engine/webui/dist
cd .. && go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster
# Restart Hoster
lsof -i :8080 -t | xargs kill 2>/dev/null
/tmp/hoster-e2e-test/start.sh &
```

### Backend changes only:
```bash
cd /Users/artpar/workspace/code/hoster
go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster
# Restart Hoster
lsof -i :8080 -t | xargs kill 2>/dev/null
/tmp/hoster-e2e-test/start.sh &
```

### Automated rebuild (runs before E2E tests):
A hook in `.claude/hooks/rebuild-before-e2e.sh` automatically reminds to rebuild when Playwright tests are run.

## Checking Status

```bash
lsof -i :8080  # Hoster
lsof -i :8082  # APIGate
lsof -i :3000  # Vite (dev only)
lsof -i :9091  # App Proxy
```

## APIGate Reserved Paths (DO NOT USE IN HOSTER FRONTEND)

APIGate intercepts these root-level paths for its admin UI — they are NOT forwarded to Hoster even with a `/*` catch-all route:

| Path | APIGate Purpose | Hoster Alternative |
|------|----------------|--------------------|
| `/login` | Admin login (302 → `/dashboard` if logged in) | `/sign-in` |
| `/signup` | Redirects to `/portal/signup` | `/sign-up` |
| `/dashboard` | Admin dashboard | `/home` |
| `/admin` | Admin panel | — |
| `/portal` | Customer portal | — |
| `/auth` | Auth API endpoints | — |
| `/docs` | Documentation portal | — |
| `/mod` | Module handler (NEEDED for auth API) | — |
| `/health` | Health check | — |
| `/metrics` | Metrics | — |
| `/setup` | First-run wizard | — |
| `/routes`, `/users`, `/keys`, `/plans`, `/usage`, `/payments`, `/email`, `/webhooks`, `/settings`, `/invites`, `/system` | Admin sub-pages | — |

The Handler Routes Configuration in APIGate Settings only affects API endpoints, NOT the admin UI pages. These root-level paths are hardcoded in the APIGate binary.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Frontend shows stale UI at :8082 | Rebuild: `cd web && npm run build && rm -rf ../internal/engine/webui/dist && cp -r dist ../internal/engine/webui/dist && cd .. && go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster` then restart Hoster |
| Browser redirects to `/login` instead of `/sign-in` | Stale embedded frontend — the `webui/dist/` has old bundles. Run the full rebuild above. |
| APIGate setup wizard reappears | Database was deleted. Re-run setup wizard (Step 5). |
| Auth not working on API calls | Check routes have `auth_required=1`. With `auth_required=0`, APIGate strips Authorization header. |
| "Monthly request quota exceeded" | `hoster-api` route metering expression is `1` instead of `0`. Fix in admin → Routes → hoster-api → set metering expression to `0`. |
| "plan limit reached: maximum 1 deployments" | Free plan allows only 1 deployment. Delete existing deployments before creating a new one. Tests handle this in `beforeAll`. |
| `bind: address already in use` | Kill old process: `lsof -i :PORT -t \| xargs kill` |
| Monitoring data empty | JWT must be valid. Monitoring routes go through `hoster-api` (auth_required=1). |
| Port 9091 conflict | Kill old App Proxy: `lsof -i :9091 -t \| xargs kill` |
| Billing 401 `missing_api_key` | `HOSTER_BILLING_API_KEY` not set in `start.sh`. Create an API key in APIGate admin → Keys, then add it to `start.sh` and restart Hoster. |
| Billing 404 on meter endpoint | The `hoster-api` route (`/api/*`) is shadowing APIGate's built-in metering endpoint. Set `routes.meter_base_path` to `/_internal/meter` in APIGate settings (Step 8). Requires APIGate v0.3.8+. |
| Hoster doesn't start in background | Use `start.sh` script instead of inline env vars. Background processes may not inherit env vars. |
| Encryption key "must be exactly 32 bytes" | The key `e2e-test-encryption-key-32bytes!` is exactly 32 bytes. If using inline env vars with `&`, use the start.sh script instead. |
| Playwright picks up wrong version | Always run from `cd web && npx playwright test`, NOT from the repo root. |
| E2E tests fail with "No infrastructure state" | Global setup didn't run or failed. Check that Hoster+APIGate are running, then run tests from `web/` directory. |
| Orphaned DO droplets after test failure | Teardown now auto-detects leaked droplets via DO API and force-destroys them. If tests crash before teardown runs, check `TEST_DO_API_KEY` env var is set and run teardown manually or check DO console. |

## Reset Everything

```bash
# Stop services
lsof -i :8080 -t | xargs kill 2>/dev/null
lsof -i :8082 -t | xargs kill 2>/dev/null

# Delete databases (DESTRUCTIVE)
rm -f /tmp/hoster-e2e-test/hoster.db /tmp/hoster-e2e-test/apigate.db
rm -f /tmp/hoster-e2e-test/*.db-shm /tmp/hoster-e2e-test/*.db-wal

# Rebuild
cd /Users/artpar/workspace/code/hoster/web && npm run build
rm -rf ../internal/engine/webui/dist && cp -r dist ../internal/engine/webui/dist
cd .. && go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster

# Start fresh (will trigger APIGate setup wizard at http://localhost:8082/setup)
/tmp/hoster-e2e-test/start.sh &
sleep 2
cd /tmp/hoster-e2e-test && APIGATE_DATABASE_DSN=/tmp/hoster-e2e-test/apigate.db APIGATE_SERVER_PORT=8082 ./apigate-darwin-arm64 serve >> apigate.log 2>&1 &
```

After reset, re-run the setup wizard (Step 5) and configure routes (Step 6).

## Cost

- Smallest DO droplet: `s-1vcpu-1gb` = $0.009/hour
- Two droplets for ~15 min (one shared, one for UJ4) = ~$0.005 total per test run
- Global teardown ensures no orphaned infrastructure
