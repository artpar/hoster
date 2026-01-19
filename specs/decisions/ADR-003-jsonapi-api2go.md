# ADR-003: JSON:API Standard with api2go

## Status
Accepted

## Context

We need a standardized API format for Hoster that:
- Enables consistent frontend development
- Supports strongly-typed client generation
- Handles relationships between resources (templates ↔ deployments)
- Provides standardized pagination, filtering, and sorting
- Works with automated tooling (OpenAPI generation, TypeScript generation)

The current API uses custom JSON structures with chi router. This requires manual type maintenance and doesn't follow any standard format.

Common API format options:
1. **Custom REST JSON** - What we have now
2. **JSON:API** - Specification for building APIs in JSON
3. **GraphQL** - Query language for APIs
4. **gRPC** - Protocol buffer-based RPC

## Decision

We will adopt the **JSON:API specification** (jsonapi.org) using the `github.com/manyminds/api2go` library.

### Why JSON:API?

1. **Standardized format**: Consistent `{type, id, attributes, relationships}` structure
2. **Compound documents**: Include related resources in single response (reduces round-trips)
3. **Standardized pagination**: `page[number]`, `page[size]` or `page[offset]`, `page[limit]`
4. **Standardized filtering**: Query parameter conventions
5. **Relationships**: First-class support for resource links
6. **Tooling ecosystem**: Clients exist for most languages

### Why api2go?

1. **Go native**: Written in Go, maintained library
2. **Interface-based**: Define resources via interfaces, not annotations
3. **Router support**: Built-in Gorilla mux integration (also Gin, Echo)
4. **Full JSON:API compliance**: Handles marshaling/unmarshaling automatically
5. **Relationship support**: To-one, to-many, included resources

### Router Change: chi → Gorilla mux

api2go has built-in support for Gorilla mux but not chi. Rather than building a custom adapter:
- We migrate from `github.com/go-chi/chi/v5` to `github.com/gorilla/mux`
- Gorilla mux is mature, well-maintained, and compatible with our middleware patterns
- Migration is straightforward (both are `http.Handler` based)

## Implementation

### Resource Definition

```go
// internal/shell/api/resources/template.go

type TemplateResource struct {
    store store.Store
    auth  *auth.Middleware
}

// JSON:API model wrapping domain type
type Template struct {
    domain.Template
}

func (t Template) GetID() string              { return t.ID }
func (t Template) SetID(id string) error      { t.ID = id; return nil }
func (t Template) GetName() string            { return "templates" }
func (t Template) GetReferences() []api2go.Reference {
    return []api2go.Reference{
        {Type: "deployments", Name: "deployments"},
    }
}

// Resource CRUD operations
func (r TemplateResource) FindAll(req api2go.Request) (api2go.Responder, error)
func (r TemplateResource) FindOne(id string, req api2go.Request) (api2go.Responder, error)
func (r TemplateResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error)
func (r TemplateResource) Update(obj interface{}, req api2go.Request) (api2go.Responder, error)
func (r TemplateResource) Delete(id string, req api2go.Request) (api2go.Responder, error)
```

### API Setup

```go
// cmd/hoster/server.go

func setupAPI(store store.Store, docker docker.Client) *api2go.API {
    api := api2go.NewAPIWithRouting(
        "v1",
        api2go.NewStaticResolver("/api"),
        gzip.New(),
    )

    api.AddResource(Template{}, &TemplateResource{store: store})
    api.AddResource(Deployment{}, &DeploymentResource{store: store, docker: docker})

    return api
}
```

### Response Format

```json
{
  "data": {
    "type": "templates",
    "id": "tmpl_abc123",
    "attributes": {
      "name": "WordPress",
      "slug": "wordpress",
      "version": "1.0.0",
      "description": "WordPress with MySQL",
      "price_monthly_cents": 500,
      "published": true
    },
    "relationships": {
      "deployments": {
        "links": {
          "related": "/api/v1/templates/tmpl_abc123/deployments"
        }
      }
    }
  },
  "links": {
    "self": "/api/v1/templates/tmpl_abc123"
  }
}
```

### List Response with Pagination

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
    "total": 100
  }
}
```

## Consequences

### Positive
- **Standardized API**: Follows JSON:API spec exactly
- **Reduced boilerplate**: api2go handles marshaling, routing
- **Frontend compatibility**: JSON:API clients exist for React, Vue, etc.
- **Relationship handling**: Included resources reduce N+1 queries
- **Pagination built-in**: Standard format, automatic link generation
- **Content negotiation**: Proper `application/vnd.api+json` media type

### Negative
- **Router migration**: Must switch from chi to Gorilla mux
- **Learning curve**: JSON:API has specific conventions to learn
- **Verbose responses**: More structure than plain JSON (but more useful)
- **Test updates**: All API tests must be updated for new format

### Neutral
- api2go is a thin wrapper - we still control business logic
- Migration can be done incrementally (run both routers during transition)

## Alternatives Considered

### Keep Custom REST JSON
- **Rejected because**: No standard format, manual type sync, no relationship handling
- **Would reconsider if**: JSON:API proves too verbose for our use case

### GraphQL
- **Rejected because**: Overkill for our resource-oriented API, more complex tooling
- **Would reconsider if**: Frontend needs complex query flexibility

### gRPC
- **Rejected because**: Requires code generation, no browser-native support
- **Would reconsider if**: We need high-performance internal service communication

### Build chi adapter for api2go
- **Rejected because**: Additional maintenance burden, Gorilla mux works well
- **Would reconsider if**: chi provides features we critically need

## Migration Path

1. Add Gorilla mux and api2go dependencies
2. Create resource wrappers for Template, Deployment
3. Set up api2go routing alongside existing chi routes
4. Migrate one resource at a time with tests
5. Remove chi once migration complete
6. Update frontend to use JSON:API format

## Files Affected

### New Files
- `internal/shell/api/resources/template.go`
- `internal/shell/api/resources/deployment.go`
- `internal/shell/api/jsonapi/types.go`

### Modified Files
- `go.mod` - Add api2go, gorilla/mux; remove chi
- `cmd/hoster/server.go` - api2go setup
- `internal/shell/api/handler.go` - Refactor to use api2go
- All `*_test.go` files - Update for JSON:API format

## Frontend Implementation: Generic Factories

### Design Principle: Maximum Reuse, Minimum Debt

To maintain JSON:API spec alignment across the entire stack while minimizing code duplication and technical debt, we implement **generic factory patterns** for both API clients and React hooks.

**Key Insight**: Every JSON:API resource follows the same `{type, id, attributes}` structure and supports the same CRUD operations. Rather than writing boilerplate for each resource, we generate type-safe clients and hooks from configuration.

### API Client Factory

```typescript
// web/src/api/createResourceApi.ts

export function createResourceApi<
  Resource extends JsonApiResource<string, unknown>,
  CreateRequest,
  UpdateRequest = Partial<CreateRequest>,
  CustomActions extends string = never
>(options: CreateResourceApiOptions<CustomActions>): ResourceApi<...> & Record<CustomActions, ...>
```

**Features:**
- Type-safe CRUD operations (list, get, create, update, delete)
- Custom actions support (e.g., `publish`, `enterMaintenance`)
- Configurable update/delete support (for immutable resources like SSH keys)
- Automatic JSON:API request body formatting

**Usage:**
```typescript
// Simple CRUD resource
export const templatesApi = createResourceApi<Template, CreateTemplateRequest>({
  resourceName: 'templates',
});

// Resource with custom actions
export const nodesApi = createResourceApi<Node, CreateNodeRequest, UpdateNodeRequest,
  'enterMaintenance' | 'exitMaintenance'
>({
  resourceName: 'nodes',
  customActions: {
    enterMaintenance: { method: 'POST', path: 'maintenance', requiresId: true },
    exitMaintenance: { method: 'DELETE', path: 'maintenance', requiresId: true },
  },
});

// Immutable resource (no update)
export const sshKeysApi = createResourceApi<SSHKey, CreateSSHKeyRequest, never>({
  resourceName: 'ssh_keys',
  supportsUpdate: false,
});
```

### TanStack Query Hooks Factory

```typescript
// web/src/hooks/createResourceHooks.ts

export function createResourceHooks<Resource, CreateRequest, UpdateRequest>(
  options: CreateResourceHooksOptions<...>
): ResourceHooks<...>
```

**Features:**
- Standard query keys pattern for cache management
- Automatic cache invalidation on mutations
- `createIdActionHook` for custom id-based actions
- Type-safe throughout

**Usage:**
```typescript
const nodeHooks = createResourceHooks<Node, CreateNodeRequest, UpdateNodeRequest>({
  resourceName: 'nodes',
  api: nodesApi,
});

export const nodeKeys = nodeHooks.keys;
export const useNodes = nodeHooks.useList;
export const useNode = nodeHooks.useGet;
export const useCreateNode = nodeHooks.useCreate;
export const useUpdateNode = nodeHooks.useUpdate;
export const useDeleteNode = nodeHooks.useDelete;

// Custom actions
export const useEnterMaintenanceMode = createIdActionHook(
  nodeKeys, nodesApi.enterMaintenance
);
```

### Code Reduction Results

| Resource | Without Factory | With Factory | Savings |
|----------|-----------------|--------------|---------|
| nodes.ts | ~45 lines | ~27 lines | 40% |
| ssh-keys.ts | ~27 lines | ~21 lines | 22% |
| useNodes.ts | ~73 lines | ~34 lines | 53% |
| useSSHKeys.ts | ~45 lines | ~27 lines | 40% |
| templates.ts | ~44 lines | ~25 lines | 43% |
| useTemplates.ts | ~73 lines | ~29 lines | 60% |

**Average code reduction: ~43%**

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           JSON:API Specification                         │
│                         (jsonapi.org standard)                           │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
┌─────────────────────────────────┐ ┌─────────────────────────────────────┐
│        BACKEND (Go)              │ │         FRONTEND (TypeScript)        │
│                                  │ │                                      │
│  api2go Resource Interface       │ │  createResourceApi<T> Factory        │
│  ┌────────────────────────────┐  │ │  ┌──────────────────────────────┐   │
│  │ FindAll, FindOne, Create,  │  │ │  │ list, get, create, update,   │   │
│  │ Update, Delete             │  │ │  │ delete, + customActions      │   │
│  └────────────────────────────┘  │ │  └──────────────────────────────┘   │
│              │                   │ │              │                       │
│              ▼                   │ │              ▼                       │
│  ┌────────────────────────────┐  │ │  ┌──────────────────────────────┐   │
│  │ TemplateResource           │  │ │  │ createResourceHooks<T>       │   │
│  │ DeploymentResource         │  │ │  │ useList, useGet, useCreate,  │   │
│  │ NodeResource               │  │ │  │ useUpdate, useDelete         │   │
│  │ SSHKeyResource             │  │ │  │ + createIdActionHook         │   │
│  └────────────────────────────┘  │ │  └──────────────────────────────┘   │
└─────────────────────────────────┘ └─────────────────────────────────────┘
```

### Adding New Resources

When adding a new JSON:API resource, follow this pattern:

**1. Backend (Go):**
```go
// internal/shell/api/resources/foo.go
type FooResource struct {
    store store.Store
    // ...
}

func (r FooResource) FindAll(req api2go.Request) (api2go.Responder, error) { ... }
func (r FooResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) { ... }
// ...
```

**2. Frontend Types:**
```typescript
// web/src/api/types.ts
export interface FooAttributes { name: string; /* ... */ }
export type Foo = JsonApiResource<'foos', FooAttributes>;
export interface CreateFooRequest { name: string; }
```

**3. Frontend API Client:**
```typescript
// web/src/api/foos.ts
export const foosApi = createResourceApi<Foo, CreateFooRequest>({
  resourceName: 'foos',
});
```

**4. Frontend Hooks:**
```typescript
// web/src/hooks/useFoos.ts
const fooHooks = createResourceHooks({ resourceName: 'foos', api: foosApi });
export const useFoos = fooHooks.useList;
// ...
```

**Total new code for a standard CRUD resource: ~50 lines** (vs ~150+ lines without factories)

### Benefits of This Approach

1. **Spec Alignment**: Both backend and frontend follow JSON:API exactly
2. **Type Safety**: Full TypeScript generics from resource types through to React hooks
3. **Consistency**: Every resource works identically - no special cases
4. **Maintainability**: Bug fixes in factories benefit all resources
5. **Onboarding**: New resources take minutes, not hours
6. **Testing**: Factory logic tested once, all resources inherit correctness

### Technical Debt Prevention

| Debt Pattern | How Factories Prevent It |
|--------------|--------------------------|
| Copy-paste errors | Single source of truth for CRUD logic |
| Inconsistent API calls | Factory enforces JSON:API format |
| Missing cache invalidation | Built into factory mutations |
| Query key mismatches | Keys generated from resourceName |
| Divergent patterns | All resources use same factory |

## References

- JSON:API Specification: https://jsonapi.org
- api2go GitHub: https://github.com/manyminds/api2go
- Gorilla mux: https://github.com/gorilla/mux
- TanStack Query: https://tanstack.com/query
