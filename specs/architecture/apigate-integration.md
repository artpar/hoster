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
| **APIGate** | Billing, quota, rate-limiting, routing | User authentication |
| **Hoster** | Auth, business logic, data, ownership | Billing/quota |

## Auth: Token-Based, Single Path

Hoster owns authentication. APIGate does not.

```
1. User registers/logs in with Hoster → gets auth token
2. Frontend stores token in localStorage
3. Every API request: Authorization: Bearer <token>
4. Request goes through APIGate → forwarded to Hoster
5. Hoster validates token → extracts user → enforces ownership
```

- Single auth mechanism: Bearer token validated by Hoster
- Same auth path for all routes (billing and pass-through)
- No cookies for API auth
- No manual X-User-ID/X-Plan-ID header construction

## Billing: Only Deployment CRUD

Not all APIs need billing. Only deployment lifecycle actions are billable.

**Billable (auth_required=1 in APIGate):**
- Deployment create, start, stop, delete

**Pass-through (auth_required=0 in APIGate):**
- Monitoring: events, logs, stats, health
- Templates: list, get
- Nodes, SSH keys
- Auth endpoints

## Route Configuration

Routes are evaluated by priority (highest first).

| Route | Path | auth_required | Priority | Purpose |
|-------|------|---------------|----------|---------|
| `hoster-billing` | `/api/v1/deployments*` | 1 (billing) | 55 | Deployment CRUD - billable |
| `hoster-api` | `/api/*` | 0 (pass-thru) | 50 | All other APIs |
| `hoster-front` | `/*` | 0 (public) | 10 | SPA frontend |

### Route Behavior

- **auth_required=0**: APIGate forwards request as-is to Hoster. Authorization header passes through. Hoster authenticates.
- **auth_required=1**: APIGate runs billing pipeline (quota check, rate-limit, metering), then forwards to Hoster. Hoster still authenticates.

### Route Pattern Rules

APIGate compiles route patterns to regex:
- `/*` → `^/.*$` (prefix match - CORRECT for catch-all)
- `/api/*` → `^/api/.*$` (prefix match - CORRECT)
- `/api/` → `^/api/$` (exact match only - WRONG for prefix routes!)

**Always use `/*` suffix for prefix routes.**

## Request Flow: Pass-Through Route (Monitoring)

```
Browser
  │ GET /api/v1/deployments/{id}/monitoring/events
  │ Authorization: Bearer <hoster-token>
  ▼
APIGate (:8082)
  │ Route: hoster-api (auth_required=0, priority 50)
  │ Action: forward as-is (no billing, no quota check)
  ▼
Hoster (:8080)
  │ Auth middleware: validate Bearer token → extract user
  │ Handler: check deployment.user_id == user.id (ownership)
  │ Return: events data
  ▼
Response → Browser
```

## Request Flow: Billing Route (Deployment Create)

```
Browser
  │ POST /api/v1/deployments
  │ Authorization: Bearer <hoster-token>
  ▼
APIGate (:8082)
  │ Route: hoster-billing (auth_required=1, priority 55)
  │ Billing pipeline:
  │   1. Identify user (from token/key)
  │   2. Check quota (requests_per_month)
  │   3. Check rate limit
  │   4. Forward to upstream
  │   5. Calculate cost (metering_expr)
  │   6. Record usage event
  ▼
Hoster (:8080)
  │ Auth middleware: validate Bearer token → extract user
  │ Handler: create deployment for user
  ▼
Response → APIGate → Browser
```

## APIGate Request Lifecycle (auth_required=1 only)

Full 15-step pipeline (from APIGate wiki):

1. Extract API key (X-API-Key / Bearer / query param)
2. Validate format (key prefix)
3. Lookup key / validate token
4. Verify hash (bcrypt for keys)
5. Check user status = "active"
6. **Check quota** (monthly limit from user's plan)
7. **Check rate limit** (per-minute from plan)
8. Resolve entitlements (plan features → headers)
9. Match route (path, method, headers, priority)
10. Transform request (add/modify headers per route config)
11. Rewrite path (if path_rewrite expression set)
12. Forward to upstream
13. Transform response
14. Calculate cost (metering_expr)
15. Record usage event (async)

Steps 6-7 are what blocked monitoring previously. Pass-through routes skip all 15 steps.

## Why Monitoring Was Broken

All API endpoints used a single route with auth_required=1. APIGate applied billing pipeline (quota + rate-limit) to everything, including monitoring.

Frontend monitoring polling: ~50 requests/min. Default plan: 1000 requests/month, burst_tokens=5. Result: quota exhausted instantly, monitoring tabs show nothing.

**Fix:** Separate billing routes (deployment CRUD only) from pass-through routes (everything else). Monitoring bypasses billing entirely.

## Hoster Endpoints

| Endpoint | Auth | Billing |
|----------|------|---------|
| `/* ` | No | No | Embedded SPA |
| `/health` | No | No | Health check |
| `/auth/*` | No | No | Register/Login (issues tokens) |
| `/api/v1/templates` | Yes | No | Template CRUD |
| `/api/v1/deployments` | Yes | **Yes** | Deployment CRUD |
| `/api/v1/deployments/{id}/start` | Yes | **Yes** | Start deployment |
| `/api/v1/deployments/{id}/stop` | Yes | **Yes** | Stop deployment |
| `/api/v1/deployments/{id}/monitoring/*` | Yes | No | Events, logs, stats, health |
| `/api/v1/nodes` | Yes | No | Node CRUD |
| `/api/v1/ssh_keys` | Yes | No | SSH key CRUD |
