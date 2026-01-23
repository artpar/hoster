# CLAUDE.md - Project Memory for Hoster

> **CRITICAL**: Read this file completely before making ANY changes to this project.
> This file is the source of truth for project decisions, methodology, and architecture.

## Project Identity

**Hoster** is a modern deployment marketplace platform - like Railway/Render/Heroku but self-hosted with a template marketplace.

**Vision**: Package creators define deployment templates (docker-compose + config + pricing), customers one-click deploy instances onto YOUR VPS infrastructure.

**Status**: Backend deployed to production at https://emptychair.dev. Monitoring features complete. Local E2E environment fully functional. **Remote node deployment verified on AWS EC2.** Ready for v0.2.2 release.

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

## Architecture: Values as Boundaries

### THE PATTERN - NEVER VIOLATE

```
┌────────────────────────────────────────────────────────────────┐
│                     IMPERATIVE SHELL                           │
│  internal/shell/ - HTTP handlers, Docker client, Database     │
│                                                                 │
│    ┌──────────────────────────────────────────────────────┐    │
│    │                 FUNCTIONAL CORE                       │    │
│    │  internal/core/ - Pure functions, NO I/O              │    │
│    │                                                       │    │
│    │   • Takes VALUES in (structs, primitives)            │    │
│    │   • Returns VALUES out (structs, errors)             │    │
│    │   • Has NO side effects                              │    │
│    │   • Is trivially testable (no mocks needed)          │    │
│    └──────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────┘
```

### Rules

1. **`internal/core/`** = Pure functions only. NO:
   - Database calls
   - HTTP calls
   - File I/O
   - Docker API calls
   - Any external dependency

2. **`internal/shell/`** = Thin I/O wrapper. Pattern:
   - Read data (I/O) → Pass values to core
   - Core returns decisions → Shell executes them

3. **Testing**:
   - Core tests: NO mocks, just values in/out
   - Shell tests: Real dependencies (Docker, SQLite)

---

## Directory Structure

```
hoster/
├── CLAUDE.md                   # THIS FILE - READ FIRST
├── specs/                      # SOURCE OF TRUTH
│   ├── README.md               # How to write specs
│   ├── SESSION-HANDOFF.md      # New session onboarding protocol
│   ├── domain/                 # Entity specifications
│   │   ├── template.md         # Template entity (IMPLEMENTED)
│   │   ├── deployment.md       # Deployment entity (IMPLEMENTED)
│   │   ├── monitoring.md       # Monitoring types (SPEC READY)
│   │   └── user-context.md     # Auth context (SPEC READY)
│   ├── features/               # Feature specifications
│   │   ├── F008-authentication.md       # Auth integration (SPEC READY)
│   │   ├── F009-billing-integration.md  # Billing (SPEC READY)
│   │   ├── F010-monitoring-dashboard.md # Monitoring (SPEC READY)
│   │   ├── F011-marketplace-ui.md       # Marketplace UI (SPEC READY)
│   │   ├── F012-deployment-management-ui.md # Deployment UI (SPEC READY)
│   │   └── F013-creator-dashboard-ui.md # Creator UI (SPEC READY)
│   └── decisions/              # Architecture Decision Records
│       ├── ADR-001-docker-direct.md       # (IMPLEMENTED)
│       ├── ADR-002-values-as-boundaries.md # (IMPLEMENTED)
│       ├── ADR-003-jsonapi-api2go.md      # JSON:API (SPEC READY)
│       ├── ADR-004-reflective-openapi.md  # OpenAPI gen (SPEC READY)
│       ├── ADR-005-apigate-integration.md # APIGate auth (SPEC READY)
│       ├── ADR-006-frontend-architecture.md # React frontend (SPEC READY)
│       └── ADR-007-uiux-guidelines.md     # UI/UX patterns (SPEC READY)
├── internal/
│   ├── core/                   # FUNCTIONAL CORE (no I/O)
│   │   ├── domain/             # Domain types + validation (IMPLEMENTED)
│   │   ├── compose/            # Compose parsing (IMPLEMENTED)
│   │   ├── deployment/         # Deployment logic (IMPLEMENTED)
│   │   ├── traefik/            # Traefik config generation (IMPLEMENTED)
│   │   ├── auth/               # Auth context (TODO - F008)
│   │   ├── limits/             # Plan limits (TODO - F009)
│   │   └── monitoring/         # Health aggregation (TODO - F010)
│   └── shell/                  # IMPERATIVE SHELL (I/O)
│       ├── api/                # HTTP handlers (IMPLEMENTED)
│       │   ├── resources/      # api2go resources (TODO - ADR-003)
│       │   ├── openapi/        # OpenAPI generator (TODO - ADR-004)
│       │   └── middleware/     # Auth middleware (TODO - F008)
│       ├── docker/             # Docker SDK wrapper (IMPLEMENTED)
│       ├── store/              # Database layer (IMPLEMENTED)
│       └── billing/            # APIGate billing client (TODO - F009)
├── web/                        # FRONTEND (TODO - ADR-006)
│   ├── src/
│   │   ├── api/                # API client + generated types
│   │   ├── pages/              # Page components
│   │   ├── components/         # Reusable components
│   │   ├── hooks/              # TanStack Query hooks
│   │   └── stores/             # Zustand stores
│   └── package.json
├── tests/
│   ├── e2e/                    # End-to-end tests (IMPLEMENTED)
│   └── fixtures/               # Test data (IMPLEMENTED)
├── examples/                   # Sample templates (IMPLEMENTED)
├── cmd/hoster/                 # Entry point (IMPLEMENTED)
├── Makefile                    # Build commands (IMPLEMENTED)
└── go.mod                      # Go module (IMPLEMENTED)
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

### ADR-003: JSON:API with api2go

- **Decision**: Use JSON:API specification via api2go library
- **Rationale**: Standardized format, relationship support, tooling ecosystem
- **Implication**: Consistent `{type, id, attributes, relationships}` format
- **DO NOT**: Use custom JSON format or GraphQL
- **Spec**: `specs/decisions/ADR-003-jsonapi-api2go.md`

### ADR-004: Reflective OpenAPI Generation

- **Decision**: Generate OpenAPI 3.0 spec at runtime by reflecting on api2go resources
- **Rationale**: Spec always matches implementation, no drift
- **Implication**: Serve `/openapi.json` endpoint, generate TypeScript types
- **DO NOT**: Use annotation-based generation (swaggo) or manual spec
- **Spec**: `specs/decisions/ADR-004-reflective-openapi.md`

### ADR-005: APIGate Integration

- **Decision**: Use APIGate as reverse proxy for auth and billing
- **Rationale**: Leverage existing auth/billing infrastructure
- **Implication**: Trust X-User-ID headers, network isolation required
- **DO NOT**: Build auth from scratch or use external auth providers
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
| JSON:API | `github.com/manyminds/api2go` | Custom marshaling |
| OpenAPI types | `github.com/getkin/kin-openapi` | Other OpenAPI libs |
| Database | `github.com/jmoiron/sqlx` | gorm, ent |
| Migrations | `github.com/golang-migrate/migrate/v4` | goose, others |
| Testing | `github.com/stretchr/testify` | Other assertion libs |
| UUID | `github.com/google/uuid` | Other UUID libs |
| Config | `github.com/spf13/viper` | Other config libs |
| Logging | `log/slog` (stdlib) | logrus, zap |

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

## Current Implementation Status

### DONE ✅
- [x] Project structure created
- [x] Makefile with test commands
- [x] specs/README.md (spec guidelines)
- [x] ADR-001: Docker Direct
- [x] ADR-002: Values as Boundaries
- [x] specs/domain/template.md
- [x] specs/domain/deployment.md
- [x] Template domain (type, validation, tests)
- [x] Deployment domain (type, state machine, tests)
- [x] internal/core/compose/parser.go - Compose parsing with compose-go
- [x] internal/shell/docker/client.go - Docker SDK wrapper
- [x] internal/shell/docker/orchestrator.go - Deployment lifecycle
- [x] internal/shell/store/sqlite.go - SQLite storage
- [x] internal/shell/api/handler.go - HTTP API handlers
- [x] tests/e2e/ - E2E testing infrastructure
- [x] Docker integration (create/start/stop/restart/delete containers)
- [x] internal/shell/apigate/client.go - Fixed admin API prefix (/admin/ instead of /api/)
- [x] internal/shell/api/webui.go - Embedded frontend handler (following APIGate pattern)
- [x] .github/workflows/ci.yml - CI workflow with frontend build
- [x] .github/workflows/release.yml - Release workflow for versioned releases
- [x] v0.1.0 released and deployed to production (backend only)
- [x] Container event recording in orchestrator - Lifecycle tracking
- [x] Deployment monitoring UI - Events, Stats, Logs tabs working
- [x] Default marketplace templates - 6 templates with pricing and resource limits
- [x] Local E2E environment - APIGate + Hoster integration fully working
- [x] Remote node E2E - AWS EC2 deployment verified (January 23, 2026)
- [x] React dialog components - ConfirmDialog and AlertDialog replacing native dialogs
- [x] SSH key management via web UI - Encrypted storage with AES-256-GCM
- [x] Node registration via web UI - Health checks working

### IN PROGRESS
- [ ] Fix CI npm/rollup issues - see specs/SESSION-HANDOFF.md for details
- [ ] Create v0.2.2 release with monitoring features
- [ ] Deploy v0.2.2 to production
- [ ] Production E2E testing with monitoring features

### Test Counts (January 23, 2026)
| Suite | Count | Status |
|-------|-------|--------|
| Unit (core/) | ~180 | PASS |
| Integration (shell/) | ~150 | PASS |
| E2E | ~100 | PASS |
| **Total** | **427** | **ALL PASS** |

### Remote Node E2E Test: ✅ VERIFIED (January 23, 2026)
- AWS EC2 instance: 98.82.190.29
- Node: aws-test-node (online)
- Deployment: test-app-mkqrkyep (running)
- Events: container_created, container_started

### MVP STATUS: ✅ COMPLETE + MONITORING + REMOTE NODES

The core deployment loop is fully functional:
1. ✅ Creator creates template with docker-compose
2. ✅ Creator publishes template
3. ✅ Creator registers remote nodes (SSH key + node)
4. ✅ Customer deploys from published template
5. ✅ Deployment gets auto-generated domain
6. ✅ Deployment gets Traefik labels for external routing
7. ✅ Scheduler assigns deployment to available node
8. ✅ Customer can start/stop/restart deployments
9. ✅ Customer can delete deployments
10. ✅ Customer can monitor deployments (Events, Stats, Logs)
11. ✅ **Deployments run on remote Docker hosts** (verified on AWS EC2)

### Monitoring Features: ✅ COMPLETE (January 22, 2026)

1. ✅ **Container Event Recording**
   - Events tracked: created, started, stopped, restarted, died, OOM, health checks
   - Stored in `container_events` table with timestamps
   - Orchestrator records events after each lifecycle operation

2. ✅ **Stats Monitoring**
   - Real-time CPU, memory, network, disk I/O metrics
   - Endpoint: `GET /api/v1/deployments/{id}/monitoring/stats`
   - UI: Stats tab shows current resource usage

3. ✅ **Logs Streaming**
   - Container logs with timestamps
   - Filtering by container name and tail count
   - Endpoint: `GET /api/v1/deployments/{id}/monitoring/logs`
   - UI: Logs tab shows scrollable log output

4. ✅ **Events Timeline**
   - Deployment lifecycle history
   - Event types, messages, and timestamps
   - Endpoint: `GET /api/v1/deployments/{id}/monitoring/events`
   - UI: Events tab shows chronological event list

### Remote Node Management: ✅ COMPLETE (January 23, 2026)

Full support for deploying to remote Docker hosts:

1. ✅ **SSH Key Management**
   - Add/delete SSH keys via Creator Dashboard
   - Keys encrypted with AES-256-GCM (32-byte key required)
   - Stored in `ssh_keys` table with encrypted private key

2. ✅ **Node Registration**
   - Register remote nodes via Creator Dashboard
   - Health checks verify connectivity and Docker availability
   - Nodes must be owned by template creator for scheduling

3. ✅ **Remote Deployment**
   - Scheduler assigns deployments to available nodes
   - SSH tunnel used for Docker API communication
   - Container events recorded from remote operations
   - Verified on AWS EC2 (98.82.190.29)

4. ✅ **UI Components**
   - React ConfirmDialog for destructive actions (delete node/key)
   - AlertDialog for notifications
   - No native browser dialogs (confirm/alert) used

**Critical Note:** Encryption key must be consistent across restarts:
```bash
HOSTER_ENCRYPTION_KEY=12345678901234567890123456789012  # exactly 32 bytes
```

### Default Marketplace Templates: ✅ COMPLETE (January 22, 2026)

6 production-ready templates with pricing and resource limits:
- PostgreSQL Database ($5/month, 512MB RAM, 0.5 CPU, 5GB disk)
- MySQL Database ($5/month, 512MB RAM, 0.5 CPU, 5GB disk)
- Redis Cache ($3/month, 256MB RAM, 0.25 CPU, 2GB disk)
- MongoDB Database ($5/month, 512MB RAM, 0.5 CPU, 10GB disk)
- Nginx Web Server ($2/month, 64MB RAM, 0.1 CPU, 512MB disk)
- Node.js Application ($4/month, 256MB RAM, 0.5 CPU, 2GB disk)

### ADR-002 Compliance: ✅ COMPLETE

All pure logic has been moved to `internal/core/`:

#### `internal/core/deployment/` package
- [x] `naming.go` - networkName(), volumeName(), containerName()
- [x] `ordering.go` - topologicalSort() for service dependencies
- [x] `container.go` - buildContainerSpec() mapping
- [x] `ports.go` - convertPorts() transformation
- [x] `variables.go` - substituteVariables() for env var substitution
- [x] `planner.go` - DetermineStartPath(), CanStopDeployment()

#### `internal/core/traefik/` package
- [x] `labels.go` - GenerateLabels() for Traefik routing

#### `internal/core/validation/` package
- [x] `template.go` - ValidateCreateTemplateFields(), CanUpdateTemplate(), CanCreateDeployment()

#### `internal/core/domain/` package
- [x] `slug.go` - Slugify() for URL-safe names
- [x] `deployment.go` - GenerateDomain() for auto domain assignment

### What's Next (Post-MVP) - SPECS COMPLETE

All specs for the next phase have been written. Implementation can proceed following STC.

**Phase 0: API Layer Migration** (ADR-003, ADR-004)
- [ ] Migrate from chi to Gorilla mux
- [ ] Implement api2go resources for Template, Deployment
- [ ] Build reflective OpenAPI generator
- [ ] Serve `/openapi.json` endpoint

**Phase 1: Authentication** (F008, ADR-005)
- [ ] Create auth middleware (extract X-User-ID headers)
- [ ] Implement authorization functions (pure core)
- [ ] Add auth checks to resources

**Phase 2: Billing** (F009)
- [ ] Create usage event storage
- [ ] Implement APIGate billing client
- [ ] Add plan limit validation
- [ ] Background event reporter

**Phase 3: Monitoring** (F010)
- [ ] Add health/logs/stats/events endpoints
- [ ] Implement Docker stats integration
- [ ] Create event recording in orchestrator

**Phase 4-6: Frontend** (F011, F012, F013, ADR-006)
- [ ] Set up React + Vite + TailwindCSS
- [ ] Generate TypeScript types from OpenAPI
- [ ] Implement Marketplace UI
- [ ] Implement Deployment Management UI
- [ ] Implement Creator Dashboard UI

### Blocked
- Nothing blocked - specs complete, implementation ready

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

## Embedded Frontend Architecture

Following APIGate's pattern, the frontend is embedded into the Go binary using `//go:embed`.

```
internal/shell/api/
├── setup.go           # Mounts WebUIHandler() at PathPrefix("/")
├── webui.go           # Embedded UI handler (SPA pattern)
└── webui/
    ├── .gitignore     # Ignores dist/
    └── dist/          # Copied from web/dist during build (NOT committed)
```

**Key Files:**
- `internal/shell/api/webui.go` - Handler using `//go:embed all:webui/dist`
- `web/vite.config.ts` - Base path set to `/` (served at root)

**Build Process:**
1. `cd web && npm install && npm run build`
2. `cp -r dist ../internal/shell/api/webui/`
3. `go build ./cmd/hoster` (embeds webui/dist via //go:embed)

**Local Development:**
- Frontend: `cd web && npm run dev` (Vite dev server on :3000 or :5173)
- Backend: `make run` (Hoster on :8080)
- Vite proxies /api to backend

---

## Production Deployment (emptychair.dev)

### Architecture

```
                    ┌─────────────────────────────────────┐
                    │      APIGate (Direct TLS)           │
    Internet ──────►│  Port 80  → HTTP redirect           │
                    │  Port 443 → TLS termination         │
                    └─────────────────────────────────────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    ▼                 ▼                 ▼
              Hoster API        App Proxy         Portal
            localhost:8080    localhost:9091    (built-in)
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| APIGate | 80, 443 | TLS termination, auth, billing, routing |
| Hoster | 8080 | Deployment API |
| App Proxy | 9091 | Routes `*.apps.emptychair.dev` to deployments |

### Deployment Files (gitignored - `deploy/local/`)

| File | Purpose |
|------|---------|
| `Makefile` | Management commands (`make status`, `make logs`, etc.) |
| `emptychair.env` | Environment variables |
| `infrastructure.md` | AWS/DNS details (sensitive) |

### Management Commands

```bash
cd deploy/local
make status         # Check service status
make logs           # Tail all logs
make logs-apigate   # Tail APIGate logs
make logs-hoster    # Tail Hoster logs
make restart        # Restart all services
make shell          # SSH into server
make settings       # Show APIGate settings
make errors         # Show recent errors
```

### CI/CD

**Workflows:**
- `.github/workflows/ci.yml` - Runs on push to main and PRs (test, build, vet)
- `.github/workflows/release.yml` - Runs on version tags (v*), creates GitHub releases

**Build Process:**
1. Build frontend: `cd web && npm install && npm run build`
2. Copy to embed dir: `cp -r dist ../internal/shell/api/webui/`
3. Build Go binary with embedded frontend

**Current Issue (January 2026):**
- CI failing with `@rollup/rollup-linux-x64-gnu` module error
- Fix attempt: `rm -rf node_modules package-lock.json` before `npm install`
- See `specs/SESSION-HANDOFF.md` for detailed troubleshooting steps

### Deployment Process

**Via Makefile (RECOMMENDED):**
```bash
cd deploy/local
make deploy-release                    # Deploy latest GitHub release
make deploy-release VERSION=v0.2.0     # Deploy specific version
```

**Manual (if needed):**
1. Tag a release: `git tag v0.2.0 && git push origin v0.2.0`
2. Wait for GitHub release to be created
3. Deploy: `make deploy-release VERSION=v0.2.0`

---

## Files to Read First (New Session Checklist)

1. **This file** (`CLAUDE.md`) - You're reading it
2. `specs/README.md` - How to write specs
3. `specs/SESSION-HANDOFF.md` - Session onboarding protocol
4. `specs/decisions/ADR-001-docker-direct.md` - Docker architecture
5. `specs/decisions/ADR-002-values-as-boundaries.md` - Code organization
6. `specs/decisions/ADR-003-jsonapi-api2go.md` - JSON:API standard
7. `specs/decisions/ADR-004-reflective-openapi.md` - OpenAPI generation
8. `specs/decisions/ADR-005-apigate-integration.md` - Auth/billing
9. `specs/decisions/ADR-006-frontend-architecture.md` - React frontend
10. `specs/decisions/ADR-007-uiux-guidelines.md` - UI/UX patterns
11. `specs/domain/template.md` - Template entity spec
12. `specs/domain/deployment.md` - Deployment entity spec

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

## Summary for New Session

If you're starting a new session with no memory:

1. **Read this entire file first**
2. **Read `specs/SESSION-HANDOFF.md`** for the complete onboarding protocol
3. You're building a deployment marketplace platform
4. Follow STC: Spec → Test → Code (NEVER skip)
5. Architecture: Pure core (`internal/core/`), thin shell (`internal/shell/`)
6. **MVP is COMPLETE** - Core deployment loop is working
7. **Post-MVP specs are COMPLETE** - See ADR-003 through ADR-006, F008-F013
8. **Next: Implementation Phase 0** - Migrate to api2go + OpenAPI
9. Key libraries: api2go (JSON:API), gorilla/mux (router), kin-openapi (OpenAPI)
10. Run `make test` to verify everything works
11. Check the plan file at `.claude/plans/merry-baking-rain.md` for detailed phases

**When in doubt, read the specs in `specs/` directory.**

---

## Failure Mode Reference

What goes wrong if key decisions are forgotten:

| Forgotten | Failure Mode | Impact |
|-----------|--------------|--------|
| STC methodology | Code without specs/tests | Future changes break things |
| Values as Boundaries | I/O in core | Tests need mocks, tech debt |
| State machine transitions | Invalid status changes | Runtime bugs, data corruption |
| Library choices | Using different libs | Inconsistency, conflicts |
| "NOT Supported" items | Building wrong features | Wasted effort, complexity |
| Directory structure | Files in wrong places | Confusion, broken imports |
| Testing strategy | Adding mocks to core | Slow tests, false confidence |

**The cost of forgetting increases with time. Document decisions immediately.**

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

Hoster uses APIGate for authentication and billing (see ADR-005).

- **APIGate Repository**: https://github.com/artpar/apigate
- **APIGate Issues**: https://github.com/artpar/apigate/issues
- **APIGate Wiki**: https://github.com/artpar/apigate/wiki

When encountering APIGate-related issues during development or testing, report them at the issues link above. The `gh` CLI is available and logged in for creating issues.


---

## Local E2E Testing Environment

### Current Setup (January 22, 2026)

**Location:** `/tmp/hoster-e2e-test/`

**Architecture:**
```
Browser → localhost:8082 (APIGate) → Hoster/App Proxy
    ├── Frontend Route    (/* priority 10, auth_required=0)
    ├── API Route         (/api/* priority 50, auth_required=0)
    └── App Proxy Route   (*.apps.localhost/* priority 100, auth_required=0)
```

**Services:**
- APIGate: localhost:8082 (single entry point)
- Hoster: localhost:8080 (API + embedded frontend)
- App Proxy: localhost:9091 (deployment routing)

**Important:** All access MUST go through APIGate (localhost:8082). Never access Hoster directly on localhost:8080.

### Starting the Environment

```bash
# Terminal 1: Start APIGate
cd /tmp/hoster-e2e-test
apigate serve --config apigate.yaml > apigate.log 2>&1 &

# Terminal 2: Start Hoster (auto-registration disabled)
cd /Users/artpar/workspace/code/hoster
HOSTER_DATABASE_DSN=/tmp/hoster-e2e-test/hoster.db \
HOSTER_APIGATE_AUTO_REGISTER=false \
./bin/hoster > /tmp/hoster-e2e-test/hoster.log 2>&1 &
```

### Checking Status

```bash
# Check running services
ps aux | grep -E "(apigate|hoster)" | grep -v grep
lsof -i :8082  # APIGate
lsof -i :8080  # Hoster
lsof -i :9091  # App Proxy

# View logs
tail -f /tmp/hoster-e2e-test/apigate.log
tail -f /tmp/hoster-e2e-test/hoster.log

# Check routes configuration
sqlite3 /tmp/hoster-e2e-test/apigate.db "SELECT name, path_pattern, host_pattern, priority, auth_required FROM routes ORDER BY priority DESC;"
```

### Testing E2E Flow

1. **Access Frontend:** http://localhost:8082/
2. **Browse Marketplace:** http://localhost:8082/marketplace
3. **Sign in via dev auth:** Use any email/password
4. **Deploy template:** Select a template and deploy
5. **Monitor deployment:** View Events, Stats, Logs tabs
6. **Access deployed app:** http://{deployment-name}.apps.localhost:8082/

### Database Files

- **Hoster:** `/tmp/hoster-e2e-test/hoster.db` - Templates, deployments, events
- **APIGate:** `/tmp/hoster-e2e-test/apigate.db` - Routes, upstreams, users

### Routes Configuration (Manual)

Routes are manually configured in APIGate database:

```sql
-- Frontend route (priority 10 - catches /* after higher priority routes)
UPDATE routes SET auth_required=0 WHERE name='hoster-frontend';

-- API route (priority 50 - catches /api/* before frontend)
UPDATE routes SET auth_required=0 WHERE name='hoster-api';

-- App proxy route (priority 100 - highest, catches *.apps.localhost/*)
UPDATE routes SET auth_required=0 WHERE name='hoster-app-proxy';
```

### Known Limitations

1. **Auto-registration disabled:** Admin endpoints (`/admin/*`) are caught by frontend route
2. **Auth disabled for testing:** All routes have `auth_required=0`
3. **Port in URLs:** App proxy requires port in URL for local testing (`:8082`)

### Troubleshooting

**Problem:** Frontend shows 404
- **Check:** APIGate is running on 8082
- **Check:** Hoster is running on 8080
- **Check:** Routes are configured correctly

**Problem:** API returns 401
- **Check:** Route has `auth_required=0`
- **Fix:** `sqlite3 /tmp/hoster-e2e-test/apigate.db "UPDATE routes SET auth_required=0 WHERE name='hoster-api';"`

**Problem:** Can't access deployed app
- **Check:** App Proxy is running on 9091
- **Check:** Deployment has domain assigned
- **Check:** Using correct URL format: `http://{name}.apps.localhost:8082/`
