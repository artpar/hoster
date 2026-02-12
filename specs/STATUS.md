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

## IN PROGRESS

- [ ] Fix CI npm/rollup issues - see specs/SESSION-HANDOFF.md
- [ ] Production E2E testing of billing flow

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
| Setup | `setup.go` | ~270 | Router, middleware, custom actions |
| Handlers | `handlers.go` | ~350 | Deployment lifecycle commands |
| Commands | `commands.go` | ~100 | Command bus dispatch |
| Workers | `workers.go` | ~600 | Health, DNS, provisioner |
| Auth | `auth_bridge.go` | ~120 | APIGate header extraction |
| Migrate | `migrate.go` | ~120 | File + schema migrations |
| **Total** | | **~3,400** | |

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
