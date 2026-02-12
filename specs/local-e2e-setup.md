# Local E2E Testing Environment

> Moved from CLAUDE.md to reduce file size.

## Current Setup (February 2026 — APIGate v0.3.6)

**Location:** `/tmp/hoster-e2e-test/`

**Architecture:**
```
Browser → APIGate (:8082, front-facing) → Hoster (:8080, backend + embedded SPA)

Routes (priority order):
    ├── hoster-billing   (/api/v1/deployments* priority 55, auth_required=1) ← billing
    ├── hoster-api       (/api/* priority 50, auth_required=1) ← authenticated
    └── hoster-front     (/* priority 10, auth_required=0) ← SPA
```

**Services:**
- APIGate: localhost:8082 (single entry point — auth + billing + routing)
- Hoster: localhost:8080 (API + embedded frontend SPA)
- App Proxy: localhost:9091 (deployment routing)

**Important:**
- All access MUST go through APIGate (localhost:8082). Never access Hoster directly.
- This is a prod-like setup: no Vite dev server, no hot-reload.
- APIGate v0.3.6 uses **environment variables**, not YAML config files.
- `auth_required=1` routes: APIGate validates JWT, injects `X-User-ID`, `X-Plan-ID` headers.
- `auth_required=0` routes: APIGate **strips** the Authorization header before forwarding.

## One-Time Setup

### 1. Download APIGate v0.3.6

```bash
mkdir -p /tmp/hoster-e2e-test
cd /tmp/hoster-e2e-test
gh release download v0.3.6 --repo artpar/apigate --pattern 'apigate-darwin-arm64'
chmod +x apigate-darwin-arm64
```

### 2. Build Frontend + Hoster

```bash
cd /Users/artpar/workspace/code/hoster/web
npm run build
mkdir -p ../internal/engine/webui && cp -r dist ../internal/engine/webui/

cd /Users/artpar/workspace/code/hoster
go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster
```

### 3. Start APIGate (first run triggers setup wizard)

```bash
cd /tmp/hoster-e2e-test
APIGATE_DATABASE_DSN=/tmp/hoster-e2e-test/apigate.db \
APIGATE_SERVER_PORT=8082 \
./apigate-darwin-arm64
```

### 4. Complete Setup Wizard

On first run, navigate to `http://localhost:8082/setup` and complete the wizard:
- Set admin credentials
- Configure upstream (Hoster at `http://localhost:8080`)

### 5. Create Routes via Admin UI

Navigate to `http://localhost:8082/admin/routes` and create these routes:

| Name | Path Pattern | Upstream | auth_required | Priority |
|------|-------------|----------|---------------|----------|
| `hoster-billing` | `/api/v1/deployments*` | `http://localhost:8080` | 1 | 55 |
| `hoster-api` | `/api/*` | `http://localhost:8080` | 1 | 50 |
| `hoster-front` | `/*` | `http://localhost:8080` | 0 | 10 |

**Note:** `hoster-api` uses `auth_required=1` so APIGate validates JWT and injects `X-User-ID` on all authenticated API routes. Only `hoster-front` (SPA) is public.

### 6. Register a User

Navigate to `http://localhost:8082` and use the Sign Up page to create a test user.

## Starting the Environment

```bash
# Terminal 1: Start APIGate
cd /tmp/hoster-e2e-test
APIGATE_DATABASE_DSN=/tmp/hoster-e2e-test/apigate.db \
APIGATE_SERVER_PORT=8082 \
./apigate-darwin-arm64

# Terminal 2: Start Hoster
cd /tmp/hoster-e2e-test
HOSTER_DATA_DIR=/tmp/hoster-e2e-test \
./hoster
```

**Note:** `HOSTER_DATA_DIR` sets both the database path (`hoster.db`) and config directory. Using the same data dir across restarts preserves state.

## Rebuilding After Code Changes

If you change backend code:
```bash
cd /Users/artpar/workspace/code/hoster
go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster
# Restart Hoster in Terminal 2
```

If you change frontend code:
```bash
cd /Users/artpar/workspace/code/hoster/web
npm run build
mkdir -p ../internal/engine/webui && cp -r dist ../internal/engine/webui/
cd /Users/artpar/workspace/code/hoster
go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster
# Restart Hoster in Terminal 2
```

## Checking Status

```bash
ps aux | grep -E "(apigate|hoster)" | grep -v grep
lsof -i :8082  # APIGate
lsof -i :8080  # Hoster
lsof -i :9091  # App Proxy
```

## Testing E2E Flow

All access through `http://localhost:8082` (APIGate).

1. **Access Frontend:** http://localhost:8082/
2. **Sign Up:** Create a test user via the Sign Up page
3. **Dashboard:** Verify redirect after login, see empty deployments
4. **Browse Marketplace:** Click Marketplace, verify templates visible
5. **Deploy template:** Select a template, fill in name, submit
6. **Monitor deployment:** Check status transitions in deployment detail
7. **Check billing:** Navigate to Billing & Usage, verify billing events appear
8. **Stop deployment:** Click Stop, verify status changes
9. **Delete deployment:** Click Delete, verify cleanup
10. **Final billing check:** All event types should appear in billing page

## Database Files

- **Hoster:** `/tmp/hoster-e2e-test/hoster.db`
- **APIGate:** `/tmp/hoster-e2e-test/apigate.db`

## Routes Configuration

| Route | Path | auth_required | Priority | Purpose |
|-------|------|---------------|----------|---------|
| `hoster-billing` | `/api/v1/deployments*` | 1 (billing) | 55 | Deployment CRUD - billable + metered |
| `hoster-api` | `/api/*` | 1 (auth) | 50 | All other APIs - authenticated |
| `hoster-front` | `/*` | 0 (public) | 10 | SPA frontend |

### Auth Behavior by Route Type

- **auth_required=1**: APIGate validates JWT, injects `X-User-ID`/`X-Plan-ID`/`X-Plan-Limits` headers, runs billing pipeline (quota/metering on billing routes). Hoster reads injected headers.
- **auth_required=0**: APIGate **strips** Authorization header. Request forwarded without auth context. Used only for public routes (SPA shell).

## Troubleshooting

**Frontend shows "Frontend not built":** Run the build steps in "Build Frontend + Hoster" section.
**Frontend 404:** Check APIGate (8082), Hoster (8080), and route config.
**Auth not working:** Ensure routes use `auth_required=1`. With `auth_required=0`, APIGate strips the Authorization header and Hoster receives no user identity.
**Billing events missing:** Check APIGate meter endpoint: `GET /mod/meter/events` via admin. Verify `event_type` field is populated (requires APIGate v0.3.6+).
**Monitoring empty:** Monitoring routes go through `hoster-api` (auth_required=1). JWT must be valid.
**Can't access deployed app:** Check App Proxy (9091), deployment domain, URL format `http://{name}.apps.localhost:8082/`.
**Setup wizard keeps appearing:** The setup wizard only runs on first start. If it reappears, the database may have been deleted.
