# ADR-004: Reflective OpenAPI Generation

## Status
Accepted

## Context

We need an OpenAPI specification for the Hoster API that:
- Enables strongly-typed TypeScript client generation
- Stays synchronized with the actual API implementation
- Requires minimal manual maintenance
- Supports JSON:API format (per ADR-003)

Common approaches to OpenAPI:
1. **Spec-first**: Write OpenAPI YAML, generate server code
2. **Annotation-based**: Add comments/annotations to code, generate spec (swaggo)
3. **Reflective**: Generate spec at runtime by reflecting on registered resources
4. **Manual**: Write and maintain spec separately from code

## Decision

We will build a **custom reflective OpenAPI generator** that produces an OpenAPI 3.0 specification at runtime by inspecting api2go registered resources.

### Why Reflective Generation?

1. **Always in sync**: Spec generated from actual code, no drift possible
2. **No annotations**: Clean code without swagger comments
3. **No build step**: Spec available at runtime via `/openapi.json` endpoint
4. **api2go integration**: Leverages existing resource definitions
5. **JSON:API aware**: Can generate JSON:API-compliant schemas

### Why Not Spec-First?

- Requires maintaining spec separately from code
- Drift between spec and implementation is common bug source
- Two sources of truth (spec and code)

### Why Not Annotation-Based (swaggo)?

- Adds noise to code with comments
- Requires build step to generate spec
- Doesn't integrate well with api2go patterns
- Comments can drift from actual behavior

## Implementation

### Generator Architecture

```go
// internal/shell/api/openapi/generator.go

type Generator struct {
    api     *api2go.API
    info    *openapi3.Info
    baseURL string
}

func NewGenerator(api *api2go.API, opts ...Option) *Generator

// Generate produces complete OpenAPI spec
func (g *Generator) Generate() *openapi3.T

// Handler returns HTTP handler serving /openapi.json
func (g *Generator) Handler() http.Handler
```

### Reflection Strategy

The generator inspects each api2go resource to extract:

1. **Resource type** → Path prefix (`/templates`, `/deployments`)
2. **Struct fields** → Schema properties with JSON types
3. **Field tags** → Required fields, descriptions, validation
4. **Relationships** → Links and relationship endpoints
5. **Implemented interfaces** → Available operations (GET, POST, PATCH, DELETE)

```go
// internal/shell/api/openapi/schema.go

// ExtractSchema reflects on a struct to produce OpenAPI schema
func ExtractSchema(v interface{}) *openapi3.SchemaRef

// ExtractOperations determines which HTTP methods are supported
func ExtractOperations(resource interface{}) []string

// BuildJSONAPISchema wraps schema in JSON:API envelope
func BuildJSONAPISchema(resourceType string, schema *openapi3.SchemaRef) *openapi3.SchemaRef
```

### Generated Paths

For a `templates` resource implementing all CRUD interfaces:

| Method | Path | Operation ID |
|--------|------|--------------|
| GET | /api/v1/templates | listTemplates |
| POST | /api/v1/templates | createTemplate |
| GET | /api/v1/templates/{id} | getTemplate |
| PATCH | /api/v1/templates/{id} | updateTemplate |
| DELETE | /api/v1/templates/{id} | deleteTemplate |
| GET | /api/v1/templates/{id}/relationships/deployments | getTemplateDeployments |

### Generated Schema (JSON:API Format)

```yaml
components:
  schemas:
    Template:
      type: object
      properties:
        type:
          type: string
          enum: ["templates"]
        id:
          type: string
        attributes:
          $ref: '#/components/schemas/TemplateAttributes'
        relationships:
          $ref: '#/components/schemas/TemplateRelationships'

    TemplateAttributes:
      type: object
      required: [name, version, compose_spec]
      properties:
        name:
          type: string
          minLength: 3
          maxLength: 100
        version:
          type: string
          pattern: "^\\d+\\.\\d+\\.\\d+$"
        # ... other fields

    TemplateResponse:
      type: object
      properties:
        data:
          $ref: '#/components/schemas/Template'
        links:
          $ref: '#/components/schemas/Links'

    TemplateListResponse:
      type: object
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Template'
        links:
          $ref: '#/components/schemas/PaginationLinks'
        meta:
          $ref: '#/components/schemas/PaginationMeta'
```

### Serving the Spec

```go
// cmd/hoster/server.go

func setupAPI(store store.Store, docker docker.Client) http.Handler {
    api := api2go.NewAPIWithRouting(...)

    // Register resources
    api.AddResource(Template{}, &TemplateResource{store: store})
    api.AddResource(Deployment{}, &DeploymentResource{store: store, docker: docker})

    // Create OpenAPI generator
    openAPIGen := openapi.NewGenerator(api,
        openapi.WithTitle("Hoster API"),
        openapi.WithVersion("1.0.0"),
        openapi.WithServer("http://localhost:9090"),
    )

    // Mount OpenAPI endpoint
    mux := api.Router().(*mux.Router)
    mux.HandleFunc("/openapi.json", openAPIGen.Handler()).Methods("GET")

    return mux
}
```

### Type Mapping

Go types are mapped to OpenAPI types:

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| string | string | - |
| int, int32 | integer | int32 |
| int64 | integer | int64 |
| float32 | number | float |
| float64 | number | double |
| bool | boolean | - |
| time.Time | string | date-time |
| []T | array | items: T |
| map[string]T | object | additionalProperties: T |
| *T | T (nullable: true) | - |

### Struct Tag Support

```go
type Template struct {
    Name        string  `json:"name" openapi:"required,minLength=3,maxLength=100"`
    Version     string  `json:"version" openapi:"required,pattern=^\\d+\\.\\d+\\.\\d+$"`
    Description string  `json:"description,omitempty" openapi:"maxLength=1000"`
    Price       int64   `json:"price_monthly_cents" openapi:"minimum=0"`
}
```

## Consequences

### Positive
- **Zero drift**: Spec always matches implementation
- **No maintenance**: No separate spec file to maintain
- **Runtime available**: Spec served dynamically, no build step
- **Clean code**: No annotation noise in source files
- **TypeScript generation**: `openapi-typescript` works with served spec

### Negative
- **Custom code**: Must build and maintain generator
- **Reflection complexity**: Go reflection has sharp edges
- **Startup cost**: Small overhead generating spec on first request
- **Limited customization**: Hard to add non-code-derived documentation

### Neutral
- Caching can mitigate startup cost (generate once, serve cached)
- Can augment generated spec with manual additions if needed

## Alternatives Considered

### swaggo/swag (Annotation-based)
- **Rejected because**: Clutters code with comments, requires build step
- **Would reconsider if**: Reflection proves too complex

### kin-openapi with Manual Spec
- **Rejected because**: Manual maintenance, drift risk
- **Would reconsider if**: We need heavily customized documentation

### ogen (Spec-first)
- **Rejected because**: Generates server code, conflicts with api2go
- **Would reconsider if**: We move away from api2go

## TypeScript Generation

Frontend uses generated spec:

```bash
# web/package.json
"scripts": {
  "generate:types": "openapi-typescript http://localhost:9090/openapi.json -o src/api/schema.d.ts"
}
```

```typescript
// Usage in frontend
import type { components, paths } from './schema';

type Template = components['schemas']['Template'];
type ListTemplatesResponse = paths['/api/v1/templates']['get']['responses']['200']['content']['application/vnd.api+json'];
```

## Files to Create

- `internal/shell/api/openapi/generator.go` - Main generator
- `internal/shell/api/openapi/schema.go` - Schema extraction
- `internal/shell/api/openapi/paths.go` - Path generation
- `internal/shell/api/openapi/jsonapi.go` - JSON:API envelope handling
- `internal/shell/api/openapi/generator_test.go` - Tests

## Dependencies

```go
"github.com/getkin/kin-openapi/openapi3"  // OpenAPI 3.0 types
```

## References

- OpenAPI Specification 3.0: https://spec.openapis.org/oas/v3.0.3
- kin-openapi: https://github.com/getkin/kin-openapi
- openapi-typescript: https://github.com/drwpow/openapi-typescript
- JSON:API OpenAPI Guide: https://jsonapi.org/recommendations/#openapi
