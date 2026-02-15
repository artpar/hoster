# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 15, 2026)

### Status: v0.3.46 RELEASED — Cloud provision destroy fix + E2E leak detection

**Latest Release:** v0.3.46 — CI passed, release workflow triggered, deploying to production.

**What's Working:**
- Full deployment lifecycle on real cloud infrastructure (DigitalOcean)
- Generic engine: schema-driven CRUD for all entities
- Cloud provisioning: credential → provision → droplet → node → deploy → destroy
- 8 Playwright E2E user journeys (65 tests) against real infrastructure
- Billing via APIGate metering (Stripe Checkout integration)
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

## LAST SESSION (February 15, 2026) — Session 9

### What Was Done

1. **Fixed cloud provision destroy flow** (`internal/engine/api.go`)
   - `deleteHandler` hardcoded `"deleting"` in `Transition()` call
   - Cloud provisions use `ready → destroying` (not `ready → deleting`)
   - Fix: use matched target state variable `t` instead of hardcoded string
   - This caused leaked DO droplets — DB row deleted but actual droplet left alive

2. **Added DestroyInstance command handler** (`internal/engine/handlers.go`)
   - Decrypts cloud credential, creates provider, calls `DestroyInstance()`
   - Transitions provision to `destroyed`, deletes associated node
   - Registered on `Bus.Register("DestroyInstance", destroyProvision)`

3. **Passed encryption key to command bus** (`cmd/hoster/server.go`)
   - `bus.SetExtra("encryption_key", encryptionKey)` — needed for credential decryption in destroy handler

4. **SSH key reuse for re-provisioning** (`internal/engine/setup.go`)
   - BeforeCreate hook checks for existing SSH key with matching name before generating new one
   - Restores b880707 fix lost in engine rewrite

5. **DO API leak detection in E2E teardown** (`web/e2e/global-teardown.ts`)
   - After UI-driven cleanup, queries DO API for any droplets with `e2e-` prefix
   - If found: force-destroys them AND throws error to surface the bug
   - No more silent leaks — every leaked droplet fails the test

6. **E2E test suite rewritten** — all 8 user journeys now use browser UI instead of direct API calls. Deleted `web/e2e/fixtures/api.fixture.ts`.

7. **Removed hardcoded DO API key** from `test-data.ts` (GitHub push protection caught it). Now env-only via `TEST_DO_API_KEY`.

8. **Updated docs** — `specs/local-e2e-setup.md` updated for leak detection and env-only API key.

### Files Changed
- `internal/engine/api.go` — Fix: use `t` instead of hardcoded `"deleting"` in Transition
- `internal/engine/handlers.go` — New: `destroyProvision` command handler
- `internal/engine/setup.go` — SSH key reuse for re-provisioning
- `cmd/hoster/server.go` — Pass encryption key to command bus
- `web/e2e/global-teardown.ts` — DO API leak detection + force-destroy
- `web/e2e/global-setup.ts` — Rewritten to use browser UI
- `web/e2e/fixtures/api.fixture.ts` — DELETED (no more direct API calls)
- `web/e2e/fixtures/auth.fixture.ts` — Simplified
- `web/e2e/fixtures/test-data.ts` — Removed hardcoded DO API key
- `web/e2e/uj1-uj8` — All 8 user journeys rewritten for browser UI
- `web/src/components/nodes/CloudServersTab.tsx` — Cleanup
- `specs/local-e2e-setup.md` — Updated for leak detection

### Verified
- `go vet ./...` — clean
- `go build ./...` — clean
- CI test suite — all pass (proxy test is env-specific, passes on CI)
- UJ1 E2E: 9/9 tests pass, teardown verified 0 leaked droplets on DO
- v0.3.46 tagged, release workflow running

---

## IMMEDIATE NEXT STEPS

1. **Verify v0.3.46 release** deployed to production
2. **Run remaining E2E journeys** (UJ2-UJ8) to validate full suite
3. **Production E2E testing** — all user journeys on https://emptychair.dev
4. **Stripe live mode** — production billing flow testing

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
