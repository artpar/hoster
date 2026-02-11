# CLAUDE.md - Project Memory for Hoster

> **CRITICAL**: Read this file completely before making ANY changes to this project.
> This file is the source of truth for project decisions, methodology, and architecture.

## Project Identity

**Hoster** is a modern deployment marketplace platform - like Railway/Render/Heroku but self-hosted with a template marketplace.

**Vision**: Package creators define deployment templates (docker-compose + config + pricing), customers one-click deploy instances onto YOUR VPS infrastructure.

**Status**: Backend deployed to production at https://emptychair.dev. **Engine rewrite complete** — generic schema-driven CRUD replaced ~13,700 lines of per-entity boilerplate with a ~3,400-line generic engine. Monitoring features complete. Remote node deployment verified on AWS EC2. 12 marketplace templates. All auth via APIGate headers.

---

## Engineering Methodology: STC (Spec → Test → Code)

### THE CARDINAL RULES - NEVER VIOLATE

```
┌─────────┐         ┌─────────┐         ┌─────────┐
│  SPEC   │ ──────► │  TEST   │ ──────► │  CODE   │
│         │         │         │         │         │
│ Source  │         │ Verify  │         │ Implement│
│ of Truth│ ◄────── │ Behavior│ ◄────── │ Behavior │
└─────────┘         └─────────┘         └─────────┘
```

1. **NEVER write code without a spec first** → Write spec in `specs/` directory
2. **NEVER write code without tests first** → Write tests based on spec
3. **NEVER skip the flow** → Always: Spec → Test → Code
4. **Sync is MANDATORY** → If spec changes, update tests, then code
5. **Specs live in `specs/`** → Domain specs, feature specs, ADRs

### File Locations for STC

| What | Where |
|------|-------|
| Domain specs | `specs/domain/{entity}.md` |
| Feature specs | `specs/features/F###-{name}.md` |
| ADRs | `specs/decisions/ADR-###-{name}.md` |
| Unit tests | `internal/core/{package}/*_test.go` |
| Integration tests | `internal/shell/{package}/*_test.go` |
| E2E tests | `tests/e2e/*_test.go` |

---

## Architecture: Schema → Engine → Handlers

### THE PATTERN

```
┌──────────────────────────────────────┐
│     RESOURCE SCHEMA (data)           │  resources.go (~300 lines)
│  entities, fields, state machines,   │  Define once, get CRUD for free
│  relationships, validations          │
└──────────────────┬───────────────────┘
                   │ interpreted by
                   ▼
┌──────────────────────────────────────┐
│     GENERIC ENGINE (code)            │  store.go + api.go (~1,200 lines)
│  auto-CRUD, REST API, state machine  │  One implementation, all entities
│  enforcer, migration generator       │
└──────────────────┬───────────────────┘
                   │ emits transition events to
                   ▼
┌──────────────────────────────────────┐
│     COMMAND HANDLERS (code)          │  handlers.go + workers.go (~2,000 lines)
│  StartDeployment, StopDeployment,    │  Side effects on state transitions
│  ProvisionNode, HealthCheck          │
└──────────────────┬───────────────────┘
                   │ calls into
                   ▼
┌──────────────────────────────────────┐
│     INFRASTRUCTURE (code)            │  shell/ (~1,500 lines)
│  Docker SSH client, cloud providers, │  Irreducible I/O
│  reverse proxy, billing reporter     │
└──────────────────────────────────────┘
```

### Layers

1. **`internal/engine/resources.go`** = Entity definitions as data. ~50 lines per entity. Adding a new entity = add schema definition, get full CRUD + API + migration for free.

2. **`internal/engine/store.go`** = Generic CRUD. One `Create/Get/List/Update/Delete/Transition` works for all entities. State machine transitions validated + commands dispatched automatically.

3. **`internal/engine/api.go`** = Generic REST handlers. JSON:API format for all entities. No per-entity handler code.

4. **`internal/engine/handlers.go`** = Business logic. Command handlers fire on state transitions (e.g., entering "starting" dispatches `StartDeployment`).

5. **`internal/shell/`** = Thin I/O only: Docker SSH, cloud providers, reverse proxy, billing.

6. **`internal/core/`** = Pure functions only. NO I/O. Domain types, compose parsing, validation.

### Rules

- **Add new entity**: Add to `resources.go` (~50 lines). Done.
- **Add new state**: Add transition to schema's `StateMachine.Transitions` + command handler.
- **No per-entity CRUD code**: The engine handles it generically.
- **Shell consumers use narrow interfaces**: `proxy.ProxyStore` (2 methods), `billing.BillingStore` (3 methods), `docker.NodeStore` (2 methods).

---

## Directory Structure

```
hoster/
├── CLAUDE.md                   # THIS FILE - READ FIRST
├── specs/                      # SOURCE OF TRUTH
│   ├── domain/                 # Entity specifications
│   ├── features/               # Feature specifications
│   └── decisions/              # Architecture Decision Records
├── internal/
│   ├── core/                   # FUNCTIONAL CORE (no I/O)
│   │   ├── domain/             # Domain types + validation
│   │   ├── compose/            # Compose parsing
│   │   ├── deployment/         # Deployment logic
│   │   ├── traefik/            # Traefik config generation
│   │   ├── auth/               # Auth context extraction
│   │   ├── crypto/             # AES encryption
│   │   ├── limits/             # Plan limits
│   │   └── monitoring/         # Health aggregation
│   ├── engine/                 # GENERIC ENGINE (schema → CRUD → API)
│   │   ├── schema.go           # Resource/Field/StateMachine types
│   │   ├── resources.go        # Schema() — all entity definitions as data
│   │   ├── store.go            # Generic CRUD store (Create/Get/List/Update/Delete/Transition)
│   │   ├── api.go              # Generic REST handlers (JSON:API format)
│   │   ├── setup.go            # HTTP router setup + middleware + action handlers
│   │   ├── handlers.go         # Command handlers (start/stop/delete deployment)
│   │   ├── commands.go         # Command bus (dispatch side-effects on state transitions)
│   │   ├── workers.go          # Background workers (health, DNS, provisioner)
│   │   ├── auth_bridge.go      # Auth context from APIGate headers
│   │   └── migrate.go          # File + schema migrations
│   └── shell/                  # IMPERATIVE SHELL (I/O only)
│       ├── docker/             # Docker SSH client + orchestrator
│       ├── proxy/              # Reverse proxy (*.apps.domain → containers)
│       ├── billing/            # APIGate usage reporter
│       ├── dns/                # DNS verification client
│       └── provider/           # Cloud providers (AWS/DO/Hetzner)
├── web/                        # FRONTEND (React + Vite + TailwindCSS)
├── tests/e2e/                  # End-to-end tests
├── cmd/hoster/                 # Entry point
└── go.mod
```

---

## Key Decisions (MUST FOLLOW)

### ADR-001: Docker Direct (No Orchestration)

- **Decision**: Use Docker API directly, no Swarm/K8s
- **Rationale**: Minimal overhead for prototype, full control
- **Implication**: We manage containers ourselves via `github.com/docker/docker/client`
- **DO NOT**: Add Kubernetes, Nomad, or Docker Swarm

### ADR-002: Values as Boundaries

- **Decision**: Functional core, imperative shell
- **Rationale**: Testability, minimal tech debt
- **Implication**: Core has NO I/O, shell is thin
- **DO NOT**: Put I/O in `internal/core/`

### ADR-003: JSON:API Format (Generic Engine)

- **Decision**: Use JSON:API format via the generic engine (replaced api2go with schema-driven handlers)
- **Rationale**: Standardized format, zero per-entity handler code
- **Implication**: Consistent `{data: {type, id, attributes}}` format for all entities
- **DO NOT**: Write per-entity API handler code
- **Spec**: `specs/decisions/ADR-003-jsonapi-api2go.md`

### ADR-004: Schema-Driven Types

- **Decision**: Generate types from schema definitions in `resources.go`
- **Rationale**: Single source of truth for entities, no drift between schema/API/types
- **Implication**: Schema drives migrations, CRUD, API responses, and validation
- **DO NOT**: Write per-entity store methods or migration SQL
- **Spec**: `specs/decisions/ADR-004-reflective-openapi.md`

### ADR-005: APIGate Integration

- **Decision**: Use APIGate as reverse proxy for auth and billing
- **Rationale**: Leverage existing auth/billing infrastructure
- **Implication**: Trust X-User-ID headers injected by APIGate. Hoster has NO custom auth — no login/signup endpoints, no session management. Frontend sends JWT Bearer tokens; APIGate validates them, injects X-User-ID/X-Plan-ID headers, and forwards to Hoster. Auth endpoints are at `/mod/auth/*`.
- **DO NOT**: Build auth from scratch, add login/signup endpoints to Hoster, or use external auth providers
- **Spec**: `specs/decisions/ADR-005-apigate-integration.md`

### ADR-006: Frontend Architecture

- **Decision**: React + Vite + TanStack Query + Zustand + TailwindCSS
- **Rationale**: Modern stack, good DX, strong typing from OpenAPI
- **Implication**: Separate frontend build in `web/` directory
- **DO NOT**: Use Vue, Angular, or server-rendered templates
- **Spec**: `specs/decisions/ADR-006-frontend-architecture.md`

### ADR-007: UI/UX Implementation Guidelines

- **Decision**: shadcn/ui components, semantic colors, consistent patterns
- **Rationale**: Consistency, correctness, completeness across all UI
- **Implication**: Follow design system for all components
- **DO NOT**: Use raw colors, inconsistent spacing, skip loading/error states
- **Spec**: `specs/decisions/ADR-007-uiux-guidelines.md`

### SQLite for Prototype

- **Decision**: Use SQLite now, migrate to PostgreSQL later
- **Rationale**: Fast start, easy to develop
- **Implication**: Use `sqlx` with SQLite driver
- **DO NOT**: Start with PostgreSQL yet

### Library Choices (USE THESE, NOT ALTERNATIVES)

| Purpose | Library | DO NOT USE |
|---------|---------|------------|
| Docker SDK | `github.com/docker/docker/client` | Other Docker libs |
| Compose parsing | `github.com/compose-spec/compose-go/v2` | Custom parser |
| HTTP router | `github.com/gorilla/mux` | chi, gin, echo |
| JSON:API | Generic engine (`internal/engine/api.go`) | api2go, custom marshaling |
| Database | `github.com/jmoiron/sqlx` | gorm, ent |
| Migrations | `github.com/golang-migrate/migrate/v4` + schema-driven | goose, others |
| Testing | `github.com/stretchr/testify` | Other assertion libs |
| UUID | `github.com/google/uuid` | Other UUID libs |
| Config | `github.com/spf13/viper` | Other config libs |
| Logging | `log/slog` (stdlib) | logrus, zap |

---

## Entity ID Pattern (STANDARD — ALL ENTITIES)

Every entity in the system has **two IDs**:

| Column | Type | Purpose | Used By |
|--------|------|---------|---------|
| `id` | `INTEGER PRIMARY KEY` | Internal DB auto-increment PK | Foreign keys, DB joins, internal references |
| `reference_id` | `TEXT UNIQUE` | External UUID-like ID (e.g., `tmpl_abc123`, `depl_xyz789`, `user_bc6849d9`) | API responses, frontend, URLs, logs |

### Standard Columns (all entities)

| Column | Type | Description |
|--------|------|-------------|
| `id` | `INTEGER PRIMARY KEY` | Auto-increment, used in FK columns |
| `reference_id` | `TEXT UNIQUE NOT NULL` | External identifier, prefixed by type |
| `created_at` | `DATETIME` | Creation timestamp |
| `updated_at` | `DATETIME` | Last modification timestamp |

### Rules — NEVER VIOLATE

1. **FK columns use `id` (integer PK)** — e.g., `deployments.customer_id` references `users.id`, `deployments.template_id` references `templates.id`
2. **API/frontend use `reference_id`** — all JSON:API responses expose `reference_id` as the resource `id`, never the integer PK
3. **Store lookups by `reference_id`** — `store.Get(ctx, "templates", "tmpl_abc123")` looks up by `reference_id`, returns `map[string]any` with both `id` (int) and `reference_id` (string)
4. **Engine works with `map[string]any`** — generic store returns maps, domain conversion at boundaries (`mapToDeployment`, `mapToNode`, etc.)
5. **JSON:API resource conversion** — generic `rowToJSONAPI()` maps `reference_id` → JSON:API `id`, strips internal `id` field

### Reference ID Prefixes

| Entity | Prefix | Example |
|--------|--------|---------|
| Template | `tmpl_` | `tmpl_wordpress` |
| Deployment | `depl_` | `depl_32f7e16a` |
| Node | `node_` | `node_abc123` |
| SSH Key | `sshkey_` | `sshkey_def456` |
| User | (from APIGate) | `c22469ce-d68b-4e5b-...` or `user_bc6849d9` |
| Event | `evt_` | `evt_abc123` |

---

## Domain Model (CURRENT STATE)

### Template (IMPLEMENTED)

- **Spec**: `specs/domain/template.md`
- **Code**: `internal/core/domain/template.go`
- **Tests**: `internal/core/domain/template_test.go` (22 tests)

Key validation rules:
- Name: 3-100 chars, alphanumeric + spaces + hyphens
- Version: semver X.Y.Z format
- ComposeSpec: non-empty (full validation TODO)
- Price: >= 0

### Deployment (IMPLEMENTED)

- **Spec**: `specs/domain/deployment.md`
- **Code**: `internal/core/domain/deployment.go`
- **Tests**: `internal/core/domain/deployment_test.go` (23 tests)

**State Machine (CRITICAL - DO NOT CHANGE WITHOUT UPDATING SPEC)**:
```
pending → scheduled → starting → running → stopping → stopped → deleting → deleted
                         ↓          ↓          ↓
                       failed ← ← ← ←
```

Valid transitions (exhaustive list):
- pending → scheduled
- scheduled → starting
- starting → running, failed
- running → stopping, failed
- stopping → stopped
- stopped → starting, deleting
- deleting → deleted
- failed → starting, deleting

---

## Testing Strategy

### Unit Tests (`internal/core/`)
- **NO mocks** - Pure values in, values out
- Run with: `make test-unit`
- Coverage target: >90%

### Integration Tests (`internal/shell/`)
- Real Docker, real SQLite
- Run with: `make test-integration`

### E2E Tests (`tests/e2e/`)
- Full API calls, real system
- Run with: `make test-e2e`

### Commands
```bash
make test           # All tests
make test-unit      # Core tests only
make test-integration # Shell tests only
make coverage       # Generate coverage report
```

---

## Production Testing (MANDATORY)

Before ANY production deployment: test all 10 user journeys in `specs/user-journeys.md` on https://emptychair.dev using Chrome DevTools MCP. Document results. **Unit tests are necessary but not sufficient.**

---

## No-Bypass Policy (CRITICAL)

**Fix root causes, never bypass.** If broken: identify root cause → file issue → wait for fix → test → deploy. Never comment out tests, skip manual testing, build auth in Hoster, or deploy without all journeys passing.

---

## What's NOT Supported (BY DESIGN)

These are intentional limitations documented in specs:

### Template
- No template inheritance
- No dynamic pricing (fixed per template)
- No private templates (all public)
- No collaborative editing

### Deployment
- No scaling replicas (one instance per service)
- No zero-downtime updates
- No automatic restarts
- No resource limits enforcement

**DO NOT** implement these without updating specs first.

---

## Implementation Status

**MVP: COMPLETE** — Full deployment loop + monitoring + remote nodes working.
**Engine rewrite: COMPLETE** — Replaced ~13,700 lines of per-entity boilerplate with ~3,400-line generic engine. Old packages (`shell/api`, `shell/store`, `shell/workers`, `shell/scheduler`) deleted.

See **`specs/STATUS.md`** for full implementation details and roadmap.

---

## Commands Reference

```bash
# Development
make deps           # Download dependencies
make build          # Build binary
make run            # Build and run
make dev            # Run without building

# Testing
make test           # Run all tests
make test-unit      # Run core tests only
make test-integration # Run shell tests only
make test-e2e       # Run e2e tests
make coverage       # Generate coverage report

# Code quality
make fmt            # Format code
make vet            # Vet code
```

---

## Frontend Architecture

The engine serves a minimal SPA shell at `/` (non-API paths). The full React frontend is in `web/`.

**Local Development:**
- Frontend: `cd web && npm run dev` (Vite dev server on :3000 or :5173)
- Backend: `make run` (Hoster on :8080)
- Vite proxies `/api` to backend

**Key Files:**
- `internal/engine/setup.go` — SPA handler at `PathPrefix("/")`
- `web/vite.config.ts` — Base path set to `/`

---

## Production Deployment (emptychair.dev)

**Services:** APIGate (80/443, TLS + auth + routing) → Hoster (8080, API) + App Proxy (9091, `*.apps.emptychair.dev`)

**Management:** `cd deploy/local && make status|logs|restart|deploy-release`

**CI/CD:** `ci.yml` (push to main/PRs), `release.yml` (version tags → GitHub release)
**Build:** `cd web && npm install && npm run build` → copy dist → `go build ./cmd/hoster`
**Deploy:** `cd deploy/local && make deploy-release VERSION=v0.2.0`

**Known issue:** CI failing with `@rollup/rollup-linux-x64-gnu` — see `specs/SESSION-HANDOFF.md`

---

## Anti-Patterns (NEVER DO THESE)

| DON'T | DO INSTEAD |
|-------|------------|
| Write code without spec | Write spec first in `specs/` |
| Write code without tests | Write tests based on spec first |
| Put I/O in `internal/core/` | I/O only in `internal/shell/` |
| Use mocks for core tests | Test with real values |
| Skip state machine transitions | Update spec if transitions need to change |
| Add features not in spec | Update spec first, then implement |
| Use different libraries | Use the ones listed in this file |
| Implement "NOT Supported" items | Check spec, these are intentional |
| **Use direct DB queries on APIGate** | **Use APIGate API/UI. If API doesn't exist, report bug on github.com/artpar/apigate/issues** |
| Modify production via sqlite3 commands | Use self-service APIs or admin UI |
| Skip checking for APIGate releases | Always `gh release list --repo artpar/apigate` before debugging |

---

## Project Management

- **Agile Project**: HOSTER (created in Agile MCP)
- **Methodology**: Scrum
- **Epics**:
  - HOSTER-S1: Project Foundation & Architecture
  - HOSTER-S2: Template System
  - HOSTER-S3: Deployment Engine

Use `mcp__agile__*` tools to track tasks.

---

## Commit Message Guidelines

- Use conventional commit format: `type: description`
- Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
- **DO NOT** add Co-Authored-By lines or AI attribution to commits
- **DO NOT** sign commits with Claude/Opus or any AI identity
- Keep commit messages concise and focused on the "what" and "why"

---

## External Resources

### APIGate Integration

**CRITICAL: APIGate is the front-facing server. Hoster sits behind it. All traffic goes through APIGate.**

```
Internet → APIGate (:8082, front-facing) → Hoster (:8080, backend)
```

**Separation of concerns:**
- **APIGate = Billing/Quota.** Only deployment CRUD is billable. NOT responsible for auth.
- **Hoster = Business Logic.** Reads user identity from APIGate-injected X-User-ID header. No custom auth endpoints.

**Full architecture spec:** `specs/architecture/apigate-integration.md`

- **APIGate Repository**: https://github.com/artpar/apigate
- **APIGate Issues**: https://github.com/artpar/apigate/issues
- **APIGate Wiki**: https://github.com/artpar/apigate/wiki

**Route pattern rules:** Prefix routes MUST use `/*` wildcard (e.g., `/api/*` not `/api/`).

When encountering APIGate-related issues during development or testing, report them at the issues link above. The `gh` CLI is available and logged in for creating issues.

### Auth Flow (JWT-Only, APIGate v2.0.0+)

Login/signup pages at `/login`, `/signup` (Hoster-branded) → call APIGate `/mod/auth/*` endpoints → JWT stored in localStorage → sent as `Authorization: Bearer {token}` → APIGate validates, injects `X-User-ID`/`X-Plan-ID` headers → Hoster reads headers via `ResolveUser()`.

**Key files:** `web/src/stores/authStore.ts`, `web/src/api/client.ts`, `web/src/pages/auth/`
**Removed:** `HOSTER_AUTH_MODE`, `HOSTER_NODES_ENABLED`, cookie/session auth
**Download:** `gh release download v2.0.0 --repo artpar/apigate`

---

## Local E2E Testing Environment

**Location:** `/tmp/hoster-e2e-test/` — See **`specs/local-e2e-setup.md`** for full setup, commands, and troubleshooting.

**Key points:**
- All access through APIGate (localhost:8082) — never Hoster directly (8080)
- Routes: billing (`/api/v1/deployments*`, auth_required=1), API (`/api/*`, pass-through), frontend (`/*`, public)
- `HOSTER_DATA_DIR=/tmp/hoster-e2e-test` for consistent DB across restarts
