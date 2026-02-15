# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 15, 2026)

### Status: v0.3.48 RELEASED — Fix container port binding on remote nodes

**Latest Release:** v0.3.48 — Container ports bound to 0.0.0.0 instead of 127.0.0.1 so deployed apps are accessible via public IP on cloud nodes.

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

## LAST SESSION (February 15, 2026) — Session 11

### What Was Done

1. **Fixed container port binding on remote nodes** (`internal/shell/docker/orchestrator.go`)
   - Container port was bound to `127.0.0.1` (localhost only) — deployed apps unreachable via public IP
   - Changed to `0.0.0.0` so ports listen on all interfaces
   - Existing deployments on prod need restart (stop + start) for new binding to take effect

### Files Changed
- `internal/shell/docker/orchestrator.go` — `hostIP = "0.0.0.0"` (was `"127.0.0.1"`)

### Verified
- `go vet ./...` — clean
- `go build ./...` — clean
- CI test suite — all pass (proxy test is env-specific, passes on CI)
- v0.3.48 tagged, release workflow triggered

---

## IMMEDIATE NEXT STEPS

1. **Verify v0.3.48 release** deployed to production
2. **Restart Matomo deployment** on prod (stop + start) so new port binding takes effect
3. **Test access** — verify `http://{node_ip}:{proxy_port}` now loads for Matomo
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
