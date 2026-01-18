// Package openapi provides reflective OpenAPI 3.0 specification generation.
// Following ADR-004: Reflective OpenAPI Generation
package openapi

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// =============================================================================
// Generator
// =============================================================================

// Generator produces OpenAPI 3.0 specifications by reflecting on registered resources.
type Generator struct {
	title       string
	version     string
	description string
	servers     []string
	resources   []ResourceInfo
	mu          sync.RWMutex
	cachedSpec  *openapi3.T
}

// ResourceInfo holds information about a registered resource for OpenAPI generation.
type ResourceInfo struct {
	Name          string      // Resource type name (e.g., "templates")
	Model         interface{} // The model struct for schema extraction
	SupportsFind  bool        // GET /{type} and GET /{type}/{id}
	SupportsCreate bool        // POST /{type}
	SupportsUpdate bool        // PATCH /{type}/{id}
	SupportsDelete bool        // DELETE /{type}/{id}
}

// Option configures the generator.
type Option func(*Generator)

// WithTitle sets the API title.
func WithTitle(title string) Option {
	return func(g *Generator) {
		g.title = title
	}
}

// WithVersion sets the API version.
func WithVersion(version string) Option {
	return func(g *Generator) {
		g.version = version
	}
}

// WithDescription sets the API description.
func WithDescription(description string) Option {
	return func(g *Generator) {
		g.description = description
	}
}

// WithServer adds a server URL.
func WithServer(url string) Option {
	return func(g *Generator) {
		g.servers = append(g.servers, url)
	}
}

// NewGenerator creates a new OpenAPI generator.
func NewGenerator(opts ...Option) *Generator {
	g := &Generator{
		title:       "Hoster API",
		version:     "1.0.0",
		description: "Deployment marketplace platform API",
		servers:     []string{"http://localhost:9090"},
		resources:   make([]ResourceInfo, 0),
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// RegisterResource adds a resource to the generator for spec generation.
func (g *Generator) RegisterResource(info ResourceInfo) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resources = append(g.resources, info)
	g.cachedSpec = nil // Invalidate cache
}

// Generate produces the complete OpenAPI 3.0 specification.
func (g *Generator) Generate() *openapi3.T {
	g.mu.RLock()
	if g.cachedSpec != nil {
		spec := g.cachedSpec
		g.mu.RUnlock()
		return spec
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()

	// Double-check after acquiring write lock
	if g.cachedSpec != nil {
		return g.cachedSpec
	}

	spec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       g.title,
			Version:     g.version,
			Description: g.description,
		},
		Servers: make(openapi3.Servers, 0, len(g.servers)),
		Paths:   &openapi3.Paths{},
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}

	// Add servers
	for _, url := range g.servers {
		spec.Servers = append(spec.Servers, &openapi3.Server{URL: url})
	}

	// Add common schemas
	g.addCommonSchemas(spec)

	// Process each registered resource
	for _, res := range g.resources {
		g.addResourceToSpec(spec, res)
	}

	g.cachedSpec = spec
	return spec
}

// Handler returns an HTTP handler that serves the OpenAPI specification.
func (g *Generator) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spec := g.Generate()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		if err := json.NewEncoder(w).Encode(spec); err != nil {
			http.Error(w, "Failed to encode OpenAPI spec", http.StatusInternalServerError)
		}
	}
}

// =============================================================================
// Schema Generation
// =============================================================================

// addCommonSchemas adds common JSON:API schemas to the spec.
func (g *Generator) addCommonSchemas(spec *openapi3.T) {
	// Links schema
	spec.Components.Schemas["Links"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"self": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
				"related": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
			},
		},
	}

	// Pagination links schema
	spec.Components.Schemas["PaginationLinks"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"self": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
				"first": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
				"last": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
				"prev": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
				"next": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "uri"},
				},
			},
		},
	}

	// Pagination meta schema
	spec.Components.Schemas["PaginationMeta"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"total": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}},
				},
				"limit": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}},
				},
				"offset": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}},
				},
			},
		},
	}

	// Error schema
	spec.Components.Schemas["Error"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"errors": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"array"},
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
								Properties: openapi3.Schemas{
									"status": &openapi3.SchemaRef{
										Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
									},
									"title": &openapi3.SchemaRef{
										Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
									},
									"detail": &openapi3.SchemaRef{
										Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// addResourceToSpec adds paths and schemas for a resource.
func (g *Generator) addResourceToSpec(spec *openapi3.T, res ResourceInfo) {
	basePath := "/api/v1/" + res.Name

	// Extract schema from model
	attributesSchema := g.extractSchema(res.Model)
	schemaName := capitalize(singularize(res.Name))

	// Add attributes schema
	spec.Components.Schemas[schemaName+"Attributes"] = attributesSchema

	// Add JSON:API resource schema
	spec.Components.Schemas[schemaName] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"type": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
						Enum: []interface{}{res.Name},
					},
				},
				"id": &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
				},
				"attributes": &openapi3.SchemaRef{
					Ref: "#/components/schemas/" + schemaName + "Attributes",
				},
			},
			Required: []string{"type", "id"},
		},
	}

	// Add response schemas
	spec.Components.Schemas[schemaName+"Response"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"data": &openapi3.SchemaRef{
					Ref: "#/components/schemas/" + schemaName,
				},
				"links": &openapi3.SchemaRef{
					Ref: "#/components/schemas/Links",
				},
			},
		},
	}

	spec.Components.Schemas[schemaName+"ListResponse"] = &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"object"},
			Properties: openapi3.Schemas{
				"data": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"array"},
						Items: &openapi3.SchemaRef{
							Ref: "#/components/schemas/" + schemaName,
						},
					},
				},
				"links": &openapi3.SchemaRef{
					Ref: "#/components/schemas/PaginationLinks",
				},
				"meta": &openapi3.SchemaRef{
					Ref: "#/components/schemas/PaginationMeta",
				},
			},
		},
	}

	// Add collection path
	collectionPath := &openapi3.PathItem{}

	if res.SupportsFind {
		collectionPath.Get = g.createListOperation(res, schemaName)
	}
	if res.SupportsCreate {
		collectionPath.Post = g.createCreateOperation(res, schemaName)
	}

	spec.Paths.Set(basePath, collectionPath)

	// Add item path
	itemPath := &openapi3.PathItem{
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:     "id",
					In:       "path",
					Required: true,
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
					},
				},
			},
		},
	}

	if res.SupportsFind {
		itemPath.Get = g.createGetOperation(res, schemaName)
	}
	if res.SupportsUpdate {
		itemPath.Patch = g.createUpdateOperation(res, schemaName)
	}
	if res.SupportsDelete {
		itemPath.Delete = g.createDeleteOperation(res, schemaName)
	}

	spec.Paths.Set(basePath+"/{id}", itemPath)
}

// extractSchema extracts an OpenAPI schema from a Go struct.
func (g *Generator) extractSchema(model interface{}) *openapi3.SchemaRef {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &openapi3.Schema{
		Type:       &openapi3.Types{"object"},
		Properties: make(openapi3.Schemas),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Parse JSON tag for name
		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
		}

		// Convert Go type to OpenAPI type
		propSchema := g.goTypeToSchema(field.Type)
		if propSchema != nil {
			schema.Properties[name] = propSchema
		}
	}

	return &openapi3.SchemaRef{Value: schema}
}

// goTypeToSchema converts a Go type to an OpenAPI schema.
func (g *Generator) goTypeToSchema(t reflect.Type) *openapi3.SchemaRef {
	switch t.Kind() {
	case reflect.String:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}}}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}, Format: "int32"}}

	case reflect.Int64:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}, Format: "int64"}}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}}}

	case reflect.Float32:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"number"}, Format: "float"}}

	case reflect.Float64:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"number"}, Format: "double"}}

	case reflect.Bool:
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"boolean"}}}

	case reflect.Slice, reflect.Array:
		elemSchema := g.goTypeToSchema(t.Elem())
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:  &openapi3.Types{"array"},
				Items: elemSchema,
			},
		}

	case reflect.Map:
		valueSchema := g.goTypeToSchema(t.Elem())
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:                 &openapi3.Types{"object"},
				AdditionalProperties: openapi3.AdditionalProperties{Schema: valueSchema},
			},
		}

	case reflect.Ptr:
		schema := g.goTypeToSchema(t.Elem())
		if schema != nil && schema.Value != nil {
			schema.Value.Nullable = true
		}
		return schema

	case reflect.Struct:
		// Handle time.Time specially
		if t == reflect.TypeOf(time.Time{}) {
			return &openapi3.SchemaRef{
				Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "date-time"},
			}
		}
		// For other structs, extract recursively
		return g.extractSchema(reflect.New(t).Interface())

	default:
		// Unknown type, return generic object
		return &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"object"}}}
	}
}

// =============================================================================
// Operation Generation
// =============================================================================

func (g *Generator) createListOperation(res ResourceInfo, schemaName string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "list" + capitalize(res.Name),
		Summary:     "List " + res.Name,
		Tags:        []string{capitalize(res.Name)},
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name: "page[size]",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}, Default: 20},
					},
				},
			},
			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name: "page[number]",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: &openapi3.Types{"integer"}, Default: 1},
					},
				},
			},
		},
		Responses: &openapi3.Responses{},
	}
}

func (g *Generator) createGetOperation(res ResourceInfo, schemaName string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "get" + schemaName,
		Summary:     "Get a " + singularize(res.Name),
		Tags:        []string{capitalize(res.Name)},
		Responses:   &openapi3.Responses{},
	}
}

func (g *Generator) createCreateOperation(res ResourceInfo, schemaName string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "create" + schemaName,
		Summary:     "Create a " + singularize(res.Name),
		Tags:        []string{capitalize(res.Name)},
		RequestBody: &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content: openapi3.Content{
					"application/vnd.api+json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/" + schemaName + "Response",
						},
					},
				},
			},
		},
		Responses: &openapi3.Responses{},
	}
}

func (g *Generator) createUpdateOperation(res ResourceInfo, schemaName string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "update" + schemaName,
		Summary:     "Update a " + singularize(res.Name),
		Tags:        []string{capitalize(res.Name)},
		RequestBody: &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required: true,
				Content: openapi3.Content{
					"application/vnd.api+json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{
							Ref: "#/components/schemas/" + schemaName + "Response",
						},
					},
				},
			},
		},
		Responses: &openapi3.Responses{},
	}
}

func (g *Generator) createDeleteOperation(res ResourceInfo, schemaName string) *openapi3.Operation {
	return &openapi3.Operation{
		OperationID: "delete" + schemaName,
		Summary:     "Delete a " + singularize(res.Name),
		Tags:        []string{capitalize(res.Name)},
		Responses:   &openapi3.Responses{},
	}
}

// =============================================================================
// Helpers
// =============================================================================

// capitalize returns the string with the first letter capitalized.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// singularize performs basic singularization (removes trailing 's').
func singularize(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
