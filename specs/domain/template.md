# Template

## Overview

A Template is a deployable package definition. It contains a Docker Compose specification, configurable variables, resource requirements, and pricing information. Templates are created by package creators and deployed as instances by customers.

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Yes (auto) | Unique identifier, generated on creation |
| `name` | string | Yes | Human-readable name (3-100 chars, alphanumeric + spaces + hyphens) |
| `slug` | string | Yes (auto) | URL-safe identifier derived from name |
| `description` | string | No | Markdown description of what this template deploys |
| `version` | string | Yes | Semantic version (e.g., "1.0.0") |
| `compose_spec` | string | Yes | Docker Compose YAML content |
| `variables` | []Variable | No | User-configurable variables |
| `resource_requirements` | Resources | Yes (auto) | Computed from compose spec |
| `price_monthly_cents` | int64 | Yes | Monthly price in cents (0 = free) |
| `category` | string | No | Category for marketplace (e.g., "cms", "database") |
| `tags` | []string | No | Tags for search/filtering |
| `published` | bool | Yes | Whether visible in marketplace |
| `creator_id` | UUID | Yes | Who created this template |
| `created_at` | timestamp | Yes (auto) | When created |
| `updated_at` | timestamp | Yes (auto) | When last modified |

### Variable Type

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Variable name (used in compose spec as `${VAR_NAME}`) |
| `label` | string | Yes | Human-readable label |
| `description` | string | No | Help text |
| `type` | enum | Yes | `string`, `number`, `boolean`, `password`, `select` |
| `default` | string | No | Default value |
| `required` | bool | Yes | Whether user must provide value |
| `options` | []string | No | Valid options (for `select` type) |
| `validation` | string | No | Regex pattern for validation |

### Resources Type

| Field | Type | Description |
|-------|------|-------------|
| `cpu_cores` | float64 | CPU cores required |
| `memory_mb` | int64 | Memory in MB |
| `disk_mb` | int64 | Disk space in MB |

## Invariants

1. **Name is required**: Must be 3-100 characters
2. **Name format**: Only alphanumeric, spaces, and hyphens allowed
3. **Slug is unique**: No two templates can have the same slug
4. **Version is semver**: Must match `X.Y.Z` pattern
5. **Compose spec is valid**: Must parse as valid Docker Compose
6. **Price is non-negative**: Must be >= 0
7. **Variables are unique**: No duplicate variable names
8. **Published requires version**: Cannot publish without a version

## Behaviors

### Slug Generation
- Derived from name: lowercase, spaces → hyphens, remove special chars
- Example: "WordPress Blog" → "wordpress-blog"

### Version Comparison
- Follows semver ordering: 1.0.0 < 1.0.1 < 1.1.0 < 2.0.0

### Resource Calculation
- Extracted from compose spec services
- Sum of all service resource limits
- If not specified in compose, use defaults:
  - CPU: 0.5 cores per service
  - Memory: 256 MB per service
  - Disk: 1024 MB per volume

### Variable Substitution
Variables are substituted in compose spec before deployment:
```yaml
# Template compose spec
services:
  db:
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD}

# After substitution with DB_PASSWORD=secret123
services:
  db:
    environment:
      MYSQL_ROOT_PASSWORD: secret123
```

## Validation Rules

### Name Validation
```go
func ValidateName(name string) error
// - Non-empty
// - 3-100 characters
// - Only alphanumeric, spaces, hyphens
// Returns: ErrNameRequired, ErrNameTooShort, ErrNameTooLong, ErrNameInvalidChars
```

### Version Validation
```go
func ValidateVersion(version string) error
// - Matches semver pattern: X.Y.Z
// Returns: ErrVersionRequired, ErrVersionInvalidFormat
```

### Compose Spec Validation
```go
func ValidateComposeSpec(spec string) (*ParsedSpec, error)
// - Valid YAML
// - Valid Docker Compose structure
// - At least one service defined
// Returns: ErrComposeInvalidYAML, ErrComposeNoServices
```

### Variable Validation
```go
func ValidateVariables(vars []Variable) []error
// - Unique names
// - Valid types
// - Options provided for select type
// Returns: []error (multiple validation errors possible)
```

## State Transitions

```
[Draft] --publish--> [Published] --unpublish--> [Draft]
                          |
                          +--archive--> [Archived]
```

- **Draft**: Initial state, not visible in marketplace
- **Published**: Visible in marketplace, can be deployed
- **Archived**: Hidden, existing deployments continue to work

## Not Supported

1. **Template inheritance**: Templates cannot extend other templates
   - *Reason*: Adds complexity, compose-go doesn't support it well
   - *Workaround*: Copy and modify

2. **Dynamic pricing**: Price is fixed per template
   - *Reason*: Prototype simplicity
   - *Future*: May add resource-based pricing

3. **Private templates**: All published templates are public
   - *Reason*: Prototype simplicity
   - *Future*: May add visibility controls

4. **Collaborative editing**: Single creator per template
   - *Reason*: Prototype simplicity
   - *Future*: May add team ownership

## Examples

### Valid Template
```go
Template{
    ID:          "550e8400-e29b-41d4-a716-446655440000",
    Name:        "WordPress with MySQL",
    Slug:        "wordpress-with-mysql",
    Version:     "1.0.0",
    Description: "A WordPress blog with MySQL database",
    ComposeSpec: `
services:
  wordpress:
    image: wordpress:latest
    ports:
      - "80:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_PASSWORD: ${DB_PASSWORD}
  db:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: ${DB_PASSWORD}
`,
    Variables: []Variable{
        {Name: "DB_PASSWORD", Label: "Database Password", Type: "password", Required: true},
    },
    PriceMonthly: 999, // $9.99
    Published:    true,
}
```

### Invalid Template (validation errors)
```go
Template{
    Name: "WP",        // ErrNameTooShort (< 3 chars)
    Version: "1.0",    // ErrVersionInvalidFormat (not X.Y.Z)
    ComposeSpec: "not yaml", // ErrComposeInvalidYAML
}
```

## JSON:API Resource Definition

Per ADR-003, Templates are exposed as JSON:API resources.

### Resource Type

```
templates
```

### Resource Structure

```json
{
  "data": {
    "type": "templates",
    "id": "tmpl_abc123",
    "attributes": {
      "name": "WordPress with MySQL",
      "slug": "wordpress-with-mysql",
      "description": "A WordPress blog with MySQL database",
      "version": "1.0.0",
      "compose_spec": "services:\n  wordpress:\n    image: wordpress:latest\n...",
      "variables": [
        {
          "name": "DB_PASSWORD",
          "label": "Database Password",
          "description": "Root password for MySQL",
          "type": "password",
          "required": true
        }
      ],
      "resource_requirements": {
        "cpu_cores": 1.0,
        "memory_mb": 512,
        "disk_mb": 2048
      },
      "price_monthly_cents": 999,
      "category": "cms",
      "tags": ["wordpress", "blog", "mysql"],
      "published": true,
      "creator_id": "user_xyz789",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T12:00:00Z"
    },
    "relationships": {
      "deployments": {
        "links": {
          "related": "/api/v1/templates/tmpl_abc123/deployments"
        }
      },
      "creator": {
        "data": {
          "type": "users",
          "id": "user_xyz789"
        }
      }
    },
    "links": {
      "self": "/api/v1/templates/tmpl_abc123"
    }
  }
}
```

### List Response

```json
{
  "data": [
    {"type": "templates", "id": "tmpl_1", "attributes": {...}},
    {"type": "templates", "id": "tmpl_2", "attributes": {...}}
  ],
  "links": {
    "self": "/api/v1/templates?page[number]=1&page[size]=20",
    "first": "/api/v1/templates?page[number]=1&page[size]=20",
    "last": "/api/v1/templates?page[number]=5&page[size]=20",
    "next": "/api/v1/templates?page[number]=2&page[size]=20"
  },
  "meta": {
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### Filtering & Sorting

| Parameter | Description | Example |
|-----------|-------------|---------|
| `filter[published]` | Filter by publish status | `?filter[published]=true` |
| `filter[creator_id]` | Filter by creator | `?filter[creator_id]=user_xyz` |
| `filter[search]` | Full-text search name/description | `?filter[search]=wordpress` |
| `sort` | Sort field (prefix `-` for descending) | `?sort=-created_at` |
| `page[number]` | Page number (1-based) | `?page[number]=2` |
| `page[size]` | Items per page | `?page[size]=20` |

### api2go Implementation

```go
// internal/shell/api/resources/template.go

type Template struct {
    domain.Template
}

func (t Template) GetID() string {
    return t.ID
}

func (t *Template) SetID(id string) error {
    t.ID = id
    return nil
}

func (t Template) GetName() string {
    return "templates"
}

func (t Template) GetReferences() []api2go.Reference {
    return []api2go.Reference{
        {Type: "deployments", Name: "deployments"},
        {Type: "users", Name: "creator"},
    }
}

func (t Template) GetReferencedIDs() []api2go.ReferenceID {
    return []api2go.ReferenceID{
        {ID: t.CreatorID, Name: "creator", Type: "users"},
    }
}
```

### OpenAPI Schema

Generated reflectively from struct fields. Maps to:

```yaml
components:
  schemas:
    TemplateAttributes:
      type: object
      required: [name, version, compose_spec, price_monthly_cents]
      properties:
        name:
          type: string
          minLength: 3
          maxLength: 100
        slug:
          type: string
          readOnly: true
        description:
          type: string
        version:
          type: string
          pattern: "^\\d+\\.\\d+\\.\\d+$"
        compose_spec:
          type: string
        variables:
          type: array
          items:
            $ref: '#/components/schemas/Variable'
        resource_requirements:
          $ref: '#/components/schemas/Resources'
        price_monthly_cents:
          type: integer
          minimum: 0
        category:
          type: string
        tags:
          type: array
          items:
            type: string
        published:
          type: boolean
        creator_id:
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
```

## Tests

- `internal/core/domain/template_test.go` - Template validation tests
- `internal/core/compose/parser_test.go` - Compose parsing tests
- `internal/shell/api/resources/template_test.go` - JSON:API resource tests
