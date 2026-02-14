# Request Flow — Full Lifecycle

Source: `internal/shell/api/setup.go`, `specs/architecture/apigate-integration.md`

## Three Request Lanes

```mermaid
flowchart TD
    Browser["Browser / Client"]

    Browser -->|"Request"| APIGate

    subgraph APIGate["APIGate (:8082)"]
        direction TB
        RouteMatch{"Route match<br/>(highest priority)"}
        RouteMatch -->|"/api/v1/deployments*<br/>priority 55"| BillingLane
        RouteMatch -->|"/api/*<br/>priority 50"| PassThruLane
        RouteMatch -->|"/*<br/>priority 10"| PublicLane

        subgraph BillingLane["Billing Route (auth_required=1)"]
            B1["Extract API key / JWT"]
            B2["Validate key / token"]
            B3["Check quota (monthly)"]
            B4["Check rate limit (per-min)"]
            B5["Inject headers:<br/>X-User-ID, X-Plan-ID,<br/>X-Plan-Limits, X-Key-ID"]
            B6["Forward to Hoster"]
            B1 --> B2 --> B3 --> B4 --> B5 --> B6
        end

        subgraph PassThruLane["Pass-Through Route (auth_required=0)"]
            P1["Forward as-is to Hoster<br/>(Authorization header passes through)"]
        end

        subgraph PublicLane["Public Route (auth_required=0)"]
            PU1["Forward to Hoster<br/>(serves SPA)"]
        end
    end

    subgraph Hoster["Hoster (:8080)"]
        direction TB
        MW1["requestIDMiddleware<br/>Generate/forward X-Request-ID"]
        MW2["recoveryMiddleware<br/>Catch panics → 500"]
        MW3["authMW.Handler<br/>Extract X-User-ID → ResolveUser()"]

        MW1 --> MW2 --> MW3

        MW3 --> RouteHandler{"Route handler"}

        RouteHandler -->|"/health, /ready"| HealthH["Health handler<br/>(no auth needed)"]
        RouteHandler -->|"/api/v1/*"| APIH["JSON:API resource / custom action"]
        RouteHandler -->|"/*"| SPAH["WebUIHandler<br/>(serve embedded SPA)"]

        APIH --> OwnerCheck{"Ownership check<br/>auth.CanViewDeployment()"}
        OwnerCheck -->|"Pass"| Response["200 Response"]
        OwnerCheck -->|"Fail"| Deny["404 Not Found"]
    end

    BillingLane --> MW1
    PassThruLane --> MW1
    PublicLane --> MW1

    subgraph PostForward["APIGate Post-Forward (billing only)"]
        PF1["Calculate cost (metering_expr)"]
        PF2["Record usage event (async)"]
        PF1 --> PF2
    end

    Response --> PostForward
    PostForward --> Browser
```

## Lane Summary

| Lane | APIGate Route | auth_required | What APIGate Does | What Hoster Does |
|------|--------------|---------------|-------------------|------------------|
| **Billing** | `/api/v1/deployments*` (priority 55) | 1 | JWT validate, quota, rate-limit, meter, inject headers | Auth middleware resolves user, handler checks ownership |
| **Pass-through** | `/api/*` (priority 50) | 0 | Forward as-is | Auth middleware resolves user from headers, handler checks ownership |
| **Public** | `/*` (priority 10) | 0 | Forward as-is | Serve embedded SPA (no auth needed) |

## Key Points

- APIGate evaluates routes by **priority** (highest first) — deployment routes (55) match before general API (50)
- Hoster **always** runs auth middleware — even for public routes (it just sets `Authenticated: false`)
- Ownership checks happen **in the handler**, not the middleware — middleware only extracts identity
- Health endpoints (`/health`, `/ready`) bypass auth entirely — they respond before the auth middleware matters
