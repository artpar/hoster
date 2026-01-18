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
│   ├── domain/                 # Entity specifications
│   │   ├── template.md         # Template entity (IMPLEMENTED)
│   │   └── deployment.md       # Deployment entity (IMPLEMENTED)
│   ├── features/               # Feature specifications
│   │   └── F###-{name}.md      # (TODO)
│   └── decisions/              # Architecture Decision Records
│       ├── ADR-001-docker-direct.md       # (IMPLEMENTED)
│       └── ADR-002-values-as-boundaries.md # (IMPLEMENTED)
├── internal/
│   ├── core/                   # FUNCTIONAL CORE (no I/O)
│   │   ├── domain/             # Domain types + validation (IMPLEMENTED)
│   │   ├── compose/            # Compose parsing (TODO)
│   │   ├── deployment/         # Deployment logic (TODO)
│   │   └── traefik/            # Traefik config generation (TODO)
│   └── shell/                  # IMPERATIVE SHELL (I/O)
│       ├── api/                # HTTP handlers (TODO)
│       ├── docker/             # Docker SDK wrapper (TODO)
│       └── store/              # Database layer (TODO)
├── tests/
│   ├── e2e/                    # End-to-end tests (TODO)
│   └── fixtures/               # Test data (TODO)
├── examples/                   # Sample templates (TODO)
├── cmd/hoster/                 # Entry point (TODO)
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

### ADR-003: SQLite for Prototype

- **Decision**: Use SQLite now, migrate to PostgreSQL later
- **Rationale**: Fast start, easy to develop
- **Implication**: Use `sqlx` with SQLite driver
- **DO NOT**: Start with PostgreSQL yet

### Library Choices (USE THESE, NOT ALTERNATIVES)

| Purpose | Library | DO NOT USE |
|---------|---------|------------|
| Docker SDK | `github.com/docker/docker/client` | Other Docker libs |
| Compose parsing | `github.com/compose-spec/compose-go/v2` | Custom parser |
| HTTP router | `github.com/go-chi/chi/v5` | gin, echo, mux |
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
| Unit (core/) | 117 | PASS |
| Integration (shell/) | 123 | PASS |
| E2E | 22 | PASS |
| **Total** | **262** | **ALL PASS** |

### TODO (Next Steps) - ARCHITECTURAL REFACTOR

**CRITICAL: ADR-002 is being violated.** Pure logic exists in shell that should be in core.

#### 1. Create `internal/core/deployment/` package (HIGH PRIORITY)
Move these pure functions from `shell/docker/orchestrator.go`:
- [ ] `naming.go` - networkName(), volumeName(), containerName()
- [ ] `ordering.go` - topologicalSort() for service dependencies
- [ ] `container.go` - buildContainerSpec() mapping
- [ ] `ports.go` - convertPorts() transformation
- [ ] `planner.go` - DetermineStartPath() for state transitions

#### 2. Create `internal/core/traefik/` package (HIGH PRIORITY)
- [ ] `labels.go` - GenerateTraefikLabels(deployment) map[string]string
- Without this, deployments have NO external routing!

#### 3. Refactor `shell/docker/orchestrator.go`
- [ ] Import from core/deployment/
- [ ] Keep ONLY I/O operations (actual Docker calls)
- [ ] Should be ~50% smaller after refactor

#### 4. Refactor `shell/api/handler.go`
- [ ] Move state transition logic to core
- [ ] Handler should just call core and execute decisions

#### 5. Missing spec features
- [ ] Domain auto-generation (subdomain on deployment start)
- [ ] Resources calculation from compose spec

### Blocked
- Nothing blocked, but architectural debt is accumulating

### Architecture Violation Details

**orchestrator.go has these PURE functions that should be in core:**
```
topologicalSort()      → core/deployment/ordering.go
substituteVariables()  → core/compose/variables.go
buildContainerSpec()   → core/deployment/container.go
convertPorts()         → core/deployment/ports.go
networkName()          → core/deployment/naming.go
volumeName()           → core/deployment/naming.go
containerName()        → core/deployment/naming.go
```

Per ADR-002: "Functional Core has NO side effects" - these functions qualify but are in wrong place.

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
3. `specs/decisions/ADR-001-docker-direct.md` - Architecture decision
4. `specs/decisions/ADR-002-values-as-boundaries.md` - Code organization
5. `specs/domain/template.md` - Template entity spec
6. `specs/domain/deployment.md` - Deployment entity spec
7. `internal/core/domain/template.go` - See implementation pattern
8. `internal/core/domain/deployment.go` - See state machine

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
6. Domain model has Template and Deployment entities (implemented, tested)
7. Next: Implement compose parsing feature (spec first!)
8. Use the specific libraries listed above
9. Run `make test` to verify everything works
10. Check Agile MCP for task tracking

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
