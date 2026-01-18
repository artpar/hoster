# Hoster

Modern deployment marketplace platform - your own Railway/Render/Heroku with a template marketplace.

## Vision

A multi-tenant SaaS platform where:
- **Package Creators** define deployment templates (docker-compose + config + pricing)
- **Customers** browse a marketplace, one-click deploy instances onto YOUR infrastructure
- **You** run the platform on your VPS pool, earn per-deployment revenue

## Status: MVP Complete

The core deployment loop is fully functional:

| Feature | Status |
|---------|--------|
| Template CRUD + publish | ✅ |
| Deployment from template | ✅ |
| Auto domain generation | ✅ |
| Traefik routing labels | ✅ |
| Start/stop/restart/delete | ✅ |
| State machine transitions | ✅ |
| SQLite persistence | ✅ |
| HTTP REST API | ✅ |

## Quick Start

```bash
# Install dependencies
make deps

# Run tests (427 tests)
make test

# Build and run
make build
./hoster

# Or with environment config
HOSTER_SERVER_PORT=8080 \
HOSTER_DATABASE_PATH=./data/hoster.db \
HOSTER_DOMAIN_BASE_DOMAIN=apps.localhost \
./hoster
```

## API Usage

```bash
# Create a template
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Simple Nginx",
    "version": "1.0.0",
    "description": "A simple nginx web server",
    "compose_spec": "version: \"3.8\"\nservices:\n  web:\n    image: nginx:alpine\n    ports:\n      - \"80:80\"",
    "creator_id": "user-123",
    "price_cents": 0
  }'

# Publish the template
curl -X POST http://localhost:8080/api/v1/templates/{id}/publish

# Create a deployment
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Nginx Site",
    "template_id": "{template_id}",
    "customer_id": "customer-456"
  }'

# Start the deployment
curl -X POST http://localhost:8080/api/v1/deployments/{id}/start

# Stop the deployment
curl -X POST http://localhost:8080/api/v1/deployments/{id}/stop

# Delete the deployment
curl -X DELETE http://localhost:8080/api/v1/deployments/{id}
```

## Project Structure

```
hoster/
├── cmd/hoster/                 # Entry point
├── specs/                      # SOURCE OF TRUTH (STC methodology)
│   ├── domain/                 # Domain model specs
│   ├── features/               # Feature specs
│   └── decisions/              # Architecture Decision Records
├── internal/
│   ├── core/                   # FUNCTIONAL CORE (pure, no I/O)
│   │   ├── domain/             # Domain types and validation
│   │   ├── compose/            # Compose parsing
│   │   ├── deployment/         # Deployment planning logic
│   │   ├── traefik/            # Traefik label generation
│   │   └── validation/         # Input validation
│   └── shell/                  # IMPERATIVE SHELL (I/O)
│       ├── api/                # HTTP handlers (chi router)
│       ├── docker/             # Docker SDK wrapper
│       └── store/              # SQLite storage (sqlx)
├── tests/e2e/                  # End-to-end tests
└── examples/                   # Sample templates
```

## Development Methodology

We follow **STC (Spec → Test → Code)**:

1. **SPEC** - Write specification first (`specs/`)
2. **TEST** - Write tests that verify the spec
3. **CODE** - Implement code that passes tests

## Architecture

We use **Values as Boundaries** (ADR-002):

- **Functional Core** (`internal/core/`): Pure functions, no I/O, trivially testable
- **Imperative Shell** (`internal/shell/`): Thin I/O layer, integration tested

## Testing

```bash
make test             # All tests (427)
make test-unit        # Core logic tests (fast, no I/O)
make test-integration # Shell tests (Docker, DB)
make test-e2e         # Full system tests
make coverage         # Coverage report
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HOSTER_SERVER_PORT` | 8080 | HTTP server port |
| `HOSTER_DATABASE_PATH` | ./hoster.db | SQLite database path |
| `HOSTER_DOMAIN_BASE_DOMAIN` | apps.localhost | Base domain for auto-generated domains |
| `HOSTER_DOCKER_HOST` | unix:///var/run/docker.sock | Docker host |

## Key Decisions

- [ADR-001: Docker Direct](specs/decisions/ADR-001-docker-direct.md) - No orchestration layer
- [ADR-002: Values as Boundaries](specs/decisions/ADR-002-values-as-boundaries.md) - Architecture pattern

## Roadmap

- [ ] Frontend UI
- [ ] User authentication/authorization
- [ ] Billing/pricing integration
- [ ] Multi-node deployment support
- [ ] Monitoring/logging dashboard

## License

MIT
