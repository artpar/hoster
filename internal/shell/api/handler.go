// Package api provides HTTP handlers for the Hoster API.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	coredeployment "github.com/artpar/hoster/internal/core/deployment"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/validation"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/scheduler"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// =============================================================================
// Handler
// =============================================================================

// Handler provides HTTP handlers for the API.
type Handler struct {
	store        store.Store
	docker       docker.Client
	orchestrator *docker.Orchestrator
	scheduler    *scheduler.Service
	logger       *slog.Logger
	baseDomain   string
	configDir    string
}

// NewHandler creates a new API handler.
// configDir is the base directory for storing deployment config files.
func NewHandler(s store.Store, d docker.Client, l *slog.Logger, baseDomain, configDir string) *Handler {
	if l == nil {
		l = slog.Default()
	}
	if configDir == "" {
		configDir = "/var/lib/hoster/configs"
	}
	return &Handler{
		store:        s,
		docker:       d,
		orchestrator: docker.NewOrchestrator(d, l, configDir),
		scheduler:    scheduler.NewService(s, nil, d, l), // nil NodePool for backward compat
		logger:       l,
		baseDomain:   baseDomain,
		configDir:    configDir,
	}
}

// NewHandlerWithScheduler creates a new API handler with a custom scheduler service.
// Use this when you need remote node support via NodePool.
func NewHandlerWithScheduler(s store.Store, d docker.Client, sched *scheduler.Service, l *slog.Logger, baseDomain, configDir string) *Handler {
	if l == nil {
		l = slog.Default()
	}
	if configDir == "" {
		configDir = "/var/lib/hoster/configs"
	}
	if sched == nil {
		sched = scheduler.NewService(s, nil, d, l)
	}
	return &Handler{
		store:        s,
		docker:       d,
		orchestrator: docker.NewOrchestrator(d, l, configDir), // Default orchestrator with local client
		scheduler:    sched,
		logger:       l,
		baseDomain:   baseDomain,
		configDir:    configDir,
	}
}

// Routes returns the router with all routes configured.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(h.jsonContentType)
	r.Use(h.requestIDHeader)

	// Health endpoints
	r.Get("/health", h.handleHealth)
	r.Get("/ready", h.handleReady)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Template routes
		r.Route("/templates", func(r chi.Router) {
			r.Post("/", h.handleCreateTemplate)
			r.Get("/", h.handleListTemplates)
			r.Get("/{id}", h.handleGetTemplate)
			r.Put("/{id}", h.handleUpdateTemplate)
			r.Delete("/{id}", h.handleDeleteTemplate)
			r.Post("/{id}/publish", h.handlePublishTemplate)
		})

		// Deployment routes
		r.Route("/deployments", func(r chi.Router) {
			r.Post("/", h.handleCreateDeployment)
			r.Get("/", h.handleListDeployments)
			r.Get("/{id}", h.handleGetDeployment)
			r.Delete("/{id}", h.handleDeleteDeployment)
			r.Post("/{id}/start", h.handleStartDeployment)
			r.Post("/{id}/stop", h.handleStopDeployment)
		})
	})

	return r
}

// =============================================================================
// Middleware
// =============================================================================

// jsonContentType sets Content-Type header to application/json.
func (h *Handler) jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// requestIDHeader copies the request ID to the response header.
func (h *Handler) requestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			w.Header().Set("X-Request-ID", reqID)
		}
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// Health Handlers
// =============================================================================

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, HealthResponse{Status: "healthy"})
}

func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)

	// Check database (implicit - if we got here, store was created)
	checks["database"] = "ok"

	// Check Docker
	if err := h.docker.Ping(); err != nil {
		checks["docker"] = "failed"
		h.writeJSON(w, http.StatusServiceUnavailable, ReadyResponse{
			Status: "not_ready",
			Checks: checks,
		})
		return
	}
	checks["docker"] = "ok"

	h.writeJSON(w, http.StatusOK, ReadyResponse{
		Status: "ready",
		Checks: checks,
	})
}

// =============================================================================
// Template Handlers
// =============================================================================

func (h *Handler) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON", "validation_error")
		return
	}

	// Validate required fields using core validation
	if field, msg := validation.ValidateCreateTemplateFields(req.Name, req.Version, req.ComposeSpec, req.CreatorID); field != "" {
		h.writeError(w, http.StatusBadRequest, msg, "validation_error")
		return
	}

	now := time.Now()
	template := &domain.Template{
		ID:           "tmpl_" + uuid.New().String()[:8],
		Name:         req.Name,
		Slug:         domain.Slugify(req.Name),
		Version:      req.Version,
		ComposeSpec:  req.ComposeSpec,
		CreatorID:    req.CreatorID,
		Description:  req.Description,
		Category:     req.Category,
		Tags:         req.Tags,
		PriceMonthly: int64(req.PriceMonthly),
		Published:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Convert variables
	for _, v := range req.Variables {
		template.Variables = append(template.Variables, domain.Variable{
			Name:        v.Name,
			Description: v.Description,
			Type:        domain.VariableType(v.Type),
			Default:     v.Default,
			Required:    v.Required,
		})
	}

	// Convert config files
	for _, cf := range req.ConfigFiles {
		template.ConfigFiles = append(template.ConfigFiles, domain.ConfigFile{
			Name:    cf.Name,
			Path:    cf.Path,
			Content: cf.Content,
			Mode:    cf.Mode,
		})
	}

	if err := h.store.CreateTemplate(r.Context(), template); err != nil {
		h.logger.Error("failed to create template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create template", "internal_error")
		return
	}

	h.writeJSON(w, http.StatusCreated, h.templateToResponse(template))
}

func (h *Handler) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	template, err := h.store.GetTemplate(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "template not found", "template_not_found")
			return
		}
		h.logger.Error("failed to get template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get template", "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, h.templateToResponse(template))
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	opts := store.DefaultListOptions()

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}

	templates, err := h.store.ListTemplates(r.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list templates", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to list templates", "internal_error")
		return
	}

	resp := ListTemplatesResponse{
		Templates: make([]TemplateResponse, 0, len(templates)),
		Total:     len(templates),
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	}
	for _, t := range templates {
		resp.Templates = append(resp.Templates, h.templateToResponse(&t))
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	template, err := h.store.GetTemplate(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "template not found", "template_not_found")
			return
		}
		h.logger.Error("failed to get template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get template", "internal_error")
		return
	}

	// Can't update published templates
	if allowed, reason := validation.CanUpdateTemplate(template.Published); !allowed {
		h.writeError(w, http.StatusConflict, reason, "template_published")
		return
	}

	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON", "validation_error")
		return
	}

	// Apply updates
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Category != "" {
		template.Category = req.Category
	}
	if len(req.Tags) > 0 {
		template.Tags = req.Tags
	}
	if req.PriceMonthly > 0 {
		template.PriceMonthly = int64(req.PriceMonthly)
	}
	if len(req.ConfigFiles) > 0 {
		template.ConfigFiles = nil // Reset to replace
		for _, cf := range req.ConfigFiles {
			template.ConfigFiles = append(template.ConfigFiles, domain.ConfigFile{
				Name:    cf.Name,
				Path:    cf.Path,
				Content: cf.Content,
				Mode:    cf.Mode,
			})
		}
	}
	template.UpdatedAt = time.Now()

	if err := h.store.UpdateTemplate(r.Context(), template); err != nil {
		h.logger.Error("failed to update template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to update template", "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, h.templateToResponse(template))
}

func (h *Handler) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if template exists
	_, err := h.store.GetTemplate(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "template not found", "template_not_found")
			return
		}
		h.logger.Error("failed to get template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get template", "internal_error")
		return
	}

	// Check for active deployments
	deployments, err := h.store.ListDeploymentsByTemplate(r.Context(), id, store.ListOptions{Limit: 1})
	if err != nil {
		h.logger.Error("failed to check deployments", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to check deployments", "internal_error")
		return
	}
	if len(deployments) > 0 {
		h.writeError(w, http.StatusConflict, "template has active deployments", "template_in_use")
		return
	}

	if err := h.store.DeleteTemplate(r.Context(), id); err != nil {
		h.logger.Error("failed to delete template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to delete template", "internal_error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePublishTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	template, err := h.store.GetTemplate(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "template not found", "template_not_found")
			return
		}
		h.logger.Error("failed to get template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get template", "internal_error")
		return
	}

	if template.Published {
		h.writeError(w, http.StatusConflict, "template is already published", "already_published")
		return
	}

	template.Published = true
	template.UpdatedAt = time.Now()

	if err := h.store.UpdateTemplate(r.Context(), template); err != nil {
		h.logger.Error("failed to publish template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to publish template", "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, h.templateToResponse(template))
}

// =============================================================================
// Deployment Handlers
// =============================================================================

func (h *Handler) handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req CreateDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON", "validation_error")
		return
	}

	// Validate required fields
	if req.TemplateID == "" {
		h.writeError(w, http.StatusBadRequest, "template_id is required", "validation_error")
		return
	}
	if req.CustomerID == "" {
		h.writeError(w, http.StatusBadRequest, "customer_id is required", "validation_error")
		return
	}

	// Get template
	template, err := h.store.GetTemplate(r.Context(), req.TemplateID)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "template not found", "template_not_found")
			return
		}
		h.logger.Error("failed to get template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get template", "internal_error")
		return
	}

	// Check if template is published
	if allowed, reason := validation.CanCreateDeployment(template.Published); !allowed {
		h.writeError(w, http.StatusConflict, reason, "template_not_published")
		return
	}

	now := time.Now()
	name := req.Name
	if name == "" {
		name = template.Name + " Deployment"
	}

	deployment := &domain.Deployment{
		ID:              "depl_" + uuid.New().String()[:8],
		Name:            name,
		TemplateID:      template.ID,
		TemplateVersion: template.Version,
		CustomerID:      req.CustomerID,
		Status:          domain.StatusPending,
		Variables:       req.Variables,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if deployment.Variables == nil {
		deployment.Variables = make(map[string]string)
	}

	if err := h.store.CreateDeployment(r.Context(), deployment); err != nil {
		h.logger.Error("failed to create deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create deployment", "internal_error")
		return
	}

	h.writeJSON(w, http.StatusCreated, h.deploymentToResponse(deployment))
}

func (h *Handler) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	deployment, err := h.store.GetDeployment(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "deployment not found", "deployment_not_found")
			return
		}
		h.logger.Error("failed to get deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get deployment", "internal_error")
		return
	}

	h.writeJSON(w, http.StatusOK, h.deploymentToResponse(deployment))
}

func (h *Handler) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	opts := store.DefaultListOptions()

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}

	var deployments []domain.Deployment
	var err error

	// Filter by template or customer if provided
	if templateID := r.URL.Query().Get("template_id"); templateID != "" {
		deployments, err = h.store.ListDeploymentsByTemplate(r.Context(), templateID, opts)
	} else if customerID := r.URL.Query().Get("customer_id"); customerID != "" {
		deployments, err = h.store.ListDeploymentsByCustomer(r.Context(), customerID, opts)
	} else {
		deployments, err = h.store.ListDeployments(r.Context(), opts)
	}

	if err != nil {
		h.logger.Error("failed to list deployments", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to list deployments", "internal_error")
		return
	}

	resp := ListDeploymentsResponse{
		Deployments: make([]DeploymentResponse, 0, len(deployments)),
		Total:       len(deployments),
		Limit:       opts.Limit,
		Offset:      opts.Offset,
	}
	for _, d := range deployments {
		resp.Deployments = append(resp.Deployments, h.deploymentToResponse(&d))
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Check if deployment exists
	deployment, err := h.store.GetDeployment(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "deployment not found", "deployment_not_found")
			return
		}
		h.logger.Error("failed to get deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get deployment", "internal_error")
		return
	}

	// Get Docker client for the deployment's node
	client, err := h.scheduler.GetClientForNode(r.Context(), deployment.NodeID)
	if err != nil {
		h.logger.Warn("failed to get client for node, attempting local cleanup",
			"node_id", deployment.NodeID, "error", err)
		// Fall back to local client for cleanup attempt
		client = h.docker
	}

	// Create orchestrator with the node's client
	orchestrator := docker.NewOrchestrator(client, h.logger, h.configDir)

	// Remove all Docker resources (containers, network, volumes)
	if err := orchestrator.RemoveDeployment(r.Context(), deployment); err != nil {
		h.logger.Warn("failed to remove deployment resources", "error", err)
		// Continue with database deletion even if Docker cleanup fails
	}

	if err := h.store.DeleteDeployment(r.Context(), id); err != nil {
		h.logger.Error("failed to delete deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to delete deployment", "internal_error")
		return
	}

	h.logger.Info("deployment deleted", "deployment_id", id, "node_id", deployment.NodeID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleStartDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	deployment, err := h.store.GetDeployment(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "deployment not found", "deployment_not_found")
			return
		}
		h.logger.Error("failed to get deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get deployment", "internal_error")
		return
	}

	// Check if already running
	if deployment.Status == domain.StatusRunning {
		h.writeError(w, http.StatusConflict, "deployment is already running", "already_running")
		return
	}

	// Get the template to fetch compose spec
	template, err := h.store.GetTemplate(r.Context(), deployment.TemplateID)
	if err != nil {
		h.logger.Error("failed to get template", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get template", "internal_error")
		return
	}

	// Determine start path using core deployment logic
	startPath := coredeployment.DetermineStartPath(deployment.Status)
	if !startPath.Valid {
		h.writeError(w, http.StatusConflict, startPath.ErrorReason, "invalid_transition")
		return
	}

	// Schedule deployment to a node
	// If deployment already has a NodeID (restart case), try to use the same node
	schedReq := scheduler.ScheduleDeploymentRequest{
		Template:        template,
		CreatorID:       template.CreatorID,
		PreferredNodeID: deployment.NodeID, // Use existing node for restarts
	}

	schedResult, err := h.scheduler.ScheduleDeployment(r.Context(), schedReq)
	if err != nil {
		h.logger.Error("failed to schedule deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to schedule deployment: "+err.Error(), "scheduling_error")
		return
	}

	deployment.NodeID = schedResult.NodeID
	h.logger.Info("scheduled deployment",
		"deployment_id", deployment.ID,
		"node_id", schedResult.NodeID,
		"is_local", schedResult.IsLocal,
		"score", schedResult.Score,
	)

	// Execute the state transitions
	for _, status := range startPath.Transitions {
		if err := deployment.Transition(status); err != nil {
			h.logger.Error("failed to transition", "to", status, "error", err)
			h.writeError(w, http.StatusInternalServerError, "failed to start deployment", "internal_error")
			return
		}
	}

	// Generate auto domain for the deployment if none exists
	if len(deployment.Domains) == 0 && h.baseDomain != "" {
		autoDomain := domain.GenerateDomain(deployment.Name, h.baseDomain)
		deployment.Domains = append(deployment.Domains, autoDomain)
		h.logger.Info("generated auto domain", "hostname", autoDomain.Hostname)
	}

	if err := h.store.UpdateDeployment(r.Context(), deployment); err != nil {
		h.logger.Error("failed to update deployment status", "error", err)
	}

	// Create orchestrator with the scheduled node's client
	orchestrator := docker.NewOrchestrator(schedResult.Client, h.logger, h.configDir)

	// Start containers using orchestrator
	containers, err := orchestrator.StartDeployment(r.Context(), deployment, template.ComposeSpec, template.ConfigFiles)
	if err != nil {
		h.logger.Error("failed to start deployment containers", "error", err)
		// Transition to failed
		_ = deployment.TransitionToFailed(err.Error())
		_ = h.store.UpdateDeployment(r.Context(), deployment)
		h.writeError(w, http.StatusInternalServerError, "failed to start deployment: "+err.Error(), "container_error")
		return
	}

	// Update deployment with container info and transition to running
	deployment.Containers = containers
	if err := deployment.Transition(domain.StatusRunning); err != nil {
		h.logger.Error("failed to transition to running", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to start deployment", "internal_error")
		return
	}
	now := time.Now()
	deployment.StartedAt = &now
	deployment.UpdatedAt = now

	if err := h.store.UpdateDeployment(r.Context(), deployment); err != nil {
		h.logger.Error("failed to update deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to update deployment", "internal_error")
		return
	}

	h.logger.Info("deployment started",
		"deployment_id", deployment.ID,
		"node_id", deployment.NodeID,
		"containers", len(containers),
	)

	h.writeJSON(w, http.StatusOK, h.deploymentToResponse(deployment))
}

func (h *Handler) handleStopDeployment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	deployment, err := h.store.GetDeployment(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			h.writeError(w, http.StatusNotFound, "deployment not found", "deployment_not_found")
			return
		}
		h.logger.Error("failed to get deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get deployment", "internal_error")
		return
	}

	// Check if transition is valid using core deployment logic
	if allowed, reason := coredeployment.CanStopDeployment(deployment.Status); !allowed {
		h.writeError(w, http.StatusConflict, reason, "invalid_transition")
		return
	}

	// Get Docker client for the deployment's node
	client, err := h.scheduler.GetClientForNode(r.Context(), deployment.NodeID)
	if err != nil {
		h.logger.Error("failed to get client for node", "node_id", deployment.NodeID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to connect to deployment node", "node_error")
		return
	}

	// Transition to stopping
	if err := deployment.Transition(domain.StatusStopping); err != nil {
		h.logger.Error("failed to transition to stopping", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to stop deployment", "internal_error")
		return
	}
	if err := h.store.UpdateDeployment(r.Context(), deployment); err != nil {
		h.logger.Error("failed to update deployment status", "error", err)
	}

	// Create orchestrator with the node's client
	orchestrator := docker.NewOrchestrator(client, h.logger, h.configDir)

	// Stop containers using orchestrator
	if err := orchestrator.StopDeployment(r.Context(), deployment); err != nil {
		h.logger.Error("failed to stop deployment containers", "error", err)
		// Transition to failed
		_ = deployment.TransitionToFailed(err.Error())
		_ = h.store.UpdateDeployment(r.Context(), deployment)
		h.writeError(w, http.StatusInternalServerError, "failed to stop deployment: "+err.Error(), "container_error")
		return
	}

	// Transition to stopped
	if err := deployment.Transition(domain.StatusStopped); err != nil {
		h.logger.Error("failed to transition to stopped", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to stop deployment", "internal_error")
		return
	}
	now := time.Now()
	deployment.StoppedAt = &now
	deployment.UpdatedAt = now

	if err := h.store.UpdateDeployment(r.Context(), deployment); err != nil {
		h.logger.Error("failed to update deployment", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to update deployment", "internal_error")
		return
	}

	h.logger.Info("deployment stopped", "deployment_id", deployment.ID, "node_id", deployment.NodeID)

	h.writeJSON(w, http.StatusOK, h.deploymentToResponse(deployment))
}

// =============================================================================
// Helpers
// =============================================================================

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to encode JSON", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message, code string) {
	h.writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func (h *Handler) templateToResponse(t *domain.Template) TemplateResponse {
	resp := TemplateResponse{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		Description:  t.Description,
		Version:      t.Version,
		ComposeSpec:  t.ComposeSpec,
		Variables:    make([]VariableResponse, 0, len(t.Variables)),
		ConfigFiles:  make([]ConfigFileResponse, 0, len(t.ConfigFiles)),
		PriceMonthly: int(t.PriceMonthly),
		Category:     t.Category,
		Tags:         t.Tags,
		Published:    t.Published,
		CreatorID:    t.CreatorID,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		ResourceRequirements: ResourcesResponse{
			CPUCores: int(t.ResourceRequirements.CPUCores),
			MemoryMB: int(t.ResourceRequirements.MemoryMB),
			DiskMB:   int(t.ResourceRequirements.DiskMB),
		},
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	for _, v := range t.Variables {
		resp.Variables = append(resp.Variables, VariableResponse{
			Name:        v.Name,
			Description: v.Description,
			Type:        string(v.Type),
			Default:     v.Default,
			Required:    v.Required,
		})
	}
	for _, cf := range t.ConfigFiles {
		resp.ConfigFiles = append(resp.ConfigFiles, ConfigFileResponse{
			Name:    cf.Name,
			Path:    cf.Path,
			Content: cf.Content,
			Mode:    cf.Mode,
		})
	}
	return resp
}

func (h *Handler) deploymentToResponse(d *domain.Deployment) DeploymentResponse {
	resp := DeploymentResponse{
		ID:              d.ID,
		Name:            d.Name,
		TemplateID:      d.TemplateID,
		TemplateVersion: d.TemplateVersion,
		CustomerID:      d.CustomerID,
		Status:          string(d.Status),
		Variables:       d.Variables,
		Domains:         make([]DomainResponse, 0, len(d.Domains)),
		Containers:      make([]ContainerResponse, 0, len(d.Containers)),
		Resources: ResourcesResponse{
			CPUCores: int(d.Resources.CPUCores),
			MemoryMB: int(d.Resources.MemoryMB),
			DiskMB:   int(d.Resources.DiskMB),
		},
		ErrorMessage: d.ErrorMessage,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
		StartedAt:    d.StartedAt,
		StoppedAt:    d.StoppedAt,
	}
	if resp.Variables == nil {
		resp.Variables = make(map[string]string)
	}
	for i, dom := range d.Domains {
		resp.Domains = append(resp.Domains, DomainResponse{
			Domain:   dom.Hostname,
			Primary:  i == 0, // First domain is primary
			Verified: dom.SSLEnabled,
		})
	}
	for _, c := range d.Containers {
		resp.Containers = append(resp.Containers, ContainerResponse{
			ServiceName: c.ServiceName,
			ContainerID: c.ID,
			Status:      c.Status,
		})
	}
	return resp
}

// isNotFound checks if an error is a not found error.
func isNotFound(err error) bool {
	var storeErr *store.StoreError
	if errors.As(err, &storeErr) {
		return errors.Is(storeErr.Unwrap(), store.ErrNotFound)
	}
	return false
}
