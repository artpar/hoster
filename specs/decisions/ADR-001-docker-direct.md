# ADR-001: Docker Direct (No Orchestration Layer)

## Status
Accepted

## Context

We need to deploy and manage containers across one or more VPS nodes. The common choices are:

1. **Kubernetes (K8s/K3s)** - Industry standard, feature-rich
2. **Docker Swarm** - Native Docker clustering
3. **HashiCorp Nomad** - Lightweight orchestrator
4. **Docker Direct** - Direct Docker API calls with custom scheduling

For our prototype phase, we need a minimal, resource-friendly solution that:
- Has low overhead per node
- Is simple to operate and debug
- Provides full control over container lifecycle
- Can be enhanced later without rewriting

## Decision

We will use **Docker Direct** - calling the Docker API directly from our Go backend via the official `github.com/docker/docker/client` SDK.

Our custom Go layer will handle:
- Container lifecycle (create, start, stop, remove)
- Network creation per deployment
- Volume management
- Health checks
- Basic scheduling (which node to use)

## Consequences

### Positive
- **Minimal overhead**: Only Docker daemon on each node, no orchestration layer
- **Full control**: We decide exactly how containers are managed
- **Simpler debugging**: Direct Docker commands work
- **Faster prototype**: No orchestration setup/learning curve
- **Excellent Go SDK**: Official Docker client is well-maintained
- **Flexibility**: Can add Swarm/K8s later if needed

### Negative
- **More custom code**: We build scheduling, health checks ourselves
- **No built-in HA**: High availability requires custom implementation
- **Single-node initially**: Multi-node requires agent development
- **No declarative state**: We manage state imperatively

### Neutral
- We're not locked in - can migrate to Swarm/K8s if scale demands it
- The Docker SDK abstractions we build will remain useful

## Alternatives Considered

### Kubernetes (K3s)
- **Rejected because**: Overkill for prototype, high learning curve, more resource overhead (512MB+ per node even for K3s)
- **Would reconsider if**: We need 50+ nodes, advanced networking, or auto-scaling

### Docker Swarm
- **Rejected because**: Adds overhead we don't need yet, harder to debug than direct Docker
- **Would reconsider if**: We need built-in multi-node with minimal custom code

### HashiCorp Nomad
- **Rejected because**: Additional dependency, more complex than direct Docker, requires Consul for service discovery
- **Would reconsider if**: We need heterogeneous workloads (VMs, containers, etc.)

## Migration Path

If we outgrow Docker Direct:
1. Our container spec abstraction remains valid
2. Replace `internal/shell/docker/` with Swarm/K8s client
3. Core logic (`internal/core/`) unchanged
4. Tests for core logic still pass

This is enabled by our "Values as Boundaries" architecture (see ADR-002).
