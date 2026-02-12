# APIGate Integration Architecture

## Overview

```
Internet → APIGate (:8082, front-facing) → Hoster (:8080, backend)
```

- **APIGate** is the front-facing server. ALL traffic goes through it.
- **Hoster** is the backend. Never directly accessed by browsers.
- APIGate forwards requests to Hoster as its upstream.

## Separation of Concerns

| Component | Responsibility | NOT Responsible For |
|-----------|---------------|---------------------|
| **APIGate** | Auth (JWT issuance + validation), billing, quota, rate-limiting, routing | Business logic, data ownership |
| **Hoster** | Business logic, data, ownership enforcement | Auth token issuance/validation, billing/quota |

## Auth: JWT via APIGate

APIGate owns authentication. Hoster reads identity from APIGate-injected headers.

```
1. User registers/logs in via APIGate auth endpoints (/mod/auth/*)
2. APIGate issues JWT token
3. Frontend stores token in localStorage
4. Every API request: Authorization: Bearer <token>
5. Request goes through APIGate → validates JWT → injects X-User-ID, X-Plan-ID headers
6. Hoster reads X-User-ID header → resolves user → enforces ownership
```

- Auth endpoints: `/mod/auth/register`, `/mod/auth/login`, `/mod/auth/me`, `/mod/auth/logout`
- APIGate validates JWT and injects `X-User-ID`, `X-Plan-ID`, `X-Plan-Limits`, `X-Key-ID` headers
- Hoster has NO auth endpoints — reads identity from injected headers only
- No cookies for API auth
- `auth_required=1`: APIGate validates JWT, injects headers, runs billing pipeline
- `auth_required=0`: APIGate **strips** Authorization header before forwarding — NOT suitable for authenticated routes

## Billing: Only Deployment CRUD

Not all APIs need billing. Only deployment lifecycle actions are billable.

**Billable (auth_required=1, billing route with metering):**
- Deployment create, start, stop, delete

**Authenticated (auth_required=1, no metering):**
- Monitoring: events, logs, stats, health
- Templates: list, get
- Nodes, SSH keys
- Billing events query

**Public (auth_required=0):**
- SPA frontend shell

## Route Configuration

Routes are evaluated by priority (highest first).

| Route | Path | auth_required | Priority | Purpose |
|-------|------|---------------|----------|---------|
| `hoster-billing` | `/api/v1/deployments*` | 1 (billing) | 55 | Deployment CRUD - billable |
| `hoster-api` | `/api/*` | 1 (auth) | 50 | All other APIs - authenticated |
| `hoster-front` | `/*` | 0 (public) | 10 | SPA frontend |

### Route Behavior

- **auth_required=1**: APIGate validates JWT, injects `X-User-ID`/`X-Plan-ID`/`X-Plan-Limits` headers, then forwards to Hoster. On billing routes, also runs quota check, rate-limit, and metering.
- **auth_required=0**: APIGate **strips** the Authorization header and forwards without auth context. Only suitable for public routes like the SPA shell.

### Route Pattern Rules

APIGate compiles route patterns to regex:
- `/*` → `^/.*$` (prefix match - CORRECT for catch-all)
- `/api/*` → `^/api/.*$` (prefix match - CORRECT)
- `/api/` → `^/api/$` (exact match only - WRONG for prefix routes!)

**Always use `/*` suffix for prefix routes.**

## Request Flow: Authenticated Route (Monitoring)

```
Browser
  │ GET /api/v1/deployments/{id}/monitoring/events
  │ Authorization: Bearer <jwt-token>
  ▼
APIGate (:8082)
  │ Route: hoster-api (auth_required=1, priority 50)
  │ Action: validate JWT → inject X-User-ID/X-Plan-ID headers → forward
  ▼
Hoster (:8080)
  │ Auth middleware: read X-User-ID header → resolve user
  │ Handler: check deployment.user_id == user.id (ownership)
  │ Return: events data
  ▼
Response → Browser
```

## Request Flow: Billing Route (Deployment Create)

```
Browser
  │ POST /api/v1/deployments
  │ Authorization: Bearer <jwt-token>
  ▼
APIGate (:8082)
  │ Route: hoster-billing (auth_required=1, priority 55)
  │ Auth: validate JWT → inject X-User-ID/X-Plan-ID headers
  │ Billing pipeline:
  │   1. Check quota (requests_per_month)
  │   2. Check rate limit
  │   3. Forward to upstream
  │   4. Calculate cost (metering_expr)
  │   5. Record usage event (with event_type from response)
  ▼
Hoster (:8080)
  │ Auth middleware: read X-User-ID header → resolve user
  │ Handler: create deployment for user
  │ Response includes X-Event-Type header for metering
  ▼
Response → APIGate (records billing event) → Browser
```

## APIGate Request Lifecycle (auth_required=1 only)

Full 15-step pipeline (from APIGate wiki):

1. Extract API key (X-API-Key / Bearer / query param)
2. Validate format (key prefix)
3. Lookup key / validate JWT token
4. Verify hash (bcrypt for keys) or JWT signature
5. Check user status = "active"
6. **Check quota** (monthly limit from user's plan)
7. **Check rate limit** (per-minute from plan)
8. Resolve entitlements (plan features → headers)
9. Match route (path, method, headers, priority)
10. Transform request (inject X-User-ID, X-Plan-ID, X-Plan-Limits, X-Key-ID headers)
11. Rewrite path (if path_rewrite expression set)
12. Forward to upstream
13. Transform response
14. Calculate cost (metering_expr)
15. Record usage event (async, includes event_type from response header)

Steps 6-7 are what blocked monitoring previously when all routes used auth_required=1. Now monitoring routes still use auth_required=1 (for auth) but are on a non-billing route without metering.

## Why Monitoring Was Broken (Historical)

All API endpoints used a single route with auth_required=1. APIGate applied billing pipeline (quota + rate-limit) to everything, including monitoring.

Frontend monitoring polling: ~50 requests/min. Default plan: 1000 requests/month, burst_tokens=5. Result: quota exhausted instantly, monitoring tabs show nothing.

**Fix:** Separate billing route (deployment CRUD, with metering) from regular authenticated route (everything else, no metering). Monitoring is authenticated but not metered.

## Hoster Endpoints

| Endpoint | Auth | Billing |
|----------|------|---------|
| `/*` | No | No | Embedded SPA |
| `/health` | No | No | Health check |
| `/api/v1/templates` | Yes (X-User-ID) | No | Template CRUD |
| `/api/v1/deployments` | Yes (X-User-ID) | **Yes** | Deployment CRUD |
| `/api/v1/deployments/{id}/start` | Yes (X-User-ID) | **Yes** | Start deployment |
| `/api/v1/deployments/{id}/stop` | Yes (X-User-ID) | **Yes** | Stop deployment |
| `/api/v1/deployments/{id}/monitoring/*` | Yes (X-User-ID) | No | Events, logs, stats, health |
| `/api/v1/nodes` | Yes (X-User-ID) | No | Node CRUD |
| `/api/v1/ssh_keys` | Yes (X-User-ID) | No | SSH key CRUD |

**Note:** Auth is provided by APIGate injecting `X-User-ID` header on `auth_required=1` routes. Hoster has no auth endpoints — login/signup/logout are handled by APIGate at `/mod/auth/*`.
