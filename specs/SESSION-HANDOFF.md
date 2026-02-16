# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 15, 2026)

### Status: v0.3.50 RELEASED — All 65 E2E tests passing across 8 user journeys

**Latest Release:** v0.3.50 — Billing meter path fix, cloud destroy error handling, docs update. All E2E journeys validated and passing.

**What's Working:**
- Full deployment lifecycle on real cloud infrastructure (DigitalOcean)
- Generic engine: schema-driven CRUD for all entities
- Cloud provisioning: credential → provision → droplet → node → deploy → destroy
- **Deployment access info**: domain URL + direct IP:port shown on detail page
- **Domains tab**: lists auto + custom domains with DNS instructions
- **Billing**: usage metering via APIGate (`/_internal/meter`), Stripe Checkout integration
- 12 marketplace templates (6 infra + 6 web-UI apps)
- **All 65 E2E tests passing** across 8 user journeys (UJ1-UJ8)
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

## LAST SESSION (February 15, 2026) — Session 14

### What Was Done

1. **All 65 E2E tests passing across 8 user journeys**

   | Journey | Tests | Result |
   |---------|-------|--------|
   | UJ1: Discovery & Browsing | 9/9 | Passed |
   | UJ2: First Deployment | 10/10 | Passed |
   | UJ3: Monitoring & Management | 10/10 | Passed |
   | UJ4: Infrastructure Scaling | 8/8 | Passed (2 flaky on DO timing, pass on retry) |
   | UJ5: Creator Monetization | 8/8 | Passed |
   | UJ6: SSH Key Management | 7/7 | Passed |
   | UJ7: Billing & Plans | 6/6 | Passed |
   | UJ8: Full Lifecycle | 7/7 | Passed |

2. **UJ4 credential selection fix** (`web/e2e/uj4-infrastructure-scaling.spec.ts`)
   - Playwright `selectOption({ label: RegExp })` doesn't work — only accepts strings
   - Changed to `selectOption({ label: \`${credName} (digitalocean)\` })` matching `ProvisionNodeForm.tsx` label format

3. **UJ5 dialog overlay fix** (`web/e2e/uj5-creator-monetization.spec.ts`)
   - Delete confirm button click intercepted by shadcn AlertDialog overlay
   - `.last()` locator resolved to button behind the `fixed inset-0 bg-black/80` overlay
   - Fixed by scoping confirm button to dialog container: `page.locator('[role="alertdialog"], [role="dialog"], .fixed.inset-0.z-50').getByRole('button', { name: /Delete|Confirm/i })`
   - Applied same fix to afterAll cleanup

4. **Previous session fixes also validated**
   - UJ3 deployment detail click: `goto(href)` instead of `click()` (nested `<a>` tags issue)
   - UJ3 logs tab reuse: each test navigates fresh instead of reusing tabs
   - UJ8 deployment link selectors: UUID pattern instead of `depl_` prefix
   - Deployment restart race condition: `maps.Clone(row)` before goroutine in `setup.go`

### Files Changed
- `web/e2e/uj4-infrastructure-scaling.spec.ts` — Credential selection: RegExp → string label
- `web/e2e/uj5-creator-monetization.spec.ts` — Dialog-scoped Delete confirm button (test + afterAll)

### Known Flaky Areas
- **UJ4 "provision real DO droplet"** — Real DigitalOcean provisioning takes 1-6 min. First attempt may timeout if droplet is slow. Passes on Playwright retry. Not a code bug.

---

## SESSION 13 (February 15, 2026)

### What Was Done

1. **Billing meter path fix** (`internal/shell/billing/client.go`)
   - Added `MeterPath` config field with default `/_internal/meter`
   - Old hardcoded `/api/v1/meter` was shadowed by `hoster-api` route (`/api/*`)

2. **Cloud provision destroy error handling** (`internal/engine/handlers.go`)
   - `destroyProvision` now uses `failProvision()` on errors
   - Previously silently transitioned to "destroyed" even when cloud API call failed

3. **Delete handler failure detection** (`internal/engine/api.go`)
   - If handler transitioned to "failed", returns 409 instead of deleting DB record

4. **v0.3.50 released** — billing meter path, cloud destroy error handling, docs update

---

## IMMEDIATE NEXT STEPS

1. **Production E2E testing** — all user journeys on https://emptychair.dev
2. **Stripe live mode** — production billing flow testing
3. **Re-add `hoster-api` route** in local E2E env — deleted during debugging, needed for auth enforcement on API paths

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
