package engine

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

//go:embed all:webui/dist
var webUI embed.FS

// SetupConfig holds configuration for the engine HTTP handler.
type SetupConfig struct {
	Store         *Store
	Bus           *Bus
	Logger        *slog.Logger
	BaseDomain    string
	ConfigDir     string
	SharedSecret  string
	EncryptionKey []byte
}

// Setup creates the complete HTTP handler using the engine.
func Setup(cfg SetupConfig) http.Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	router := mux.NewRouter()

	// Middleware
	router.Use(requestIDMiddleware)
	router.Use(recoveryMiddleware(cfg.Logger))
	router.Use(AuthMiddleware(cfg.Store, cfg.SharedSecret, cfg.Logger))

	// Health endpoints
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/ready", readyHandler).Methods("GET")

	// Wire deployment BeforeCreate: resolve template_version from template
	if deplRes := cfg.Store.Resource("deployments"); deplRes != nil {
		store := cfg.Store
		deplRes.BeforeCreate = func(ctx context.Context, authCtx AuthContext, data map[string]any) error {
			// If template_version not set, copy from template
			if _, ok := data["template_version"]; !ok || data["template_version"] == nil || data["template_version"] == "" {
				if tid, ok := toInt64(data["template_id"]); ok && tid > 0 {
					tmpl, err := store.GetByID(ctx, "templates", int(tid))
					if err == nil {
						data["template_version"] = strVal(tmpl["version"])
					}
				}
			}
			return nil
		}
	}

	// Register generic CRUD + state machine routes for all resources
	RegisterRoutes(router, APIConfig{
		Store:          cfg.Store,
		Bus:            cfg.Bus,
		Logger:         cfg.Logger,
		ActionHandlers: buildActionHandlers(cfg),
	})

	// Serve embedded Web UI for all other paths (SPA pattern)
	router.PathPrefix("/").Handler(spaHandler())

	return router
}

// buildActionHandlers creates custom action handlers beyond standard CRUD.
func buildActionHandlers(cfg SetupConfig) map[string]http.HandlerFunc {
	handlers := map[string]http.HandlerFunc{}

	// Template: publish
	handlers["templates:publish"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		tmpl, err := cfg.Store.Get(ctx, "templates", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}

		// Check ownership — fail closed
		ownerID, ok := toInt64(tmpl["creator_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable creator_id",
				"resource", "templates", "value", tmpl["creator_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		row, err := cfg.Store.Update(ctx, "templates", id, map[string]any{"published": 1})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		res := cfg.Store.Resource("templates")
		stripFields(res, row, cfg.Store)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("templates", row),
		})
	}

	// Deployment: start (transition pending → scheduled, triggers schedule command)
	handlers["deployments:start"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		existing, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		// Check ownership — fail closed
		ownerID, ok := toInt64(existing["customer_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable customer_id",
				"resource", "deployments", "value", existing["customer_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		status, _ := existing["status"].(string)

		// Determine target state based on current status
		var targetState string
		switch status {
		case "pending":
			targetState = "scheduled"
		case "stopped", "failed":
			targetState = "starting"
		default:
			writeError(w, http.StatusConflict, "cannot start deployment in state: "+status)
			return
		}

		row, cmd, err := cfg.Store.Transition(ctx, "deployments", id, targetState)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		// Dispatch command
		if cmd != "" && cfg.Bus != nil {
			if err := cfg.Bus.Dispatch(ctx, cmd, row); err != nil {
				cfg.Logger.Error("command dispatch failed", "command", cmd, "error", err)
			}
		}

		res := cfg.Store.Resource("deployments")
		stripFields(res, row, cfg.Store)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("deployments", row),
		})
	}

	// Deployment: stop (transition running → stopping, triggers stop command)
	handlers["deployments:stop"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		existing, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		ownerID, ok := toInt64(existing["customer_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable customer_id",
				"resource", "deployments", "value", existing["customer_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		row, cmd, err := cfg.Store.Transition(ctx, "deployments", id, "stopping")
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		if cmd != "" && cfg.Bus != nil {
			if err := cfg.Bus.Dispatch(ctx, cmd, row); err != nil {
				cfg.Logger.Error("command dispatch failed", "command", cmd, "error", err)
			}
		}

		res := cfg.Store.Resource("deployments")
		stripFields(res, row, cfg.Store)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("deployments", row),
		})
	}

	return handlers
}

// =============================================================================
// Middleware
// =============================================================================

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req_" + randomString(12)
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered", "error", err)
					writeError(w, http.StatusInternalServerError, "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// Health
// =============================================================================

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func readyHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ready",
		"checks": map[string]string{"database": "ok"},
	})
}

// =============================================================================
// SPA Handler
// =============================================================================

func spaHandler() http.Handler {
	distFS, err := fs.Sub(webUI, "webui/dist")
	if err != nil {
		// Fallback if dist not embedded
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html><html><head><title>Hoster</title></head><body><p>Frontend not built. Run: cd web && npm run build</p></body></html>`))
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For API paths that weren't matched, return 404
		if len(r.URL.Path) > 4 && r.URL.Path[:5] == "/api/" {
			writeError(w, http.StatusNotFound, "not found")
			return
		}

		// Try to serve the file directly
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(distFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for all unmatched paths
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[idx.Int64()]
	}
	return string(b)
}
