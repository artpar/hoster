# F006: Deployment Planning

## User Story

As a **deployment orchestrator**, I need pure functions to transform compose specifications into Docker execution plans, so that the imperative shell can execute commands without containing any business logic.

## Overview

This feature provides pure functions that plan deployments by generating resource names, ordering services by dependencies, substituting variables, converting port bindings, and building container specifications. All functions live in `internal/core/deployment/` and have **no I/O** - they take values in and return values out, compliant with ADR-002 "Values as Boundaries".

## Acceptance Criteria

- [ ] Generate consistent network/volume/container names with `hoster_` prefix
- [ ] Topologically sort services by `depends_on` using Kahn's algorithm
- [ ] Substitute `${VAR}` and `${VAR:-default}` placeholders in strings
- [ ] Convert port bindings to domain port mappings with default protocol
- [ ] Build complete container plans from compose services and deployment data
- [ ] All functions are pure (no I/O, no side effects)
- [ ] 100% test coverage

## Functions

### Naming Functions

Generate consistent resource names for Docker objects.

```go
// NetworkName generates a network name for a deployment.
// Pattern: hoster_{deploymentID}
func NetworkName(deploymentID string) string

// VolumeName generates a volume name for a deployment.
// Pattern: hoster_{deploymentID}_{volumeName}
func VolumeName(deploymentID, volumeName string) string

// ContainerName generates a container name for a service.
// Pattern: hoster_{deploymentID}_{serviceName}
func ContainerName(deploymentID, serviceName string) string
```

| Function | Input | Output | Example |
|----------|-------|--------|---------|
| `NetworkName` | `"abc123"` | `"hoster_abc123"` | Network name |
| `VolumeName` | `"abc123", "data"` | `"hoster_abc123_data"` | Volume name |
| `ContainerName` | `"abc123", "web"` | `"hoster_abc123_web"` | Container name |

### Service Ordering

```go
// TopologicalSort sorts services by their dependencies using Kahn's algorithm.
// Services with no dependencies come first.
// If a cycle exists, remaining services are appended (cycle detection is done at parse time).
func TopologicalSort(services []compose.Service) []compose.Service
```

| Input | Output Description |
|-------|-------------------|
| Services with no dependencies | Original order preserved |
| Linear chain (a→b→c) | c, b, a (dependencies first) |
| Diamond (a→[b,c]→d) | d first, a last |
| Cycle (a↔b) | Both services appended (fallback) |

### Variable Substitution

```go
// SubstituteVariables replaces ${VAR} and ${VAR:-default} placeholders.
// If variable not found and no default: returns original placeholder.
// If variable not found but default exists: returns default value.
func SubstituteVariables(value string, variables map[string]string) string
```

| Input | Variables | Output |
|-------|-----------|--------|
| `"${DB_HOST}"` | `{"DB_HOST": "localhost"}` | `"localhost"` |
| `"${PORT:-8080}"` | `{}` | `"8080"` |
| `"${PORT:-8080}"` | `{"PORT": "3000"}` | `"3000"` |
| `"${MISSING}"` | `{}` | `"${MISSING}"` |
| `"postgres://${HOST}:${PORT}"` | `{"HOST": "db", "PORT": "5432"}` | `"postgres://db:5432"` |

### Port Conversion

```go
// PortBinding represents a Docker port binding.
type PortBinding struct {
    ContainerPort int
    HostPort      int
    Protocol      string
    HostIP        string
}

// ConvertPorts converts port bindings to domain port mappings.
// Default protocol is "tcp" if empty.
func ConvertPorts(ports []PortBinding) []domain.PortMapping
```

| Input Protocol | Output Protocol |
|----------------|-----------------|
| `""` | `"tcp"` |
| `"tcp"` | `"tcp"` |
| `"udp"` | `"udp"` |

### Container Plan Building

```go
// BuildContainerPlanParams contains all inputs for building a container plan.
type BuildContainerPlanParams struct {
    DeploymentID string
    TemplateID   string
    ServiceName  string
    Service      compose.Service
    Variables    map[string]string
    NetworkName  string
    Volumes      []compose.Volume
}

// ContainerPlan represents a planned container configuration.
type ContainerPlan struct {
    Name          string
    Image         string
    Command       []string
    Entrypoint    []string
    Env           map[string]string
    Labels        map[string]string
    Ports         []PortPlan
    Volumes       []VolumePlan
    Networks      []string
    RestartPolicy RestartPolicyPlan
    Resources     ResourcePlan
    HealthCheck   *HealthCheckPlan
}

// BuildContainerPlan builds a ContainerPlan from compose service and deployment data.
func BuildContainerPlan(params BuildContainerPlanParams) ContainerPlan
```

#### Building Rules

1. **Name**: `ContainerName(deploymentID, serviceName)`
2. **Image**: Direct from service
3. **Command/Entrypoint**: Direct from service
4. **Environment**: Merge service env with `SubstituteVariables` applied
5. **Labels**: Hoster labels + service labels
   - `com.hoster.managed=true`
   - `com.hoster.deployment={deploymentID}`
   - `com.hoster.template={templateID}`
   - `com.hoster.service={serviceName}`
6. **Networks**: Single network (the deployment network)
7. **Ports**: Direct mapping from service ports
8. **Volumes**: Named volumes prefixed with deployment ID
9. **HealthCheck**: Durations parsed from string (e.g., "30s")
10. **RestartPolicy**: Mapped from compose restart policy
11. **Resources**: Direct from service (CPU/memory limits)

#### Restart Policy Mapping

| Compose | Docker |
|---------|--------|
| `always` | `"always"` |
| `on-failure` | `"on-failure"` |
| `unless-stopped` | `"unless-stopped"` |
| `no` (or empty) | `"no"` |

## Types

### PortPlan

```go
type PortPlan struct {
    ContainerPort int
    HostPort      int
    Protocol      string
    HostIP        string
}
```

### VolumePlan

```go
type VolumePlan struct {
    Source   string
    Target   string
    ReadOnly bool
}
```

### RestartPolicyPlan

```go
type RestartPolicyPlan struct {
    Name              string
    MaximumRetryCount int
}
```

### ResourcePlan

```go
type ResourcePlan struct {
    CPULimit    float64
    MemoryLimit int64
}
```

### HealthCheckPlan

```go
type HealthCheckPlan struct {
    Test        []string
    Interval    time.Duration
    Timeout     time.Duration
    Retries     int
    StartPeriod time.Duration
}
```

## Edge Cases

### Naming
- Empty deployment ID → Returns `"hoster_"` (valid but shouldn't happen)
- UUID deployment ID → Works correctly with full UUID

### Topological Sort
- Empty services → Returns empty slice
- Single service → Returns single service
- All independent → Original order preserved
- Circular dependency → Fallback: append remaining services

### Variable Substitution
- No placeholders → Returns original string unchanged
- Empty variables map → Placeholders unchanged or defaults used
- Empty default (`${VAR:-}`) → Returns empty string
- Nested placeholders → Not supported (single pass only)

### Port Conversion
- Empty ports → Returns empty slice (not nil)
- Zero container port → Passed through (validation at compose level)

### Container Plan
- No environment → Empty map (not nil)
- No ports → Empty slice (not nil)
- No health check → Nil health check
- No resources → Zero values (uses Docker defaults)

## Not Supported

| Feature | Reason |
|---------|--------|
| Nested variable substitution | Single-pass replacement only |
| Dynamic port allocation | Handled by Docker at runtime |
| Build context | Image must be pre-built |
| Secrets injection | Out of scope for prototype |

## Dependencies

- `internal/core/compose` - For `compose.Service`, `compose.Volume` types
- `internal/core/domain` - For `domain.PortMapping` type

## Implementation Notes

1. All functions must be pure (no I/O, no side effects)
2. Use package-level functions (not methods)
3. Follow existing patterns in `internal/core/domain/`
4. Use testify for assertions
5. No mocks needed - pure value transformations

## Tests

### Test File: `internal/core/deployment/naming_test.go`

| Test | Description |
|------|-------------|
| `TestNetworkName_Simple` | Basic deployment ID |
| `TestNetworkName_UUID` | Full UUID format |
| `TestVolumeName_Simple` | Basic volume name |
| `TestVolumeName_WithUnderscore` | Volume name with underscore |
| `TestContainerName_Simple` | Basic service name |
| `TestContainerName_DBService` | Database service name |

### Test File: `internal/core/deployment/ordering_test.go`

| Test | Description |
|------|-------------|
| `TestTopologicalSort_Empty` | Empty service list |
| `TestTopologicalSort_NoDependencies` | Independent services |
| `TestTopologicalSort_LinearDependencies` | Chain: a→b→c |
| `TestTopologicalSort_DiamondDependencies` | Diamond pattern |
| `TestTopologicalSort_CycleFallback` | Cycle detection fallback |

### Test File: `internal/core/deployment/variables_test.go`

| Test | Description |
|------|-------------|
| `TestSubstituteVariables_Simple` | Single variable |
| `TestSubstituteVariables_WithDefault_Found` | Default with variable present |
| `TestSubstituteVariables_WithDefault_NotFound` | Default with variable missing |
| `TestSubstituteVariables_NotFound_NoDefault` | Missing variable, no default |
| `TestSubstituteVariables_Multiple` | Multiple variables in string |
| `TestSubstituteVariables_NoPlaceholders` | Plain text (no substitution) |
| `TestSubstituteVariables_EmptyDefault` | Empty default value |

### Test File: `internal/core/deployment/ports_test.go`

| Test | Description |
|------|-------------|
| `TestConvertPorts_Empty` | Empty port list |
| `TestConvertPorts_SinglePort` | Single port with protocol |
| `TestConvertPorts_DefaultProtocol` | Empty protocol → tcp |
| `TestConvertPorts_UDP` | UDP protocol preserved |
| `TestConvertPorts_Multiple` | Multiple ports |

### Test File: `internal/core/deployment/container_test.go`

| Test | Description |
|------|-------------|
| `TestBuildContainerPlan_BasicService` | Minimal service |
| `TestBuildContainerPlan_WithEnvironment` | Environment variable substitution |
| `TestBuildContainerPlan_WithVolumes` | Named volume prefixing |
| `TestBuildContainerPlan_WithHealthCheck` | Health check duration parsing |
| `TestBuildContainerPlan_RestartPolicies` | All restart policy mappings |
| `TestBuildContainerPlan_WithResources` | CPU/memory limits |
| `TestBuildContainerPlan_WithPorts` | Port mapping |
| `TestBuildContainerPlan_Labels` | Hoster + service labels merge |

**Total: ~30 tests**
