# Hoster

Modern deployment marketplace platform - your own Railway/Render/Heroku with a template marketplace.

## Vision

A multi-tenant SaaS platform where:
- **Package Creators** define deployment templates (docker-compose + config + pricing)
- **Customers** browse a marketplace, one-click deploy instances onto YOUR infrastructure
- **You** run the platform on your VPS pool, earn per-deployment revenue

## Quick Start

```bash
# Install dependencies
make deps

# Run tests
make test

# Run the server (when implemented)
make run
```

## Project Structure

```
hoster/
├── specs/                      # SOURCE OF TRUTH (STC methodology)
│   ├── domain/                 # Domain model specs
│   ├── features/               # Feature specs
│   └── decisions/              # Architecture Decision Records
├── internal/
│   ├── core/                   # FUNCTIONAL CORE (pure, no I/O)
│   │   └── domain/             # Domain types and validation
│   └── shell/                  # IMPERATIVE SHELL (I/O)
│       ├── api/                # HTTP handlers
│       ├── docker/             # Docker SDK wrapper
│       └── store/              # Database layer
├── tests/e2e/                  # End-to-end tests
└── examples/                   # Sample templates
```

## Development Methodology

We follow **STC (Spec → Test → Code)**:

1. **SPEC** - Write specification first (`specs/`)
2. **TEST** - Write tests that verify the spec
3. **CODE** - Implement code that passes tests

The trifecta (Spec ↔ Test ↔ Code) must always be in sync.

## Architecture

We use **Values as Boundaries** (Gary Bernhardt's "Boundaries" pattern):

- **Functional Core** (`internal/core/`): Pure functions, no I/O, trivially testable
- **Imperative Shell** (`internal/shell/`): Thin I/O layer, integration tested

See [ADR-002](specs/decisions/ADR-002-values-as-boundaries.md) for details.

## Testing

```bash
make test-unit        # Core logic tests (fast, no I/O)
make test-integration # Shell tests (Docker, DB)
make test-e2e         # Full system tests
make coverage         # Coverage report (>90% for core/)
```

## Key Decisions

- [ADR-001: Docker Direct](specs/decisions/ADR-001-docker-direct.md) - No orchestration layer
- [ADR-002: Values as Boundaries](specs/decisions/ADR-002-values-as-boundaries.md) - Architecture pattern

## License

MIT
