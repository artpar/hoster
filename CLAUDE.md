# CLAUDE.md - Project Memory for Hoster

> **CRITICAL**: Read this file completely before making ANY changes to this project.
> This file is the source of truth for project decisions, methodology, and architecture.

## Project Identity

**Hoster** is a modern deployment marketplace platform - like Railway/Render/Heroku but self-hosted with a template marketplace.

**Vision**: Package creators define deployment templates (docker-compose + config + pricing), customers one-click deploy instances onto YOUR VPS infrastructure.

**Status**: Prototype phase - validating core deployment loop.

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

### Test Counts (January 18, 2026)
| Suite | Count | Status |
|-------|-------|--------|
| Unit (core/) | ~180 | PASS |
| Integration (shell/) | ~150 | PASS |
| E2E | ~100 | PASS |
| **Total** | **427** | **ALL PASS** |

### MVP STATUS: ✅ COMPLETE

The core deployment loop is fully functional:
1. ✅ Creator creates template with docker-compose
2. ✅ Creator publishes template
3. ✅ Customer deploys from published template
4. ✅ Deployment gets auto-generated domain
5. ✅ Deployment gets Traefik labels for external routing
6. ✅ Customer can start/stop/restart deployments
7. ✅ Customer can delete deployments

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
