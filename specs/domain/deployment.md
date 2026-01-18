# Deployment

## Overview

A Deployment is a running instance of a Template. When a customer deploys a template, a Deployment is created that tracks the lifecycle of the containers, domains, and configuration for that instance.

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Yes (auto) | Unique identifier |
| `name` | string | Yes (auto) | Human-readable name (derived from template + random suffix) |
| `template_id` | UUID | Yes | Reference to the template being deployed |
| `template_version` | string | Yes | Version of template at time of deployment |
| `customer_id` | UUID | Yes | Who owns this deployment |
| `node_id` | UUID | No | Which node this is deployed on (assigned during scheduling) |
| `status` | enum | Yes | Current status (see State Machine) |
| `variables` | map[string]string | No | Variable values provided by customer |
| `domains` | []Domain | No | Assigned domains for this deployment |
| `containers` | []ContainerInfo | No | Container IDs and metadata |
| `resources` | Resources | Yes | Actual resources allocated |
| `error_message` | string | No | Error details if status is `failed` |
| `created_at` | timestamp | Yes (auto) | When created |
| `updated_at` | timestamp | Yes (auto) | When last modified |
| `started_at` | timestamp | No | When containers started |
| `stopped_at` | timestamp | No | When containers stopped |

### Domain Type

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `hostname` | string | Yes | Full hostname (e.g., "app-xyz.hoster.io") |
| `type` | enum | Yes | `auto` (generated) or `custom` |
| `ssl_enabled` | bool | Yes | Whether SSL is configured |
| `ssl_expires_at` | timestamp | No | SSL certificate expiration |

### ContainerInfo Type

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Docker container ID |
| `service_name` | string | Service name from compose spec |
| `image` | string | Docker image used |
| `status` | string | Container status (running, stopped, etc.) |
| `ports` | []PortMapping | Exposed ports |

## Status (State Machine)

```
                              ┌─────────────┐
                              │   pending   │
                              └──────┬──────┘
                                     │ schedule
                                     ▼
                              ┌─────────────┐
                              │  scheduled  │
                              └──────┬──────┘
                                     │ provision
                                     ▼
                              ┌─────────────┐
                     ┌───────►│  starting   │◄───────┐
                     │        └──────┬──────┘        │
                     │               │ start         │ restart
                     │               ▼               │
                     │        ┌─────────────┐        │
                     │        │   running   │────────┤
                     │        └──────┬──────┘        │
                     │               │ stop          │
                     │               ▼               │
                     │        ┌─────────────┐        │
                     │        │   stopped   │────────┘
                     │        └──────┬──────┘
                     │               │ delete
                     │               ▼
                     │        ┌─────────────┐
                     │        │   deleted   │
                     │        └─────────────┘
                     │
    ┌────────────────┴────────────────┐
    │  Any state can transition to    │
    │  'failed' on error              │
    └─────────────────────────────────┘
```

### Status Values

| Status | Description |
|--------|-------------|
| `pending` | Just created, waiting to be scheduled |
| `scheduled` | Assigned to a node, waiting for provisioning |
| `starting` | Containers are being created/started |
| `running` | All containers healthy and running |
| `stopping` | Containers are being stopped |
| `stopped` | Containers stopped but still exist |
| `deleting` | Being deleted, containers being removed |
| `deleted` | Fully removed (soft delete in DB) |
| `failed` | An error occurred, see error_message |

### Valid Transitions

| From | To | Trigger |
|------|-----|---------|
| `pending` | `scheduled` | Node assigned by scheduler |
| `scheduled` | `starting` | Provisioning started |
| `starting` | `running` | All containers healthy |
| `starting` | `failed` | Container failed to start |
| `running` | `stopping` | Stop requested |
| `running` | `failed` | Container crashed |
| `stopping` | `stopped` | All containers stopped |
| `stopped` | `starting` | Restart requested |
| `stopped` | `deleting` | Delete requested |
| `deleting` | `deleted` | Cleanup complete |
| `failed` | `starting` | Retry requested |
| `failed` | `deleting` | Delete requested |

## Invariants

1. **Template must exist**: Cannot create deployment for non-existent template
2. **Customer must exist**: Cannot create deployment for non-existent customer
3. **Variables must match template**: All required template variables must be provided
4. **Status transitions are validated**: Only valid state transitions allowed
5. **Node required for starting**: Cannot transition to `starting` without a node

## Behaviors

### Name Generation
Auto-generated from template slug + random suffix:
- Template: "wordpress-blog"
- Deployment name: "wordpress-blog-a1b2c3"

### Domain Generation
Auto-generated subdomain when deployment starts:
- Pattern: `{deployment-name}.{base-domain}`
- Example: `wordpress-blog-a1b2c3.apps.hoster.io`

### Variable Validation
Variables provided must satisfy template requirements:
- All required variables must have values
- Values must pass template-defined validation patterns
- Unknown variables are ignored

### Health Checking
Deployment is `running` when:
- All containers in compose spec are running
- Health checks (if defined) are passing

## Validation Rules

### Create Deployment
```go
func ValidateCreateDeployment(req CreateDeploymentRequest, template Template) []error
// - Template exists and is published
// - All required variables provided
// - Variable values pass validation
// Returns: ErrTemplateNotFound, ErrTemplateNotPublished, ErrMissingVariable, ErrInvalidVariable
```

### Transition Validation
```go
func ValidateTransition(from, to DeploymentStatus) error
// - Check if transition is allowed
// Returns: ErrInvalidTransition
```

## Not Supported

1. **Scaling replicas**: Each deployment runs one instance of each service
   - *Reason*: Prototype simplicity
   - *Future*: May add horizontal scaling

2. **Zero-downtime updates**: Updates require stop → start
   - *Reason*: No orchestration layer
   - *Future*: May implement rolling updates

3. **Automatic restarts**: Crashed containers stay crashed
   - *Reason*: Prototype simplicity
   - *Future*: Add restart policies

4. **Resource limits enforcement**: Trust Docker defaults
   - *Reason*: Prototype simplicity
   - *Future*: Add cgroup limits

## Examples

### Valid Deployment
```go
Deployment{
    ID:              "660e8400-e29b-41d4-a716-446655440000",
    Name:            "wordpress-blog-a1b2c3",
    TemplateID:      "550e8400-e29b-41d4-a716-446655440000",
    TemplateVersion: "1.0.0",
    CustomerID:      "770e8400-e29b-41d4-a716-446655440000",
    NodeID:          "880e8400-e29b-41d4-a716-446655440000",
    Status:          StatusRunning,
    Variables: map[string]string{
        "DB_PASSWORD": "secret123",
    },
    Domains: []Domain{
        {Hostname: "wordpress-blog-a1b2c3.apps.hoster.io", Type: "auto", SSLEnabled: true},
    },
    Containers: []ContainerInfo{
        {ID: "abc123", ServiceName: "wordpress", Status: "running"},
        {ID: "def456", ServiceName: "db", Status: "running"},
    },
}
```

### Invalid Transitions
```go
// Cannot go from pending to running directly
err := deployment.Transition(StatusRunning) // ErrInvalidTransition

// Cannot restart a running deployment
err := deployment.Transition(StatusStarting) // ErrInvalidTransition (must stop first)
```

## Tests

- `internal/core/domain/deployment_test.go` - Deployment validation and state machine tests
