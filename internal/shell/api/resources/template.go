// Package resources provides JSON:API resource implementations for the Hoster API.
// Following ADR-003: JSON:API Standard with api2go
package resources

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/validation"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/google/uuid"
	"github.com/manyminds/api2go"
	"github.com/manyminds/api2go/jsonapi"
)

// =============================================================================
// Template JSON:API Model
// =============================================================================

// Template wraps domain.Template to implement JSON:API interfaces.
// It provides the marshaling/unmarshaling for the JSON:API format.
type Template struct {
	ID                   string               `json:"-"`
	Name                 string               `json:"name"`
	Slug                 string               `json:"slug"`
	Description          string               `json:"description,omitempty"`
	Version              string               `json:"version"`
	ComposeSpec          string               `json:"compose_spec"`
	Variables            []domain.Variable    `json:"variables,omitempty"`
	ConfigFiles          []domain.ConfigFile  `json:"config_files,omitempty"`
	ResourceRequirements domain.Resources     `json:"resource_requirements"`
	PriceMonthly         int64                `json:"price_monthly_cents"`
	Category             string               `json:"category,omitempty"`
	Tags                 []string             `json:"tags,omitempty"`
	Published            bool                 `json:"published"`
	CreatorID            string               `json:"creator_id"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

// GetID returns the template ID for JSON:API.
func (t Template) GetID() string {
	return t.ID
}

// SetID sets the template ID for JSON:API.
func (t *Template) SetID(id string) error {
	t.ID = id
	return nil
}

// GetName returns the JSON:API resource type name.
func (t Template) GetName() string {
	return "templates"
}

// GetReferences returns the relationships this resource has.
func (t Template) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "deployments",
			Name: "deployments",
		},
	}
}

// GetReferencedIDs returns IDs of referenced resources (for relationship links).
func (t Template) GetReferencedIDs() []jsonapi.ReferenceID {
	// We don't eagerly load deployment IDs - use relationship endpoint instead
	return nil
}

// GetReferencedStructs returns the actual referenced objects for compound documents.
func (t Template) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	// We don't include deployments by default - use ?include=deployments
	return nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

// TemplateFromDomain converts a domain.Template to a JSON:API Template.
func TemplateFromDomain(t *domain.Template) Template {
	return Template{
		ID:                   t.ID,
		Name:                 t.Name,
		Slug:                 t.Slug,
		Description:          t.Description,
		Version:              t.Version,
		ComposeSpec:          t.ComposeSpec,
		Variables:            t.Variables,
		ConfigFiles:          t.ConfigFiles,
		ResourceRequirements: t.ResourceRequirements,
		PriceMonthly:         t.PriceMonthly,
		Category:             t.Category,
		Tags:                 t.Tags,
		Published:            t.Published,
		CreatorID:            t.CreatorID,
		CreatedAt:            t.CreatedAt,
		UpdatedAt:            t.UpdatedAt,
	}
}

// ToDomain converts the JSON:API Template to a domain.Template.
func (t Template) ToDomain() *domain.Template {
	return &domain.Template{
		ID:                   t.ID,
		Name:                 t.Name,
		Slug:                 t.Slug,
		Description:          t.Description,
		Version:              t.Version,
		ComposeSpec:          t.ComposeSpec,
		Variables:            t.Variables,
		ConfigFiles:          t.ConfigFiles,
		ResourceRequirements: t.ResourceRequirements,
		PriceMonthly:         t.PriceMonthly,
		Category:             t.Category,
		Tags:                 t.Tags,
		Published:            t.Published,
		CreatorID:            t.CreatorID,
		CreatedAt:            t.CreatedAt,
		UpdatedAt:            t.UpdatedAt,
	}
}

// =============================================================================
// TemplateResource - CRUD Operations
// =============================================================================

// TemplateResource implements the api2go resource interface for templates.
type TemplateResource struct {
	Store store.Store
}

// NewTemplateResource creates a new template resource handler.
func NewTemplateResource(s store.Store) *TemplateResource {
	return &TemplateResource{Store: s}
}

// FindAll returns all templates with optional pagination.
// GET /api/v1/templates
func (r TemplateResource) FindAll(req api2go.Request) (api2go.Responder, error) {
	opts := store.DefaultListOptions()

	// Parse pagination from query params
	if limit, ok := req.QueryParams["page[size]"]; ok && len(limit) > 0 {
		if l, err := strconv.Atoi(limit[0]); err == nil {
			opts.Limit = l
		}
	}
	if offset, ok := req.QueryParams["page[offset]"]; ok && len(offset) > 0 {
		if o, err := strconv.Atoi(offset[0]); err == nil {
			opts.Offset = o
		}
	}
	// Also support page[number] style
	if pageNum, ok := req.QueryParams["page[number]"]; ok && len(pageNum) > 0 {
		if pn, err := strconv.Atoi(pageNum[0]); err == nil && pn > 0 {
			opts.Offset = (pn - 1) * opts.Limit
		}
	}

	ctx := req.PlainRequest.Context()
	templates, err := r.Store.ListTemplates(ctx, opts)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	result := make([]Template, 0, len(templates))
	for _, t := range templates {
		result = append(result, TemplateFromDomain(&t))
	}

	return &Response{
		Code: http.StatusOK,
		Res:  result,
		Meta: map[string]interface{}{
			"total":  len(result),
			"limit":  opts.Limit,
			"offset": opts.Offset,
		},
	}, nil
}

// FindOne returns a single template by ID.
// GET /api/v1/templates/{id}
func (r TemplateResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	template, err := r.Store.GetTemplate(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("template not found"),
				"Template not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusOK,
		Res:  TemplateFromDomain(template),
	}, nil
}

// Create creates a new template.
// POST /api/v1/templates
func (r TemplateResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	template, ok := obj.(Template)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"),
			"Invalid request body",
			http.StatusBadRequest,
		)
	}

	// Validate required fields using core validation
	if field, msg := validation.ValidateCreateTemplateFields(template.Name, template.Version, template.ComposeSpec, template.CreatorID); field != "" {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("%s", msg),
			msg,
			http.StatusBadRequest,
		)
	}

	now := time.Now()
	domainTemplate := &domain.Template{
		ID:                   "tmpl_" + uuid.New().String()[:8],
		Name:                 template.Name,
		Slug:                 domain.Slugify(template.Name),
		Version:              template.Version,
		ComposeSpec:          template.ComposeSpec,
		CreatorID:            template.CreatorID,
		Description:          template.Description,
		Category:             template.Category,
		Tags:                 template.Tags,
		Variables:            template.Variables,
		ConfigFiles:          template.ConfigFiles,
		ResourceRequirements: template.ResourceRequirements,
		PriceMonthly:         template.PriceMonthly,
		Published:            false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	ctx := req.PlainRequest.Context()
	if err := r.Store.CreateTemplate(ctx, domainTemplate); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusCreated,
		Res:  TemplateFromDomain(domainTemplate),
	}, nil
}

// Update updates an existing template.
// PATCH /api/v1/templates/{id}
func (r TemplateResource) Update(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	template, ok := obj.(Template)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"),
			"Invalid request body",
			http.StatusBadRequest,
		)
	}

	ctx := req.PlainRequest.Context()
	existing, err := r.Store.GetTemplate(ctx, template.ID)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("template not found"),
				"Template not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Can't update published templates
	if allowed, reason := validation.CanUpdateTemplate(existing.Published); !allowed {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("%s", reason),
			reason,
			http.StatusConflict,
		)
	}

	// Apply updates (only non-empty fields)
	if template.Name != "" {
		existing.Name = template.Name
	}
	if template.Description != "" {
		existing.Description = template.Description
	}
	if template.Category != "" {
		existing.Category = template.Category
	}
	if len(template.Tags) > 0 {
		existing.Tags = template.Tags
	}
	if template.PriceMonthly > 0 {
		existing.PriceMonthly = template.PriceMonthly
	}
	if len(template.ConfigFiles) > 0 {
		existing.ConfigFiles = template.ConfigFiles
	}
	existing.UpdatedAt = time.Now()

	if err := r.Store.UpdateTemplate(ctx, existing); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusOK,
		Res:  TemplateFromDomain(existing),
	}, nil
}

// Delete removes a template by ID.
// DELETE /api/v1/templates/{id}
func (r TemplateResource) Delete(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()

	// Check if template exists
	_, err := r.Store.GetTemplate(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("template not found"),
				"Template not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check for active deployments
	deployments, err := r.Store.ListDeploymentsByTemplate(ctx, id, store.ListOptions{Limit: 1})
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}
	if len(deployments) > 0 {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("template has active deployments"),
			"Template has active deployments",
			http.StatusConflict,
		)
	}

	if err := r.Store.DeleteTemplate(ctx, id); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusNoContent}, nil
}

// =============================================================================
// Custom Actions - Publish
// =============================================================================

// PublishTemplate publishes a template.
// This is a custom action, handled via a separate endpoint.
func (r TemplateResource) PublishTemplate(id string, ctx *http.Request) (api2go.Responder, error) {
	template, err := r.Store.GetTemplate(ctx.Context(), id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("template not found"),
				"Template not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if template.Published {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("template is already published"),
			"Template is already published",
			http.StatusConflict,
		)
	}

	template.Published = true
	template.UpdatedAt = time.Now()

	if err := r.Store.UpdateTemplate(ctx.Context(), template); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusOK,
		Res:  TemplateFromDomain(template),
	}, nil
}

// =============================================================================
// Response Helper
// =============================================================================

// Response implements api2go.Responder for custom responses.
type Response struct {
	Code int
	Res  interface{}
	Meta map[string]interface{}
}

// Metadata returns additional metadata for the response.
func (r *Response) Metadata() map[string]interface{} {
	return r.Meta
}

// Result returns the response data.
func (r *Response) Result() interface{} {
	return r.Res
}

// StatusCode returns the HTTP status code.
func (r *Response) StatusCode() int {
	return r.Code
}

// =============================================================================
// Helper Functions
// =============================================================================

// isNotFound checks if an error is a not found error.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Check if it's a StoreError wrapping ErrNotFound
	var storeErr *store.StoreError
	if errors.As(err, &storeErr) {
		return errors.Is(storeErr.Unwrap(), store.ErrNotFound)
	}
	return errors.Is(err, store.ErrNotFound)
}
