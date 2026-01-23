// Package api provides HTTP handlers for the Hoster API.
// Following ADR-003: JSON:API Standard with api2go
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/artpar/hoster/internal/shell/api/middleware"
	"github.com/artpar/hoster/internal/shell/api/openapi"
	"github.com/artpar/hoster/internal/shell/api/resources"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/scheduler"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go"
)

// =============================================================================
// API Setup - Following ADR-003: JSON:API with api2go
// =============================================================================

// APIConfig holds configuration for the API setup.
type APIConfig struct {
	Store      store.Store
	Docker     docker.Client
	Scheduler  *scheduler.Service // Scheduler for node selection (nil = local only)
	Logger     *slog.Logger
	BaseDomain string
	ConfigDir  string

	// Auth configuration (following ADR-005)
	AuthMode         string // "header", "dev", or "none"
	AuthRequire      bool   // Require auth for protected endpoints
	AuthSharedSecret string // Optional: validate X-APIGate-Secret

	// Encryption key for SSH keys (required for node management)
	EncryptionKey []byte

	// Frontend configuration - serve static files from this directory
	FrontendDir string // e.g., "/opt/hoster/web" - if empty, no static files served
}

// SetupAPI creates the complete API router with JSON:API resources and custom endpoints.
// Returns an http.Handler that can be used as the server's main handler.
func SetupAPI(cfg APIConfig) http.Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "/var/lib/hoster/configs"
	}

	// Create the main router
	router := mux.NewRouter()

	// Add middleware
	router.Use(requestIDMiddleware)
	router.Use(recoveryMiddleware(cfg.Logger))

	// Health endpoints (not JSON:API, just simple JSON)
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/ready", readyHandler(cfg.Docker)).Methods("GET")

	// Create api2go API for JSON:API resources
	// Using NewAPIWithResolver - it creates its own internal router
	jsonAPI := api2go.NewAPIWithResolver("v1", api2go.NewStaticResolver("/api"))

	// Set content type for JSON:API
	jsonAPI.ContentType = "application/vnd.api+json"

	// Register JSON:API resources
	templateResource := resources.NewTemplateResource(cfg.Store)
	deploymentResource := resources.NewDeploymentResource(
		cfg.Store,
		cfg.Docker,
		cfg.Scheduler,
		cfg.Logger,
		cfg.BaseDomain,
		cfg.ConfigDir,
	)
	nodeResource := resources.NewNodeResource(cfg.Store)
	sshKeyResource := resources.NewSSHKeyResource(cfg.Store, cfg.EncryptionKey)

	jsonAPI.AddResource(resources.Template{}, templateResource)
	jsonAPI.AddResource(resources.Deployment{}, deploymentResource)
	jsonAPI.AddResource(resources.Node{}, nodeResource)
	jsonAPI.AddResource(resources.SSHKey{}, sshKeyResource)

	// Mount api2go handler under /api
	router.PathPrefix("/api").Handler(jsonAPI.Handler())

	// Add custom action endpoints (not standard JSON:API CRUD)
	// These are actions that don't map to standard CRUD operations
	// Note: These routes must be registered BEFORE the PathPrefix handler above
	// to avoid being caught by the api2go handler

	// Re-create router with custom actions first
	customRouter := mux.NewRouter()
	customRouter.Use(requestIDMiddleware)
	customRouter.Use(recoveryMiddleware(cfg.Logger))

	// Create dev auth handlers if in dev mode (needed for session lookup)
	var devAuth *DevAuthHandlers
	if cfg.AuthMode == "dev" {
		cfg.Logger.Info("registering dev auth endpoints (auth.mode=dev)")
		devAuth = NewDevAuthHandlers(cfg.Logger)
		devAuth.RegisterRoutes(customRouter)
	}

	// Add auth middleware (following ADR-005: APIGate Integration)
	authConfig := middleware.AuthConfig{
		Mode:         cfg.AuthMode,
		RequireAuth:  cfg.AuthRequire,
		SharedSecret: cfg.AuthSharedSecret,
		Logger:       cfg.Logger,
	}

	// In dev mode, pass the session lookup function so the middleware
	// can get the actual user ID from the dev session
	if devAuth != nil {
		authConfig.DevSessionLookup = func(sessionID string) *middleware.DevSession {
			session := devAuth.LookupSession(sessionID)
			if session == nil {
				return nil
			}
			return &middleware.DevSession{
				UserID: session.UserID,
				Email:  session.Email,
				Name:   session.Name,
			}
		}
	}

	authMW := middleware.NewAuthMiddleware(authConfig)
	customRouter.Use(authMW.Handler)

	// Health endpoints
	customRouter.HandleFunc("/health", healthHandler).Methods("GET")
	customRouter.HandleFunc("/ready", readyHandler(cfg.Docker)).Methods("GET")

	// Template custom actions
	customRouter.HandleFunc("/api/v1/templates/{id}/publish", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		resp, err := templateResource.PublishTemplate(id, r)
		writeResponder(w, resp, err, cfg.Logger)
	}).Methods("POST")

	// Deployment custom actions
	customRouter.HandleFunc("/api/v1/deployments/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		resp, err := deploymentResource.StartDeployment(id, r)
		writeResponder(w, resp, err, cfg.Logger)
	}).Methods("POST")

	customRouter.HandleFunc("/api/v1/deployments/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		resp, err := deploymentResource.StopDeployment(id, r)
		writeResponder(w, resp, err, cfg.Logger)
	}).Methods("POST")

	// Node custom actions
	customRouter.HandleFunc("/api/v1/nodes/{id}/maintenance", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		// Toggle maintenance mode on
		resp, err := nodeResource.SetMaintenance(id, true, r)
		writeResponder(w, resp, err, cfg.Logger)
	}).Methods("POST")

	customRouter.HandleFunc("/api/v1/nodes/{id}/maintenance", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		// Toggle maintenance mode off
		resp, err := nodeResource.SetMaintenance(id, false, r)
		writeResponder(w, resp, err, cfg.Logger)
	}).Methods("DELETE")

	// Monitoring endpoints - Following F010: Monitoring Dashboard
	monitoringHandlers := NewMonitoringHandlers(cfg.Store, cfg.Docker)
	monitoringHandlers.RegisterRoutes(customRouter)

	// OpenAPI endpoint - Following ADR-004: Reflective OpenAPI Generation
	openapiGen := openapi.NewGenerator(
		openapi.WithTitle("Hoster API"),
		openapi.WithVersion("1.0.0"),
		openapi.WithDescription("Deployment marketplace platform API following JSON:API specification"),
		openapi.WithServer("/api/v1"),
	)

	// Register resources for OpenAPI documentation
	openapiGen.RegisterResource(openapi.ResourceInfo{
		Name:           "templates",
		Model:          resources.Template{},
		SupportsFind:   true,
		SupportsCreate: true,
		SupportsUpdate: true,
		SupportsDelete: true,
	})
	openapiGen.RegisterResource(openapi.ResourceInfo{
		Name:           "deployments",
		Model:          resources.Deployment{},
		SupportsFind:   true,
		SupportsCreate: true,
		SupportsUpdate: false, // Deployments are managed via actions, not direct updates
		SupportsDelete: true,
	})
	openapiGen.RegisterResource(openapi.ResourceInfo{
		Name:           "nodes",
		Model:          resources.Node{},
		SupportsFind:   true,
		SupportsCreate: true,
		SupportsUpdate: true,
		SupportsDelete: true,
	})
	openapiGen.RegisterResource(openapi.ResourceInfo{
		Name:           "ssh_keys",
		Model:          resources.SSHKey{},
		SupportsFind:   true,
		SupportsCreate: true,
		SupportsUpdate: false, // SSH keys are immutable
		SupportsDelete: true,
	})

	customRouter.HandleFunc("/openapi.json", openapiGen.Handler()).Methods("GET")

	// Mount api2go handler for all other /api routes
	// api2go expects paths without the /api prefix (e.g., /v1/templates not /api/v1/templates)
	// so we strip the /api prefix before passing to the api2go handler
	customRouter.PathPrefix("/api").Handler(http.StripPrefix("/api", jsonAPI.Handler()))

	// Serve embedded Web UI for all other paths (SPA pattern)
	// This must be registered last to act as a catch-all
	customRouter.PathPrefix("/").Handler(WebUIHandler())

	return customRouter
}

// =============================================================================
// Middleware
// =============================================================================

// requestIDMiddleware generates and adds a request ID to responses.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics and returns a 500 error.
func recoveryMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered", "error", err)
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"errors": []map[string]interface{}{
							{
								"status": "500",
								"title":  "Internal Server Error",
								"detail": "An unexpected error occurred",
							},
						},
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// Health Handlers
// =============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func readyHandler(docker docker.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		checks := make(map[string]string)
		checks["database"] = "ok"

		if err := docker.Ping(); err != nil {
			checks["docker"] = "failed"
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "not_ready",
				"checks": checks,
			})
			return
		}
		checks["docker"] = "ok"

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ready",
			"checks": checks,
		})
	}
}

// =============================================================================
// Helpers
// =============================================================================

// writeResponder writes an api2go.Responder to the response writer.
func writeResponder(w http.ResponseWriter, resp api2go.Responder, err error, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/vnd.api+json")

	if err != nil {
		// Check if it's an HTTPError from api2go
		if httpErr, ok := err.(api2go.HTTPError); ok {
			// Errors is a slice, not a method
			if len(httpErr.Errors) > 0 {
				status := parseStatus(httpErr.Errors[0].Status)
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": httpErr.Errors,
				})
				return
			}
		}
		logger.Error("request error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]interface{}{
				{
					"status": "500",
					"title":  "Internal Server Error",
					"detail": err.Error(),
				},
			},
		})
		return
	}

	if resp == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteHeader(resp.StatusCode())
	if result := resp.Result(); result != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": result,
			"meta": resp.Metadata(),
		})
	}
}

// parseStatus converts a status string to an int.
func parseStatus(status string) int {
	// Try to parse status as a number
	if status == "" {
		return http.StatusInternalServerError
	}
	// Use json.Number to parse
	n := json.Number(status)
	if i, err := n.Int64(); err == nil && i > 0 {
		return int(i)
	}
	return http.StatusInternalServerError
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	// Simple implementation - could use UUID in production
	return "req_" + randomString(12)
}

// randomString generates a random string of the given length.
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
