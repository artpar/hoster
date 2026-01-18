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

## JSON:API Resource Definition

Per ADR-003, Deployments are exposed as JSON:API resources.

### Resource Type

```
deployments
```

### Resource Structure

```json
{
  "data": {
    "type": "deployments",
    "id": "dep_abc123",
    "attributes": {
      "name": "wordpress-blog-a1b2c3",
      "template_id": "tmpl_xyz789",
      "template_version": "1.0.0",
      "customer_id": "user_cust456",
      "node_id": "node_001",
      "status": "running",
      "variables": {
        "DB_PASSWORD": "***REDACTED***"
      },
      "domains": [
        {
          "hostname": "wordpress-blog-a1b2c3.apps.hoster.io",
          "type": "auto",
          "ssl_enabled": true
        }
      ],
      "containers": [
        {
          "id": "abc123def456",
          "service_name": "wordpress",
          "image": "wordpress:latest",
          "status": "running"
        },
        {
          "id": "ghi789jkl012",
          "service_name": "mysql",
          "image": "mysql:8.0",
          "status": "running"
        }
      ],
      "resources": {
        "cpu_cores": 1.0,
        "memory_mb": 512,
        "disk_mb": 2048
      },
      "error_message": null,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T12:00:00Z",
      "started_at": "2024-01-15T10:31:00Z",
      "stopped_at": null
    },
    "relationships": {
      "template": {
        "data": {
          "type": "templates",
          "id": "tmpl_xyz789"
        },
        "links": {
          "related": "/api/v1/templates/tmpl_xyz789"
        }
      },
      "customer": {
        "data": {
          "type": "users",
          "id": "user_cust456"
        }
      }
    },
    "links": {
      "self": "/api/v1/deployments/dep_abc123"
    }
  }
}
```

### List Response

```json
{
  "data": [
    {"type": "deployments", "id": "dep_1", "attributes": {...}},
    {"type": "deployments", "id": "dep_2", "attributes": {...}}
  ],
  "links": {
    "self": "/api/v1/deployments?page[number]=1&page[size]=20",
    "first": "/api/v1/deployments?page[number]=1&page[size]=20",
    "next": "/api/v1/deployments?page[number]=2&page[size]=20"
  },
  "meta": {
    "total": 5,
    "page": 1,
    "page_size": 20
  }
}
```

### Filtering & Sorting

| Parameter | Description | Example |
|-----------|-------------|---------|
| `filter[customer_id]` | Filter by customer | `?filter[customer_id]=user_xyz` |
| `filter[template_id]` | Filter by template | `?filter[template_id]=tmpl_abc` |
| `filter[status]` | Filter by status | `?filter[status]=running` |
| `sort` | Sort field | `?sort=-created_at` |
| `page[number]` | Page number (1-based) | `?page[number]=2` |
| `page[size]` | Items per page | `?page[size]=20` |

### Actions (Non-CRUD Operations)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/deployments/:id/start` | Start a stopped deployment |
| POST | `/api/v1/deployments/:id/stop` | Stop a running deployment |
| POST | `/api/v1/deployments/:id/restart` | Restart a running deployment |

Action responses return the updated deployment resource.

### Security Notes

- `variables` field is redacted in responses (sensitive values replaced with `***REDACTED***`)
- `customer_id` filter is automatically applied based on auth context (users only see their own)
- Template relationship resolved only if template is published or user is creator

### api2go Implementation

```go
// internal/shell/api/resources/deployment.go

type Deployment struct {
    domain.Deployment
}

func (d Deployment) GetID() string {
    return d.ID
}

func (d *Deployment) SetID(id string) error {
    d.ID = id
    return nil
}

func (d Deployment) GetName() string {
    return "deployments"
}

func (d Deployment) GetReferences() []api2go.Reference {
    return []api2go.Reference{
        {Type: "templates", Name: "template"},
        {Type: "users", Name: "customer"},
    }
}

func (d Deployment) GetReferencedIDs() []api2go.ReferenceID {
    return []api2go.ReferenceID{
        {ID: d.TemplateID, Name: "template", Type: "templates"},
        {ID: d.CustomerID, Name: "customer", Type: "users"},
    }
}
```

### OpenAPI Schema

Generated reflectively from struct fields. Maps to:

```yaml
components:
  schemas:
    DeploymentAttributes:
      type: object
      properties:
        name:
          type: string
          readOnly: true
        template_id:
          type: string
        template_version:
          type: string
          readOnly: true
        customer_id:
          type: string
          readOnly: true
        node_id:
          type: string
          readOnly: true
        status:
          type: string
          enum: [pending, scheduled, starting, running, stopping, stopped, deleting, deleted, failed]
          readOnly: true
        variables:
          type: object
          additionalProperties:
            type: string
        domains:
          type: array
          readOnly: true
          items:
            $ref: '#/components/schemas/Domain'
        containers:
          type: array
          readOnly: true
          items:
            $ref: '#/components/schemas/ContainerInfo'
        resources:
          $ref: '#/components/schemas/Resources'
          readOnly: true
        error_message:
          type: string
          readOnly: true
        created_at:
          type: string
          format: date-time
          readOnly: true
        updated_at:
          type: string
          format: date-time
          readOnly: true
        started_at:
          type: string
          format: date-time
          readOnly: true
        stopped_at:
          type: string
          format: date-time
          readOnly: true

    CreateDeploymentInput:
      type: object
      required: [template_id]
      properties:
        template_id:
          type: string
        name:
          type: string
          description: Optional custom name
        variables:
          type: object
          additionalProperties:
            type: string
```

## Tests

- `internal/core/domain/deployment_test.go` - Deployment validation and state machine tests
- `internal/shell/api/resources/deployment_test.go` - JSON:API resource tests
