# Implementation Status

> Moved from CLAUDE.md to reduce file size. This is the canonical status tracker.

## DONE

- [x] Project structure, Makefile, specs/README.md
- [x] ADR-001 (Docker Direct), ADR-002 (Values as Boundaries)
- [x] Domain specs: template.md, deployment.md
- [x] Template domain (type, validation, 22 tests)
- [x] Deployment domain (type, state machine, 23 tests)
- [x] internal/core/compose/parser.go - Compose parsing with compose-go
- [x] internal/shell/docker/ - Docker SSH client + orchestrator
- [x] tests/e2e/ - E2E testing infrastructure
- [x] Docker integration (create/start/stop/restart/delete containers)
- [x] CI/CD workflows (ci.yml, release.yml)
- [x] v0.1.0 released and deployed to production
- [x] Container event recording - Lifecycle tracking
- [x] Deployment monitoring UI - Events, Stats, Logs tabs
- [x] Default marketplace templates - 12 templates (6 infra + 6 web-UI apps)
- [x] Local E2E environment - APIGate + Hoster integration fully working
- [x] Remote node E2E - AWS EC2 deployment verified (January 23, 2026)
- [x] SSH key management via web UI - AES-256-GCM encryption
- [x] Node registration via web UI - Health checks working
- [x] Dev auth removed — all auth via APIGate headers (February 2026)
- [x] Hoster-branded login/signup pages restored
- [x] **Engine rewrite (February 2026)** — Schema-driven generic CRUD replaces per-entity boilerplate
  - [x] Resource schema definitions (`internal/engine/resources.go`)
  - [x] Generic store with CRUD + state machine transitions (`internal/engine/store.go`)
  - [x] Generic JSON:API REST handlers (`internal/engine/api.go`)
  - [x] Command bus + deployment handlers (`internal/engine/commands.go`, `handlers.go`)
  - [x] Background workers (health, DNS, provisioner) (`internal/engine/workers.go`)
  - [x] Auth bridge from APIGate headers (`internal/engine/auth_bridge.go`)
  - [x] `cmd/hoster/server.go` rewritten to use engine
  - [x] `tests/e2e/e2e_test.go` migrated to engine
  - [x] Old packages deleted: `shell/api/`, `shell/store/`, `shell/workers/`, `shell/scheduler/`
  - [x] ~13,700 lines deleted, ~3,400 lines added (net reduction ~10,300 lines)

- [x] **Billing & Payments (February 2026)** — Usage-based billing with Stripe Checkout
  - [x] Usage event recording via APIGate metering (`internal/shell/billing/`)
  - [x] Background billing reporter batches events to APIGate
  - [x] Invoice entity with state machine: draft → pending → paid/failed (`internal/engine/resources.go`)
  - [x] Scheduled invoice generation — `InvoiceGenerator` worker auto-creates monthly invoices
  - [x] Stripe Checkout integration via REST API (no SDK) (`internal/engine/billing_handlers.go`)
  - [x] Payment verification on return from Stripe
  - [x] Billing page: costs, invoices, deployments, usage history (`web/src/pages/billing/BillingPage.tsx`)
  - [x] E2E verified: deploy app → invoice auto-generated → pay via Stripe → invoice marked paid

- [x] **Engine Rewrite Regression Fixes (v0.3.39, February 2026)** — Six regressions caught and fixed
  - [x] **S1 Security**: Cloud provision BeforeCreate verifies credential ownership (prevents cross-user abuse)
  - [x] **B1 Billing**: Deployment BeforeCreate enforces plan deployment limits + `DefaultPlanLimits` fallback
  - [x] **D1 Data Integrity**: Template BeforeDelete rejects deletion when active deployments exist (409)
  - [x] **E1 Domains**: Full domain management — list, add, remove, verify with DNS CNAME checking
  - [x] **E2 Maintenance**: Node maintenance toggle (POST → maintenance, DELETE → online)
  - [x] **F1 Invoices Frontend**: Invoice API client + TanStack Query hooks (`web/src/api/invoices.ts`)
  - [x] Cloud provision retry action handler (failed → pending/destroying)
  - [x] All regressions E2E verified through Chrome DevTools on local prod setup

- [x] **Cloud Provision Destroy + E2E Leak Detection (v0.3.46, February 15, 2026)**
  - [x] Fixed `deleteHandler` hardcoded `"deleting"` → uses matched target state (`t`)
  - [x] `DestroyInstance` command handler: decrypt credential, call provider, transition to destroyed
  - [x] Encryption key passed to command bus for credential decryption in handlers
  - [x] SSH key reuse for re-provisioning (restores b880707 fix lost in engine rewrite)
  - [x] E2E teardown verifies 0 leaked droplets via DO API — force-destroys any found + fails test
  - [x] E2E test suite rewritten to use browser UI (deleted api.fixture.ts)
  - [x] DO API key moved to env-only (`TEST_DO_API_KEY`) — no hardcoded fallback

- [x] **Deployment Access UX + Domains Tab Fix (v0.3.47, February 15, 2026)**
  - [x] Fixed `DeploymentAttributes` type: `domains[]` array replaces broken singular `domain?` string
  - [x] Fixed Domains tab: backend returns plain JSON but frontend expected JSON:API wrapped response
  - [x] Added "Access Your Application" card on deployment Overview tab (domain URL + direct IP:port)
  - [x] Added "Open App" button in deployment header when running
  - [x] Shows node name/IP and proxy port in Deployment Info section
  - [x] Fixed domain generation inconsistency: list/add/verify handlers use `domain.Slugify(name)` matching scheduleDeployment
  - [x] Fixed production env var names: `HOSTER_PROXY_*` (not `HOSTER_APP_PROXY_*`), added `HOSTER_DOMAIN_BASE_DOMAIN`

- [x] **Container Port Binding Fix (v0.3.48, February 15, 2026)**
  - [x] Changed `hostIP` from `127.0.0.1` to `0.0.0.0` in `internal/shell/docker/orchestrator.go`
  - [x] Deployed apps on remote nodes now accessible via public IP:port

- [x] **Deployment Restart Race Condition Fix (v0.3.49, February 15, 2026)**
  - [x] Fixed race between goroutine command dispatch and `stripFields()` mutating shared row map
  - [x] `maps.Clone(row)` before passing to goroutine in start and stop action handlers
  - [x] Stopped deployments can now restart without "template not found: templates id=0" error

- [x] **Billing Meter Path + Cloud Destroy Error Handling (v0.3.50, February 15, 2026)**
  - [x] Billing client: configurable meter path, default `/_internal/meter` (avoids `/api/*` route shadowing)
  - [x] Cloud provision destroy: fails to "failed" state on errors instead of silently marking "destroyed"
  - [x] Delete handler: returns 409 if destroy dispatch results in "failed" state (prevents orphaned cloud resources)
  - [x] Cloud provisions state machine: allow destroying from pending/creating/configuring states
  - [x] E2E teardown: detect leaked droplets matching all test prefixes
  - [x] Docs: updated `specs/local-e2e-setup.md` for APIGate v0.3.8, billing config, meter path steps
  - [x] Docs: rewrote stale `docs/local-e2e-development.md` to match current architecture

## IN PROGRESS

- [ ] Production E2E testing of billing flow
- [ ] Run full E2E suite (UJ2-UJ8) after v0.3.46

## MVP STATUS: COMPLETE + ENGINE REWRITE + BILLING

The core deployment loop is fully functional via the generic engine:
1. Creator creates template with docker-compose (generic CRUD)
2. Creator publishes template (custom action handler)
3. Creator registers remote nodes (generic CRUD + SSH key encryption)
4. Customer deploys from published template (generic CRUD)
5. Deployment gets auto-generated domain via state machine transition
6. State machine: pending → scheduled → starting → running (command handlers fire at each step)
7. Customer can start/stop/delete deployments (state machine transitions)
8. Deployments run on remote Docker hosts via SSH tunnel

## Engine Architecture (February 2026)

**Pattern:** Schema → Engine → Handlers

| Layer | File(s) | Lines | What it does |
|-------|---------|-------|-------------|
| Schema | `resources.go` | ~300 | Entity definitions as data |
| Store | `store.go` | ~800 | Generic CRUD + state machine |
| API | `api.go` | ~530 | Generic JSON:API REST handlers |
| Setup | `setup.go` | ~650 | Router, middleware, custom actions, domain/node handlers |
| Handlers | `handlers.go` | ~350 | Deployment lifecycle commands |
| Commands | `commands.go` | ~100 | Command bus dispatch |
| Workers | `workers.go` | ~600 | Health, DNS, provisioner |
| Auth | `auth_bridge.go` | ~180 | APIGate header extraction + default plan limits |
| Migrate | `migrate.go` | ~120 | File + schema migrations |
| **Total** | | **~4,000** | |

**What was deleted (~13,700 lines):**
- `shell/store/sqlite.go` (2,640 lines) — per-entity store methods
- `shell/store/sqlite_test.go` (1,995 lines) — per-entity store tests
- `shell/api/resources/` (2,725 lines) — 6 api2go resource files
- `shell/api/setup.go` + handlers (1,430 lines) — HTTP setup + domain/monitoring handlers
- `shell/api/openapi/` (592 lines) — OpenAPI generator
- `shell/api/middleware/` (413 lines) — auth middleware
- `shell/workers/` (1,379 lines) — health checker, provisioner, DNS verifier
- `shell/scheduler/` (447 lines) — scheduling service
- `shell/store/migrations/` (1,453 lines) — 11 migration files
- Misc: webui.go, errors.go, store interface

## What's Next

- Production E2E testing of full billing flow (Stripe live mode)
- Stripe webhook integration for async payment confirmation
- Plan upgrade flow (Free → Pro via Stripe Checkout)
