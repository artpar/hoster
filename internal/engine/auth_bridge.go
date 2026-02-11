package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

// Auth header constants (injected by APIGate).
const (
	HeaderUserID         = "X-User-ID"
	HeaderPlanID         = "X-Plan-ID"
	HeaderPlanLimits     = "X-Plan-Limits"
	HeaderKeyID          = "X-Key-ID"
	HeaderOrganizationID = "X-Organization-ID"
	HeaderAPIGateSecret  = "X-APIGate-Secret"
)

type authContextKey struct{}

// AuthFromRequest extracts AuthContext from an HTTP request's context.
func AuthFromRequest(r *http.Request) AuthContext {
	if ac, ok := r.Context().Value(authContextKey{}).(AuthContext); ok {
		return ac
	}
	return AuthContext{}
}

// WithAuth stores an AuthContext in a context.
func WithAuth(ctx context.Context, ac AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, ac)
}

// AuthMiddleware extracts auth from APIGate-injected headers,
// resolves the user via the engine Store, and injects AuthContext.
func AuthMiddleware(store *Store, sharedSecret string, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate shared secret if configured
			if sharedSecret != "" {
				if r.Header.Get(HeaderAPIGateSecret) != sharedSecret {
					writeError(w, http.StatusForbidden, "invalid gateway secret")
					return
				}
			}

			referenceID := r.Header.Get(HeaderUserID)
			if referenceID == "" {
				// Unauthenticated — continue with empty AuthContext
				next.ServeHTTP(w, r)
				return
			}

			planID := r.Header.Get(HeaderPlanID)

			// Resolve integer user ID
			userID, err := store.ResolveUser(r.Context(), referenceID, "", "", planID)
			if err != nil {
				logger.Error("failed to resolve user", "reference_id", referenceID, "error", err)
				writeError(w, http.StatusInternalServerError, "failed to resolve user identity")
				return
			}

			ac := AuthContext{
				Authenticated: true,
				UserID:        userID,
				ReferenceID:   referenceID,
				PlanID:        planID,
			}

			// Parse plan limits if present
			if limitsJSON := r.Header.Get(HeaderPlanLimits); limitsJSON != "" {
				var limits PlanLimits
				if err := json.Unmarshal([]byte(limitsJSON), &limits); err == nil {
					ac.PlanLimits = limits
				}
			}

			r = r.WithContext(WithAuth(r.Context(), ac))
			next.ServeHTTP(w, r)
		})
	}
}

// PlanLimits defines resource limits for a user's subscription plan.
type PlanLimits struct {
	MaxDeployments      int      `json:"max_deployments"`
	MaxCPUCores         float64  `json:"max_cpu_cores"`
	MaxMemoryMB         int64    `json:"max_memory_mb"`
	MaxDiskMB           int64    `json:"max_disk_mb"`
	AllowedCapabilities []string `json:"allowed_capabilities"`
}
