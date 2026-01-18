# F001: Parse Docker Compose Specification

## User Story

As a **template creator**, I want to provide a Docker Compose YAML file so that the system can understand what services, networks, and volumes to deploy.

## Overview

This feature parses Docker Compose YAML content into a structured Hoster-specific format. It extracts services, networks, volumes, environment variable placeholders, and resource requirements. This is a **pure function** with no I/O, living in `internal/core/compose/`.

## Acceptance Criteria

- [ ] Parse valid Docker Compose YAML into `ParsedSpec` struct
- [ ] Extract all services with image, ports, volumes, environment, depends_on
- [ ] Extract top-level networks and volumes
- [ ] Extract environment variable placeholders (`${VAR_NAME}`)
- [ ] Calculate resource requirements with defaults (0.5 CPU, 256MB per service)
- [ ] Detect circular dependencies in `depends_on`
- [ ] Return typed errors for all failure modes
- [ ] 100% test coverage including all error paths

## Inputs

| Input | Type | Required | Description |
|-------|------|----------|-------------|
| yamlContent | string | Yes | Raw Docker Compose YAML content |

### Valid Input Example

```yaml
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    environment:
      API_URL: http://api:8080
    depends_on:
      - api

  api:
    image: myapp:1.0
    environment:
      DB_PASSWORD: ${DB_PASSWORD}
    volumes:
      - data:/app/data

volumes:
  data:
```

## Outputs

### ParsedSpec

```go
type ParsedSpec struct {
    Services []Service
    Networks []Network
    Volumes  []Volume
}
```

### Service

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Service name from compose key |
| Image | string | Docker image reference |
| Build | *BuildConfig | Build context (optional) |
| Command | []string | Override command |
| Entrypoint | []string | Override entrypoint |
| Ports | []Port | Port mappings |
| Environment | map[string]string | Environment variables |
| Volumes | []VolumeMount | Volume mounts |
| Networks | []string | Network names |
| DependsOn | []string | Service dependencies |
| Restart | RestartPolicy | Restart policy |
| Resources | ServiceResources | CPU/memory limits |
| HealthCheck | *HealthCheck | Health check config |
| Labels | map[string]string | Container labels |

### Port

| Field | Type | Description |
|-------|------|-------------|
| Target | uint32 | Container port |
| Published | uint32 | Host port (0 = dynamic) |
| Protocol | string | "tcp" or "udp" |
| HostIP | string | Bind IP (optional) |

### VolumeMount

| Field | Type | Description |
|-------|------|-------------|
| Type | VolumeMountType | "bind", "volume", or "tmpfs" |
| Source | string | Host path or volume name |
| Target | string | Container path |
| ReadOnly | bool | Read-only mount |

### ServiceResources

| Field | Type | Description |
|-------|------|-------------|
| CPULimit | float64 | CPU cores limit |
| CPUReservation | float64 | CPU cores reserved |
| MemoryLimit | int64 | Memory limit in bytes |
| MemoryReservation | int64 | Memory reserved in bytes |

### Network

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Network name |
| Driver | string | Network driver |
| External | bool | External network |
| Internal | bool | Internal-only network |

### Volume

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Volume name |
| Driver | string | Volume driver |
| External | bool | External volume |

## Functions

### Primary

```go
// ParseComposeSpec parses Docker Compose YAML into a ParsedSpec.
// Returns error for invalid input.
func ParseComposeSpec(yamlContent string) (*ParsedSpec, error)
```

### Supporting

```go
// CalculateResources returns total resource requirements.
// Applies defaults: 0.5 CPU, 256MB memory per service without explicit limits.
// Disk: 1024MB per named volume.
func CalculateResources(spec *ParsedSpec) domain.Resources

// ExtractVariables returns placeholder variable names from parsed spec's environment.
// Note: Since compose-go interpolates placeholders, use ExtractVariablesFromYAML instead.
func ExtractVariables(spec *ParsedSpec) []string

// ExtractVariablesFromYAML extracts placeholder variable names from raw YAML content.
// This is the preferred method since compose-go interpolates ${VAR} placeholders during parsing.
func ExtractVariablesFromYAML(yamlContent string) []string

// ValidateParsedSpec performs semantic validation.
// Returns all validation errors found.
func ValidateParsedSpec(spec *ParsedSpec) []error
```

## Error Types

| Error | Condition |
|-------|-----------|
| ErrEmptyInput | Input is empty or whitespace only |
| ErrInvalidYAML | Input is not valid YAML |
| ErrNoServices | No services defined |
| ErrServiceNoImage | Service has neither image nor build |
| ErrServiceInvalidPort | Port number out of range (1-65535) |
| ErrServiceInvalidVolume | Invalid volume specification |
| ErrCircularDependency | Circular depends_on detected |
| ErrInvalidCPU | Negative CPU value |
| ErrInvalidMemory | Negative memory value |
| ErrUnsupportedFeature | Unsupported compose feature used |

### ParseError Wrapper

```go
type ParseError struct {
    Field   string // e.g., "services.web.ports[0]"
    Message string
    Err     error
}
```

## Edge Cases

### Input Validation
- Empty string → ErrEmptyInput
- Whitespace only → ErrEmptyInput
- Invalid YAML syntax → ErrInvalidYAML
- Valid YAML but not object → ErrInvalidYAML
- YAML with no services key → ErrNoServices
- Empty services map → ErrNoServices

### Port Parsing
- Short syntax: `"80:80"` → Port{Target: 80, Published: 80}
- With protocol: `"80:80/udp"` → Port{Protocol: "udp"}
- With IP: `"127.0.0.1:80:80"` → Port{HostIP: "127.0.0.1"}
- Target only: `"80"` → Port{Target: 80, Published: 0}
- Long syntax: `{target: 80, published: 8080}` → Port{Target: 80, Published: 8080}
- Invalid port (>65535) → ErrServiceInvalidPort
- Invalid port (0 target) → ErrServiceInvalidPort

### Volume Parsing
- Short syntax bind: `"./data:/app/data"` → VolumeMount{Type: "bind", Source: "./data", Target: "/app/data"}
- Short syntax named: `"mydata:/data"` → VolumeMount{Type: "volume", Source: "mydata", Target: "/data"}
- Read-only: `"./data:/data:ro"` → VolumeMount{ReadOnly: true}
- Long syntax: `{type: volume, source: mydata, target: /data}` → VolumeMount{...}

### Environment Variables
- Map syntax: `{KEY: value}` → map["KEY"] = "value"
- List syntax: `["KEY=value"]` → map["KEY"] = "value"
- Placeholder: `${VAR_NAME}` → resolved by compose-go (empty if not set)
- Default: `${VAR:-default}` → resolved to default value if not set
- Use `ExtractVariablesFromYAML(yamlContent)` to get placeholder names from raw YAML

### Dependencies
- Simple list: `[db, redis]` → DependsOn: ["db", "redis"]
- Long form: `{db: {condition: service_healthy}}` → DependsOn: ["db"]
- Circular: a→b→a → ErrCircularDependency
- Self-reference: a→a → ErrCircularDependency
- Missing dependency service → Warning (not error)

### Resources
- No limits specified → Default: 0.5 CPU, 256MB
- Explicit limits: `deploy.resources.limits.cpus: "2"` → CPULimit: 2.0
- Memory with units: `512M`, `1G` → Parsed to bytes
- Negative CPU → ErrInvalidCPU
- Negative memory → ErrInvalidMemory

## Resource Defaults

Per `specs/domain/template.md`:

| Resource | Default per Service |
|----------|---------------------|
| CPU | 0.5 cores |
| Memory | 256 MB (268435456 bytes) |
| Disk | 1024 MB per volume |

## Not Supported

These Compose features are intentionally not supported (per ADR-001):

| Feature | Handling |
|---------|----------|
| `deploy.replicas` | Ignored (single instance only) |
| `deploy.placement` | Ignored (no orchestration) |
| `scale` | Ignored |
| `extends` | ErrUnsupportedFeature |
| `secrets` | ErrUnsupportedFeature |
| `configs` | ErrUnsupportedFeature |
| Swarm mode features | ErrUnsupportedFeature |

## Dependencies

- `github.com/compose-spec/compose-go/v2` - Official compose parser
- `internal/core/domain` - For `domain.Resources` type

## Implementation Notes

1. Use compose-go for YAML parsing and schema validation
2. Convert compose-go types to Hoster-specific types
3. Do not expose compose-go types in public API
4. All functions must be pure (no I/O)
5. Return all validation errors, not just the first one

## Tests

- `internal/core/compose/parser_test.go` - Unit tests

### Test Categories

1. **Input Validation** (~5 tests)
   - Empty input
   - Whitespace only
   - Invalid YAML
   - Non-object YAML
   - No services

2. **Service Parsing** (~10 tests)
   - Image only
   - Build only
   - Image + build (image wins)
   - With command/entrypoint
   - With labels

3. **Port Parsing** (~8 tests)
   - Short syntax variations
   - Long syntax
   - With protocol
   - With IP binding
   - Invalid ranges

4. **Volume Parsing** (~6 tests)
   - Bind mounts
   - Named volumes
   - Read-only
   - Long syntax

5. **Environment Variables** (~4 tests)
   - Map syntax
   - List syntax
   - Placeholders
   - ExtractVariables function

6. **Network/Volume Definitions** (~4 tests)
   - Top-level networks
   - Top-level volumes
   - External resources

7. **Dependencies** (~4 tests)
   - Simple depends_on
   - Long form depends_on
   - Circular detection
   - Self-reference

8. **Resources** (~5 tests)
   - Defaults applied
   - Explicit limits
   - Memory units
   - CalculateResources function

9. **Complex Specs** (~4 tests)
   - WordPress + MySQL
   - Multi-service with networks
   - Real-world examples

**Total: ~40 tests**
