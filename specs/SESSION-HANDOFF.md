# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 15, 2026)

### Status: v0.3.49 RELEASED — Fix deployment restart race condition

**Latest Release:** v0.3.49 — Fixed race condition where `stripFields()` mutated the row map shared with the goroutine dispatching start/stop commands, causing `template_id` to become a string instead of an integer. Deployments restarted from stopped/failed state would fail with "template not found: templates id=0".

**What's Working:**
- Full deployment lifecycle on real cloud infrastructure (DigitalOcean)
- Generic engine: schema-driven CRUD for all entities
- Cloud provisioning: credential → provision → droplet → node → deploy → destroy
- **Deployment access info**: domain URL + direct IP:port shown on detail page
- **Domains tab**: lists auto + custom domains with DNS instructions
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

## LAST SESSION (February 15, 2026) — Session 12

### What Was Done

1. **Fixed deployment restart race condition** (`internal/engine/setup.go`)
   - `Transition()` returns `row` with integer FK values (e.g., `template_id = 3`)
   - Command dispatch was in a goroutine capturing `row` by reference
   - `stripFields()` called immediately after mutated `row` in-place, converting `template_id` from int to reference_id string
   - Goroutine's `startDeployment` handler then read `template_id` as a string → `toInt()` returned 0 → "template not found"
   - Fix: `maps.Clone(row)` before passing to goroutine in both start and stop handlers

### Files Changed
- `internal/engine/setup.go` — Added `"maps"` import, clone row before goroutine dispatch in start handler (line ~327) and stop handler (line ~378)

### Verified
- `go vet ./...` — clean
- `go build ./...` — clean
- CI test suite — all pass (proxy test is env-specific, passes on CI)
- v0.3.49 tagged, release workflow triggered

---

## IMMEDIATE NEXT STEPS

1. **Verify v0.3.49 release** deployed to production
2. **Test deployment restart** on prod — stop a running deployment, then start it; should transition to running (not failed)
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
