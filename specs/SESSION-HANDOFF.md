# Session Handoff

> For Claude starting a new session. Read CLAUDE.md first, then this file.

---

## CURRENT STATE (February 20, 2026)

### Status: v0.3.52 DEPLOYED — Remote node proxying + production TLS + nginx

**Latest Release:** v0.3.52 — App proxy routes traffic to containers on remote cloud nodes. Production fully verified: `ghost-blog.apps.emptychair.dev` serves Ghost blog from DO droplet at 24.199.126.77.

**What's Working:**
- Full deployment lifecycle on real cloud infrastructure (DigitalOcean)
- **App proxy routes to remote nodes** — resolves node IP, proxies HTTP to remote container port
- **Production app proxy verified** — `https://ghost-blog.apps.emptychair.dev/` → 200 OK
- **Wildcard TLS cert** — `emptychair.dev`, `*.emptychair.dev`, `*.apps.emptychair.dev`
- Generic engine: schema-driven CRUD for all entities
- Cloud provisioning: credential → provision → droplet → node → deploy → destroy
- **Deployment access info**: domain URL + direct IP:port shown on detail page
- **Domains tab**: lists auto + custom domains with DNS instructions
- **Billing**: usage metering via APIGate (`/_internal/meter`), Stripe Checkout integration
- 12 marketplace templates (6 infra + 6 web-UI apps)
- **All 65 E2E tests passing** across 8 user journeys (UJ1-UJ8)
- Production at https://emptychair.dev

### Production Architecture (changed February 20, 2026)

```
Internet → nginx (:80/:443, TLS termination + routing)
              ├── emptychair.dev / *.emptychair.dev → APIGate (:8082, JWT auth + billing) → Hoster (:8080)
              └── *.apps.emptychair.dev → App Proxy (:9091) → remote node (e.g., 24.199.126.77:30000)
```

**Changed from:** APIGate directly on :443 with single-domain ACME (no wildcard support)
**Changed to:** nginx on :443 with certbot wildcard cert, APIGate on :8082 (HTTP only)

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

## LAST SESSION (February 20, 2026) — Session 15

### What Was Done

1. **Remote node proxying for app proxy** (`internal/shell/proxy/server.go`, `internal/core/proxy/target.go`, `internal/engine/store.go`)
   - Added `NodeIP` field and `RemoteAddress()` method to `ProxyTarget`
   - Added `GetNodeSSHHost()` to `ProxyStore` interface + engine `Store` implementation
   - `resolveTarget()` looks up node SSH host for remote deployments
   - `getUpstreamURL()` routes to `http://{nodeIP}:{port}` instead of erroring
   - Tests: remote proxy routing test (httptest backend), node-not-found error test

2. **Production TLS + nginx setup**
   - Installed `python3-certbot-dns-route53` on server for automated DNS challenge
   - Renewed cert: `emptychair.dev` + `*.emptychair.dev` + `*.apps.emptychair.dev` (expires 2026-05-21)
   - Switched from APIGate-on-:443 (single-domain ACME) to nginx-on-:443 (wildcard cert)
   - APIGate moved to :8082 HTTP-only (TLS disabled in systemd + DB settings)
   - nginx enabled and set to start on boot
   - AWS credentials in `/root/.aws/` for certbot auto-renewal

3. **Ghost blog fix on remote node (24.199.126.77)**
   - UFW firewall: opened ports 30000-39999 for app proxy traffic
   - Docker networking: containers were on default bridge (no DNS resolution) — reconnected to deployment network with `db` alias
   - Ghost container restarted and running

4. **v0.3.52 deployed to production** — verified `https://ghost-blog.apps.emptychair.dev/` returns 200

### Files Changed
- `internal/core/proxy/target.go` — `NodeIP` field, `RemoteAddress()` method
- `internal/core/proxy/target_test.go` — `RemoteAddress` tests
- `internal/shell/proxy/server.go` — `GetNodeSSHHost` interface, remote routing in `resolveTarget` + `getUpstreamURL`
- `internal/shell/proxy/server_test.go` — Mock `GetNodeSSHHost`, remote node tests
- `internal/engine/store.go` — `GetNodeSSHHost` implementation

### Production Infra Changes (not in repo)
- `/etc/systemd/system/apigate.service` — port 8082, TLS disabled
- `/etc/nginx/sites-enabled/emptychair` — unchanged (was already correct)
- nginx enabled and started; certbot wildcard cert renewed
- DO droplet firewall: UFW allow 30000:39999/tcp

---

## SESSION 14 (February 15, 2026)

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

1. **FIX: Docker container network aliases** — orchestrator creates containers on deployment network but doesn't set service-name aliases. Ghost couldn't resolve `db` because MariaDB had no `db` alias. This must be fixed in `internal/shell/docker/orchestrator.go` so all multi-service templates work automatically.
2. **FIX: UFW firewall on provisioned nodes** — provisioner doesn't open app ports (30000-39999). New droplets block proxy traffic until manually opened. Add `ufw allow 30000:39999/tcp` to node provisioning.
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
| nginx | 80/443 | TLS termination, routes by hostname (prod only) |
| APIGate | 8082 | JWT auth + billing + routing (HTTP behind nginx in prod) |
| Hoster | 8080 | Backend: API + embedded SPA |
| Vite | 3000 | Dev hot-reload only (NOT for testing) |
| App Proxy | 9091 | Deployment routing (`*.apps.emptychair.dev`) |
