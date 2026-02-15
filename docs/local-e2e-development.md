# Local E2E Development Guide

This guide covers running APIGate + Hoster together locally for full production-like end-to-end testing.

For detailed step-by-step setup instructions, see **`specs/local-e2e-setup.md`**.

## Architecture Overview

```
Browser
    |
    v
+-------------------------------+
|   APIGate (localhost:8082)    |
|   - JWT Auth                  |
|   - Billing / Quota           |
|   - Route-based forwarding    |
+-------------+-----------------+
              | Injects: X-User-ID, X-Plan-ID, X-Plan-Limits
              v
+-------------------------------+
|   Hoster API (localhost:8080) |
|   - Templates API             |
|   - Deployments API           |
|   - Nodes API                 |
|   - Embedded SPA frontend     |
+-------------+-----------------+
              |
              v
+-------------------------------+
| App Proxy (localhost:9091)    |
| *.apps.localhost -> containers|
+-------------------------------+
```

**ALL access goes through APIGate (localhost:8082).** Never access Hoster (8080) directly.

## Service URLs

| Service | Port | Purpose |
|---------|------|---------|
| APIGate | 8082 | Front-facing: JWT auth + billing + routing |
| Hoster | 8080 | Backend: API + embedded SPA frontend |
| Vite | 3000 | Dev hot-reload only (NOT for testing) |
| App Proxy | 9091 | Deployment routing (`*.apps.localhost`) |

## Prerequisites

- Go 1.21+
- Node.js + npm
- `gh` CLI installed and authenticated
- DigitalOcean API key (set `TEST_DO_API_KEY` env var for E2E tests)

## Quick Start

```bash
# 1. Build frontend + Hoster binary
cd /Users/artpar/workspace/code/hoster/web && npm run build
rm -rf ../internal/engine/webui/dist && cp -r dist ../internal/engine/webui/dist
cd .. && go build -o /tmp/hoster-e2e-test/hoster ./cmd/hoster

# 2. Download APIGate v0.3.8+
cd /tmp/hoster-e2e-test
gh release download v0.3.8 --repo artpar/apigate --pattern 'apigate-darwin-arm64*'
tar xzf apigate-darwin-arm64.tar.gz && chmod +x apigate-darwin-arm64

# 3. Start Hoster
/tmp/hoster-e2e-test/start.sh &
sleep 2

# 4. Start APIGate
cd /tmp/hoster-e2e-test
APIGATE_DATABASE_DSN=/tmp/hoster-e2e-test/apigate.db \
APIGATE_SERVER_PORT=8082 \
./apigate-darwin-arm64 serve >> apigate.log 2>&1 &
```

On first run, complete the APIGate setup wizard at `http://localhost:8082/setup`. See `specs/local-e2e-setup.md` Steps 5-8 for details.

## Auth Flow

Auth is entirely handled by APIGate. Hoster has NO auth endpoints.

1. Frontend shows `/sign-in` and `/sign-up` pages (Hoster-branded)
2. These pages call APIGate endpoints: `/mod/auth/login`, `/mod/auth/register`
3. APIGate returns a JWT token
4. Frontend stores JWT in localStorage, sends as `Authorization: Bearer {token}`
5. APIGate validates JWT, injects `X-User-ID`, `X-Plan-ID`, `X-Plan-Limits` headers
6. Hoster reads injected headers via `ResolveUser()` middleware

## Routes Configuration

| Name | Path Pattern | auth_required | Priority | Purpose |
|------|-------------|---------------|----------|---------|
| `hoster-billing` | `/api/v1/deployments*` | 1 (billing) | 55 | Deployment CRUD — billable + metered |
| `hoster-api` | `/api/*` | 1 (auth) | 50 | All other APIs — auth only, NO metering |
| `hoster-front` | `/*` | 0 (public) | 10 | SPA frontend — public |

All routes upstream to `http://localhost:8080` (Hoster).

## Billing Configuration

Hoster reports usage events to APIGate's metering endpoint:

- **API Key**: Create in APIGate admin (`/admin` → Keys). Set as `HOSTER_BILLING_API_KEY` in `start.sh`.
- **Meter Path**: APIGate v0.3.8+ supports configurable meter path via `routes.meter_base_path` setting. Set to `/_internal/meter` to avoid conflict with the `hoster-api` route at `/api/*`.
- **Hoster default**: Billing client uses `/_internal/meter` as default path.

## Header Contract

When running with APIGate, these headers are injected on `auth_required=1` routes:

| Header | Description |
|--------|-------------|
| `X-User-ID` | Authenticated user's UUID |
| `X-Plan-ID` | User's subscription plan |
| `X-Plan-Limits` | JSON with resource limits |
| `X-Key-ID` | API key identifier |

On `auth_required=0` routes, APIGate **strips** the Authorization header — no auth context is forwarded.

## Environment Variables (Hoster)

| Variable | Default | Description |
|----------|---------|-------------|
| `HOSTER_DATA_DIR` | `./data` | Base directory for DB and configs |
| `HOSTER_NODES_ENCRYPTION_KEY` | — | 32-byte key for AES-256-GCM SSH key encryption |
| `HOSTER_BILLING_API_KEY` | — | APIGate API key for metering requests |
| `HOSTER_PROXY_PORT` | `9091` | App proxy listen port |
| `HOSTER_PROXY_BASE_DOMAIN` | — | Base domain for app routing |
| `HOSTER_DOMAIN_BASE_DOMAIN` | — | Base domain for auto-generated deployment domains |

## Running E2E Tests

```bash
# From web/ directory (never from repo root)
cd /Users/artpar/workspace/code/hoster/web

export TEST_DO_API_KEY='dop_v1_your_key_here'

# Run all tests (~10-15 min, provisions real DO droplets)
npx playwright test

# Run a single journey
npx playwright test e2e/uj1-discovery.spec.ts

# View test report
npx playwright show-report
```

Tests require Hoster + APIGate running and configured. See `specs/local-e2e-setup.md` for full details.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Stale frontend at :8082 | Rebuild: `cd web && npm run build`, copy dist, rebuild Hoster binary, restart |
| Auth not working on API calls | Ensure `hoster-api` route exists with `auth_required=1` |
| "Monthly request quota exceeded" | `hoster-api` metering expression is `1` — must be `0` |
| Billing 401 | Set `HOSTER_BILLING_API_KEY` in `start.sh` |
| Billing 404 on meter endpoint | Set `routes.meter_base_path` to `/_internal/meter` in APIGate settings |
| Port already in use | `lsof -i :PORT -t \| xargs kill` |
