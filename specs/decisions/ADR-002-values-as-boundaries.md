# ADR-002: Values as Boundaries Architecture

## Status
Accepted

## Context

We want code that is:
- Easy to test without mocks
- Easy to refactor without breaking things
- Easy to reason about
- Minimal technical debt

The "Boundaries" pattern (Gary Bernhardt, 2012) provides a structure that achieves these goals by separating pure logic from I/O.

## Decision

We will structure the codebase into two layers:

### Functional Core (`internal/core/`)
Pure functions that:
- Take values in (structs, primitives)
- Return values out (structs, errors)
- Have NO side effects (no I/O, no state mutation)
- Are trivially testable with simple assertions

Examples:
```go
// Pure: takes values, returns values
func ParseComposeSpec(yaml []byte) (*ComposeSpec, error)
func ValidateTemplate(t Template) []ValidationError
func GenerateTraefikLabels(d Deployment) map[string]string
func CalculateResourceRequirements(spec ComposeSpec) Resources
```

### Imperative Shell (`internal/shell/`)
Thin layer that:
- Handles all I/O (HTTP, Docker, Database, Files)
- Reads data → passes values to core
- Core returns decisions → shell executes them
- Is integration tested with real dependencies

Examples:
```go
// Shell: handles I/O, delegates logic to core
func (h *Handler) CreateDeployment(w http.ResponseWriter, r *http.Request) {
    // 1. Read input (I/O)
    var req CreateDeploymentRequest
    json.NewDecoder(r.Body).Decode(&req)

    // 2. Call core logic (pure)
    deployment, errs := core.ValidateAndCreateDeployment(req)
    if len(errs) > 0 {
        respondWithErrors(w, errs)
        return
    }

    // 3. Execute decision (I/O)
    err := h.docker.CreateContainers(deployment)
    if err != nil {
        respondWithError(w, err)
        return
    }

    // 4. Persist (I/O)
    h.store.SaveDeployment(deployment)
    respondWithJSON(w, deployment)
}
```

## Directory Structure

```
internal/
├── core/                    # FUNCTIONAL CORE (pure, no I/O)
│   ├── compose/             # Compose parsing
│   ├── deployment/          # Deployment logic
│   ├── domain/              # Domain types
│   └── traefik/             # Traefik config generation
│
└── shell/                   # IMPERATIVE SHELL (I/O)
    ├── api/                 # HTTP handlers
    ├── docker/              # Docker client
    └── store/               # Database
```

## Consequences

### Positive
- **100% unit testable core**: No mocks needed, just values in/out
- **Fast tests**: Core tests don't touch I/O, run in milliseconds
- **Easy refactoring**: Change core logic, shell unchanged
- **Clear boundaries**: Know exactly where I/O happens
- **Minimal tech debt**: Pure functions don't accumulate cruft
- **Easy to reason about**: Function does what signature says

### Negative
- **Some boilerplate**: Shell must translate between I/O and values
- **Discipline required**: Must resist putting logic in shell
- **Learning curve**: Pattern may be unfamiliar initially

### Neutral
- Shell tests are integration tests (slower, but fewer of them)
- Some duplication between shell input types and core domain types

## Testing Strategy

| Layer | Test Type | Dependencies | Speed |
|-------|-----------|--------------|-------|
| Core | Unit tests | None | Fast (<1s) |
| Shell | Integration tests | Docker, SQLite | Slow (seconds) |
| E2E | System tests | Full stack | Very slow |

```go
// Core test: pure, no setup
func TestValidateTemplate(t *testing.T) {
    template := Template{Name: ""}
    errs := ValidateTemplate(template)
    assert.Contains(t, errs, ErrNameRequired)
}

// Shell test: needs real Docker
func TestDockerClient_CreateContainer(t *testing.T) {
    client := NewDockerClient()
    id, err := client.CreateContainer(spec)
    assert.NoError(t, err)
    defer client.RemoveContainer(id)
}
```

## References
- Gary Bernhardt, "Boundaries" (2012): https://www.destroyallsoftware.com/talks/boundaries
- Functional Core, Imperative Shell pattern
