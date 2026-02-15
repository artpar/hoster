# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 15, 2026)

### Status: v0.3.47 RELEASED — Deployment access UX + Domains tab fix

**Latest Release:** v0.3.47 — Deployment detail page now shows access URLs, Domains tab data loading fixed.

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

## LAST SESSION (February 15, 2026) — Session 10

### What Was Done

1. **Fixed deployment access URL display** (`web/src/api/types.ts`, `DeploymentDetailPage.tsx`, `DeploymentCard.tsx`)
   - `DeploymentAttributes.domain` (singular string) was always undefined — API returns `domains` (JSON array)
   - Added `DeploymentDomain` type and `getPrimaryDomain()` helper
   - Deployment card and detail page now correctly parse `domains[]` array

2. **Fixed Domains tab data loading** (`web/src/api/domains.ts`)
   - Backend domains endpoints return plain JSON arrays, not JSON:API wrapped `{data: [...]}`
   - `domainsApi` used `apiClient` which expected JSON:API format, then accessed `.data` → `undefined`
   - Replaced with `domainFetch()` that handles raw JSON responses correctly

3. **Added "Access Your Application" card** to deployment Overview tab
   - Shows primary domain URL as clickable link with "Open" button
   - Shows direct IP:port access (node SSH host + proxy port) as fallback
   - Fetches node info via `useNode()` hook

4. **Added "Open App" button** in deployment header action bar (shown when running + has domain)

5. **Shows node/port in Deployment Info** section on Overview tab

6. **Fixed domain generation inconsistency** (`internal/engine/setup.go`)
   - `domainListHandler` used `refID + ".apps." + baseDomain` — wrong format, caused double `.apps.` nesting
   - `domainAddHandler` CNAME target used same wrong format
   - `domainVerifyHandler` expected target used same wrong format
   - All three now use `domain.Slugify(name) + "." + baseDomain` matching `scheduleDeployment`
   - Auto domain only generated on-the-fly if none stored (legacy fallback)

7. **Fixed production env var names** (`deploy/local/emptychair.env`, gitignored)
   - `HOSTER_APP_PROXY_*` → `HOSTER_PROXY_*` (Viper key path is `proxy.*`)
   - Added `HOSTER_DOMAIN_BASE_DOMAIN=apps.emptychair.dev` (was missing)

### Files Changed
- `web/src/api/types.ts` — `DeploymentDomain` type, `getPrimaryDomain()`, updated `DeploymentAttributes`
- `web/src/api/domains.ts` — `domainFetch()` replaces broken `apiClient` usage
- `web/src/components/deployments/DeploymentCard.tsx` — Uses `getPrimaryDomain(domains)`
- `web/src/pages/deployments/DeploymentDetailPage.tsx` — Access card, Open button, node info
- `internal/engine/setup.go` — Fix domain list/add/verify hostname generation
- `deploy/local/emptychair.env` — Fix env var names (gitignored)

### Verified
- `go vet ./...` — clean
- `go build ./...` — clean
- CI test suite — all pass (proxy test is env-specific, passes on CI)
- v0.3.47 tagged, release workflow running

---

## IMMEDIATE NEXT STEPS

1. **Verify v0.3.47 release** deployed to production
2. **Update production env** — fix `HOSTER_DOMAIN_BASE_DOMAIN` and `HOSTER_PROXY_*` vars on server
3. **Test on prod** — verify Matomo deployment shows access URL after env fix + restart
4. **Run remaining E2E journeys** (UJ2-UJ8) to validate full suite
5. **Production E2E testing** — all user journeys on https://emptychair.dev
6. **Stripe live mode** — production billing flow testing

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
