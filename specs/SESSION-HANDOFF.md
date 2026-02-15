# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 15, 2026)

### Status: v0.3.50 RELEASED — Billing meter path, cloud destroy error handling, docs update

**Latest Release:** v0.3.50 — Billing client now uses configurable `/_internal/meter` path (avoids `/api/*` route shadowing APIGate's metering endpoint). Cloud provision destroy properly fails to "failed" state on errors instead of silently marking "destroyed". Delete handler returns 409 when destroy results in failure. Docs updated for APIGate v0.3.8 and billing configuration.

**What's Working:**
- Full deployment lifecycle on real cloud infrastructure (DigitalOcean)
- Generic engine: schema-driven CRUD for all entities
- Cloud provisioning: credential → provision → droplet → node → deploy → destroy
- **Deployment access info**: domain URL + direct IP:port shown on detail page
- **Domains tab**: lists auto + custom domains with DNS instructions
- **Billing**: usage metering via APIGate (`/_internal/meter`), Stripe Checkout integration
- 12 marketplace templates (6 infra + 6 web-UI apps)
- Production at https://emptychair.dev

### Architecture

```
Internet → APIGate (:8082, JWT auth + billing) → Hoster (:8080, business logic)
```

- **Engine**: `internal/engine/` — schema-driven generic CRUD (~4,000 lines replaced ~13,700)
- **Frontend**: `web/` — React + Vite + TanStack Query + Zustand + TailwindCSS
- **Infrastructure**: `internal/shell/` — Docker SSH, cloud providers, proxy, billing

### Local E2E Setup

See **`specs/local-e2e-setup.md`** for full details. Quick summary:

```bash
# Services
/tmp/hoster-e2e-test/start.sh &          # Hoster on :8080
/tmp/hoster-e2e-test/apigate-darwin-arm64 serve  # APIGate on :8082

# Run tests (MUST set TEST_DO_API_KEY env var)
cd web && npx playwright test e2e/uj1-discovery.spec.ts
```

---

## LAST SESSION (February 15, 2026) — Session 13

### What Was Done

1. **Billing meter path fix** (`internal/shell/billing/client.go`)
   - Added `MeterPath` config field with default `/_internal/meter`
   - Old hardcoded `/api/v1/meter` was shadowed by `hoster-api` route (`/api/*`)
   - APIGate v0.3.8 added configurable `routes.meter_base_path` setting
   - Created API key in APIGate admin for `HOSTER_BILLING_API_KEY`
   - Billing now working: usage events reported successfully

2. **Cloud provision destroy error handling** (`internal/engine/handlers.go`)
   - `destroyProvision` now uses `failProvision()` on errors (missing credential, decrypt failure, provider failure, API failure)
   - Previously silently transitioned to "destroyed" even when cloud API call failed → orphaned resources
   - Added `failProvision()` helper matching `failDeployment()` pattern

3. **Delete handler failure detection** (`internal/engine/api.go`)
   - After dispatching destroy command, checks resulting state
   - If handler transitioned to "failed", returns 409 instead of deleting the DB record
   - Prevents losing track of cloud resources that failed to destroy

4. **Cloud provisions state machine** (`internal/engine/resources.go`)
   - Allow `destroying` from `pending`, `creating`, `configuring` states (not just `ready`/`failed`)

5. **E2E teardown leak detection** (`web/e2e/global-teardown.ts`)
   - Detect leaked droplets matching all test prefixes (`e2e-`, `uj4node-`)

6. **Docs updated**
   - `specs/local-e2e-setup.md` — APIGate v0.3.8, billing API key step, meter path config step
   - `docs/local-e2e-development.md` — complete rewrite (removed stale HOSTER_AUTH_MODE, curl examples, Docker Compose)

### Files Changed
- `internal/shell/billing/client.go` — `MeterPath` config, default `/_internal/meter`
- `internal/shell/billing/client_test.go` — Updated test to expect `/_internal/meter`
- `internal/engine/api.go` — Delete handler checks for "failed" state after dispatch
- `internal/engine/handlers.go` — `destroyProvision` error handling, `failProvision()` helper
- `internal/engine/resources.go` — Cloud provisions state machine: destroying from more states
- `web/e2e/global-teardown.ts` — Multi-prefix leaked droplet detection
- `specs/local-e2e-setup.md` — APIGate v0.3.8, billing steps 7-8, troubleshooting
- `docs/local-e2e-development.md` — Complete rewrite

### Verified
- `go vet ./...` — clean
- `go build ./...` — clean
- CI test suite — all pass (proxy test is env-specific, passes on CI)
- v0.3.50 tagged, release workflow triggered

---

## IMMEDIATE NEXT STEPS

1. **Verify v0.3.50 release** deployed to production
2. **Re-add `hoster-api` route** in local E2E env — it was deleted during debugging but is needed for auth enforcement on API paths
3. **Run remaining E2E journeys** (UJ2-UJ8) to validate full suite
4. **Production E2E testing** — all user journeys on https://emptychair.dev
5. **Stripe live mode** — production billing flow testing

---

## Quick Reference

| Resource | Location |
|----------|----------|
| Repo | https://github.com/artpar/hoster |
| Production | https://emptychair.dev |
| APIGate repo | https://github.com/artpar/apigate |
| E2E setup guide | `specs/local-e2e-setup.md` |
| Architecture spec | `specs/architecture/apigate-integration.md` |
| Status tracker | `specs/STATUS.md` |
| User journeys | `specs/user-journeys.md` |

| Service | Port | Purpose |
|---------|------|---------|
| APIGate | 8082 | Front-facing: JWT auth + billing + routing |
| Hoster | 8080 | Backend: API + embedded SPA |
| Vite | 3000 | Dev hot-reload only (NOT for testing) |
| App Proxy | 9091 | Deployment routing (`*.apps.localhost`) |
