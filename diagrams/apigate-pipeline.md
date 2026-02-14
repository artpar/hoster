# APIGate Billing Pipeline — Detail

Source: `specs/architecture/apigate-integration.md`, [APIGate Wiki](https://github.com/artpar/apigate/wiki)

This pipeline runs **only** for routes with `auth_required=1` (currently just `/api/v1/deployments*`).
Pass-through routes (`auth_required=0`) skip this entirely.

## Full 15-Step Pipeline

```mermaid
flowchart TD
    Request["Incoming Request<br/>POST /api/v1/deployments<br/>Authorization: Bearer {jwt}"]

    subgraph Auth["Authentication (Steps 1-5)"]
        S1["① Extract API key<br/>X-API-Key / Bearer token / query param"]
        S2["② Validate format<br/>Key prefix"]
        S3["③ Lookup key / validate token<br/>Database or JWT verification"]
        S4["④ Verify hash<br/>bcrypt for API keys"]
        S5["⑤ Check user status = active"]
        S1 --> S2 --> S3 --> S4 --> S5
    end

    subgraph Billing["Billing & Quota (Steps 6-8)"]
        S6["⑥ Check quota<br/>Monthly limit from user's plan<br/>(e.g. 1000 requests/month)"]
        S7["⑦ Check rate limit<br/>Per-minute burst from plan<br/>(e.g. burst_tokens=5)"]
        S8["⑧ Resolve entitlements<br/>Plan features → response headers"]
        S6 --> S7 --> S8
    end

    subgraph Routing["Route & Transform (Steps 9-11)"]
        S9["⑨ Match route<br/>Path + method + priority<br/>/api/v1/deployments* → priority 55"]
        S10["⑩ Transform request headers<br/>Inject: X-User-ID, X-Plan-ID,<br/>X-Plan-Limits, X-Key-ID,<br/>X-Organization-ID<br/>Strip: Authorization"]
        S11["⑪ Rewrite path<br/>(if path_rewrite expression set)"]
        S9 --> S10 --> S11
    end

    subgraph Forward["Forward (Step 12)"]
        S12["⑫ Forward to Hoster upstream<br/>http://localhost:8080"]
    end

    subgraph PostProcess["Post-Processing (Steps 13-15)"]
        S13["⑬ Transform response<br/>(if response transform configured)"]
        S14["⑭ Calculate cost<br/>metering_expr evaluation"]
        S15["⑮ Record usage event<br/>(async — does not block response)"]
        S13 --> S14 --> S15
    end

    Request --> Auth
    Auth -->|"✓ Authenticated"| Billing
    Auth -->|"✗ Invalid"| Reject401["401 Unauthorized"]
    Billing -->|"✓ Within limits"| Routing
    Billing -->|"✗ Quota exceeded"| Reject429["429 Too Many Requests"]
    Billing -->|"✗ Rate limited"| Reject429
    Routing --> Forward
    Forward --> PostProcess
    PostProcess --> Response["Response → Browser"]
```

## Header Injection Detail (Step 10)

```mermaid
flowchart LR
    subgraph Before["Request Headers (from browser)"]
        H1["Authorization: Bearer {jwt}"]
        H2["Content-Type: application/vnd.api+json"]
    end

    Transform["APIGate<br/>Step ⑩<br/>Transform"]

    subgraph After["Request Headers (to Hoster)"]
        H4["X-User-ID: user_bc6849d9"]
        H5["X-Plan-ID: plan_free"]
        H6["X-Plan-Limits: {&quot;max_deployments&quot;:1,...}"]
        H7["X-Key-ID: key_abc123"]
        H8["X-Organization-ID: org_xyz"]
        H9["Content-Type: application/vnd.api+json"]
    end

    Before --> Transform --> After
```

APIGate **strips** the Authorization header and **injects** identity headers that Hoster trusts.

## What Pass-Through Routes Skip

| Step | Description | Billing Route | Pass-Through |
|------|-------------|:---:|:---:|
| 1-5 | Authentication | Yes | **Skipped** |
| 6 | Quota check | Yes | **Skipped** |
| 7 | Rate limit | Yes | **Skipped** |
| 8 | Entitlements | Yes | **Skipped** |
| 9 | Route match | Yes | Yes |
| 10 | Header transform | Yes | **No** (headers forwarded as-is) |
| 11 | Path rewrite | Yes | **No** |
| 12 | Forward to upstream | Yes | Yes |
| 13-15 | Post-processing | Yes | **Skipped** |

## Why This Matters

Monitoring endpoints (`/api/v1/deployments/{id}/monitoring/*`) poll at ~50 requests/min. If they went through the billing pipeline, a free-tier user (1000 req/month, burst=5) would exhaust quota in 20 minutes. By using pass-through routes for non-billing APIs, monitoring works without hitting limits.

**Current trade-off:** The APIGate route pattern `/api/v1/deployments*` matches monitoring sub-routes too, so they currently go through the billing lane. A more granular APIGate route configuration could separate these, but the current setup works because APIGate doesn't reject pass-through requests that happen to match a billing route pattern — the billing behavior is applied per the `auth_required` flag on the matched route.
