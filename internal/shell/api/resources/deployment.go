// Package resources provides JSON:API resource implementations for the Hoster API.
// Following ADR-003: JSON:API Standard with api2go
package resources

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	coredeployment "github.com/artpar/hoster/internal/core/deployment"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/validation"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/google/uuid"
	"github.com/manyminds/api2go"
	"github.com/manyminds/api2go/jsonapi"
)

// =============================================================================
// Deployment JSON:API Model
// =============================================================================

// Deployment wraps domain.Deployment to implement JSON:API interfaces.
type Deployment struct {
	ID              string                 `json:"-"`
	Name            string                 `json:"name"`
	TemplateID      string                 `json:"template_id"`
	TemplateVersion string                 `json:"template_version"`
	CustomerID      string                 `json:"customer_id"`
	NodeID          string                 `json:"node_id,omitempty"`
	Status          string                 `json:"status"`
	Variables       map[string]string      `json:"variables,omitempty"`
	Domains         []domain.Domain        `json:"domains,omitempty"`
	Containers      []domain.ContainerInfo `json:"containers,omitempty"`
	Resources       domain.Resources       `json:"resources"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	StoppedAt       *time.Time             `json:"stopped_at,omitempty"`
}

// GetID returns the deployment ID for JSON:API.
func (d Deployment) GetID() string {
	return d.ID
}

// SetID sets the deployment ID for JSON:API.
func (d *Deployment) SetID(id string) error {
	d.ID = id
	return nil
}

// GetName returns the JSON:API resource type name.
func (d Deployment) GetName() string {
	return "deployments"
}

// GetReferences returns the relationships this resource has.
func (d Deployment) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "templates",
			Name: "template",
		},
	}
}

// GetReferencedIDs returns IDs of referenced resources.
func (d Deployment) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{
		{
			ID:   d.TemplateID,
			Type: "templates",
			Name: "template",
		},
	}
}

// GetReferencedStructs returns the actual referenced objects.
func (d Deployment) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

// DeploymentFromDomain converts a domain.Deployment to a JSON:API Deployment.
func DeploymentFromDomain(d *domain.Deployment) Deployment {
	return Deployment{
		ID:              d.ID,
		Name:            d.Name,
		TemplateID:      d.TemplateID,
		TemplateVersion: d.TemplateVersion,
		CustomerID:      d.CustomerID,
		NodeID:          d.NodeID,
		Status:          string(d.Status),
		Variables:       d.Variables,
		Domains:         d.Domains,
		Containers:      d.Containers,
		Resources:       d.Resources,
		ErrorMessage:    d.ErrorMessage,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		StartedAt:       d.StartedAt,
		StoppedAt:       d.StoppedAt,
	}
}

// ToDomain converts the JSON:API Deployment to a domain.Deployment.
func (d Deployment) ToDomain() *domain.Deployment {
	return &domain.Deployment{
		ID:              d.ID,
		Name:            d.Name,
		TemplateID:      d.TemplateID,
		TemplateVersion: d.TemplateVersion,
		CustomerID:      d.CustomerID,
		NodeID:          d.NodeID,
		Status:          domain.DeploymentStatus(d.Status),
		Variables:       d.Variables,
		Domains:         d.Domains,
		Containers:      d.Containers,
		Resources:       d.Resources,
		ErrorMessage:    d.ErrorMessage,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		StartedAt:       d.StartedAt,
		StoppedAt:       d.StoppedAt,
	}
}

// =============================================================================
// DeploymentResource - CRUD Operations
// =============================================================================

// DeploymentResource implements the api2go resource interface for deployments.
type DeploymentResource struct {
	Store        store.Store
	Docker       docker.Client
	Orchestrator *docker.Orchestrator
	Logger       *slog.Logger
	BaseDomain   string
	ConfigDir    string
}

// NewDeploymentResource creates a new deployment resource handler.
func NewDeploymentResource(s store.Store, d docker.Client, l *slog.Logger, baseDomain, configDir string) *DeploymentResource {
	if l == nil {
		l = slog.Default()
	}
	if configDir == "" {
		configDir = "/var/lib/hoster/configs"
	}
	return &DeploymentResource{
		Store:        s,
		Docker:       d,
		Orchestrator: docker.NewOrchestrator(d, l, configDir),
		Logger:       l,
		BaseDomain:   baseDomain,
		ConfigDir:    configDir,
	}
}

// FindAll returns all deployments with optional filtering and pagination.
// GET /api/v1/deployments
func (r DeploymentResource) FindAll(req api2go.Request) (api2go.Responder, error) {
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
	if pageNum, ok := req.QueryParams["page[number]"]; ok && len(pageNum) > 0 {
		if pn, err := strconv.Atoi(pageNum[0]); err == nil && pn > 0 {
			opts.Offset = (pn - 1) * opts.Limit
		}
	}

	ctx := req.PlainRequest.Context()
	var deployments []domain.Deployment
	var err error

	// Filter by template_id or customer_id if provided
	if templateID, ok := req.QueryParams["filter[template_id]"]; ok && len(templateID) > 0 {
		deployments, err = r.Store.ListDeploymentsByTemplate(ctx, templateID[0], opts)
	} else if customerID, ok := req.QueryParams["filter[customer_id]"]; ok && len(customerID) > 0 {
		deployments, err = r.Store.ListDeploymentsByCustomer(ctx, customerID[0], opts)
	} else {
		deployments, err = r.Store.ListDeployments(ctx, opts)
	}

	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	result := make([]Deployment, 0, len(deployments))
	for _, d := range deployments {
		result = append(result, DeploymentFromDomain(&d))
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

// FindOne returns a single deployment by ID.
// GET /api/v1/deployments/{id}
func (r DeploymentResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	deployment, err := r.Store.GetDeployment(ctx, id)
	if err != nil {
		if isDeploymentNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("deployment not found"),
				"Deployment not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusOK,
		Res:  DeploymentFromDomain(deployment),
	}, nil
}

// Create creates a new deployment.
// POST /api/v1/deployments
func (r DeploymentResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	deployment, ok := obj.(Deployment)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"),
			"Invalid request body",
			http.StatusBadRequest,
		)
	}

	// Validate required fields
	if deployment.TemplateID == "" {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("template_id is required"),
			"template_id is required",
			http.StatusBadRequest,
		)
	}
	if deployment.CustomerID == "" {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("customer_id is required"),
			"customer_id is required",
			http.StatusBadRequest,
		)
	}

	ctx := req.PlainRequest.Context()

	// Get template
	template, err := r.Store.GetTemplate(ctx, deployment.TemplateID)
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

	// Check if template is published
	if allowed, reason := validation.CanCreateDeployment(template.Published); !allowed {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("%s", reason),
			reason,
			http.StatusConflict,
		)
	}

	now := time.Now()
	name := deployment.Name
	if name == "" {
		name = template.Name + " Deployment"
	}

	domainDeployment := &domain.Deployment{
		ID:              "depl_" + uuid.New().String()[:8],
		Name:            name,
		TemplateID:      template.ID,
		TemplateVersion: template.Version,
		CustomerID:      deployment.CustomerID,
		Status:          domain.StatusPending,
		Variables:       deployment.Variables,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if domainDeployment.Variables == nil {
		domainDeployment.Variables = make(map[string]string)
	}

	if err := r.Store.CreateDeployment(ctx, domainDeployment); err != nil {
		r.Logger.Error("failed to create deployment", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusCreated,
		Res:  DeploymentFromDomain(domainDeployment),
	}, nil
}

// Delete removes a deployment by ID.
// DELETE /api/v1/deployments/{id}
func (r DeploymentResource) Delete(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()

	deployment, err := r.Store.GetDeployment(ctx, id)
	if err != nil {
		if isDeploymentNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("deployment not found"),
				"Deployment not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Remove all Docker resources
	if err := r.Orchestrator.RemoveDeployment(ctx, deployment); err != nil {
		r.Logger.Warn("failed to remove deployment resources", "error", err)
		// Continue with database deletion even if Docker cleanup fails
	}

	if err := r.Store.DeleteDeployment(ctx, id); err != nil {
		r.Logger.Error("failed to delete deployment", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}

	r.Logger.Info("deployment deleted", "deployment_id", id)

	return &Response{Code: http.StatusNoContent}, nil
}

// =============================================================================
// Custom Actions - Start/Stop
// =============================================================================

// StartDeployment starts a deployment.
// This is a custom action, handled via a separate endpoint.
func (r DeploymentResource) StartDeployment(id string, req *http.Request) (api2go.Responder, error) {
	ctx := req.Context()

	deployment, err := r.Store.GetDeployment(ctx, id)
	if err != nil {
		if isDeploymentNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("deployment not found"),
				"Deployment not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if already running
	if deployment.Status == domain.StatusRunning {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("deployment is already running"),
			"Deployment is already running",
			http.StatusConflict,
		)
	}

	// Get the template
	template, err := r.Store.GetTemplate(ctx, deployment.TemplateID)
	if err != nil {
		r.Logger.Error("failed to get template", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Determine start path using core deployment logic
	startPath := coredeployment.DetermineStartPath(deployment.Status)
	if !startPath.Valid {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("%s", startPath.ErrorReason),
			startPath.ErrorReason,
			http.StatusConflict,
		)
	}

	deployment.NodeID = "local" // For now, all deployments run locally

	// Execute the state transitions
	for _, status := range startPath.Transitions {
		if err := deployment.Transition(status); err != nil {
			r.Logger.Error("failed to transition", "to", status, "error", err)
			return &Response{Code: http.StatusInternalServerError}, err
		}
	}

	// Generate auto domain for the deployment if none exists
	if len(deployment.Domains) == 0 && r.BaseDomain != "" {
		autoDomain := domain.GenerateDomain(deployment.Name, r.BaseDomain)
		deployment.Domains = append(deployment.Domains, autoDomain)
		r.Logger.Info("generated auto domain", "hostname", autoDomain.Hostname)
	}

	if err := r.Store.UpdateDeployment(ctx, deployment); err != nil {
		r.Logger.Error("failed to update deployment status", "error", err)
	}

	// Start containers using orchestrator
	containers, err := r.Orchestrator.StartDeployment(ctx, deployment, template.ComposeSpec, template.ConfigFiles)
	if err != nil {
		r.Logger.Error("failed to start deployment containers", "error", err)
		_ = deployment.TransitionToFailed(err.Error())
		_ = r.Store.UpdateDeployment(ctx, deployment)
		return &Response{Code: http.StatusInternalServerError}, api2go.NewHTTPError(
			err,
			"Failed to start deployment: "+err.Error(),
			http.StatusInternalServerError,
		)
	}

	// Update deployment with container info and transition to running
	deployment.Containers = containers
	if err := deployment.Transition(domain.StatusRunning); err != nil {
		r.Logger.Error("failed to transition to running", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}
	now := time.Now()
	deployment.StartedAt = &now
	deployment.UpdatedAt = now

	if err := r.Store.UpdateDeployment(ctx, deployment); err != nil {
		r.Logger.Error("failed to update deployment", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}

	r.Logger.Info("deployment started",
		"deployment_id", deployment.ID,
		"containers", len(containers),
	)

	return &Response{
		Code: http.StatusOK,
		Res:  DeploymentFromDomain(deployment),
	}, nil
}

// StopDeployment stops a deployment.
// This is a custom action, handled via a separate endpoint.
func (r DeploymentResource) StopDeployment(id string, req *http.Request) (api2go.Responder, error) {
	ctx := req.Context()

	deployment, err := r.Store.GetDeployment(ctx, id)
	if err != nil {
		if isDeploymentNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("deployment not found"),
				"Deployment not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if transition is valid
	if allowed, reason := coredeployment.CanStopDeployment(deployment.Status); !allowed {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			fmt.Errorf("%s", reason),
			reason,
			http.StatusConflict,
		)
	}

	// Transition to stopping
	if err := deployment.Transition(domain.StatusStopping); err != nil {
		r.Logger.Error("failed to transition to stopping", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}
	if err := r.Store.UpdateDeployment(ctx, deployment); err != nil {
		r.Logger.Error("failed to update deployment status", "error", err)
	}

	// Stop containers using orchestrator
	if err := r.Orchestrator.StopDeployment(ctx, deployment); err != nil {
		r.Logger.Error("failed to stop deployment containers", "error", err)
		_ = deployment.TransitionToFailed(err.Error())
		_ = r.Store.UpdateDeployment(ctx, deployment)
		return &Response{Code: http.StatusInternalServerError}, api2go.NewHTTPError(
			err,
			"Failed to stop deployment: "+err.Error(),
			http.StatusInternalServerError,
		)
	}

	// Transition to stopped
	if err := deployment.Transition(domain.StatusStopped); err != nil {
		r.Logger.Error("failed to transition to stopped", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}
	now := time.Now()
	deployment.StoppedAt = &now
	deployment.UpdatedAt = now

	if err := r.Store.UpdateDeployment(ctx, deployment); err != nil {
		r.Logger.Error("failed to update deployment", "error", err)
		return &Response{Code: http.StatusInternalServerError}, err
	}

	r.Logger.Info("deployment stopped", "deployment_id", deployment.ID)

	return &Response{
		Code: http.StatusOK,
		Res:  DeploymentFromDomain(deployment),
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// isDeploymentNotFound checks if an error is a not found error.
func isDeploymentNotFound(err error) bool {
	if err == nil {
		return false
	}
	var storeErr *store.StoreError
	if errors.As(err, &storeErr) {
		return errors.Is(storeErr.Unwrap(), store.ErrNotFound)
	}
	return errors.Is(err, store.ErrNotFound)
}
