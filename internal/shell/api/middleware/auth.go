// Package middleware provides HTTP middleware for the Hoster API.
// Following ADR-005: APIGate Integration for Authentication and Billing
package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/artpar/hoster/internal/core/auth"
)

// =============================================================================
// User Resolver Interface
// =============================================================================

// UserResolver resolves an APIGate reference ID to a local integer user ID.
// The store implements this interface.
type UserResolver interface {
	ResolveUser(ctx context.Context, referenceID, email, name, planID string) (int, error)
}

// =============================================================================
// Auth Configuration
// =============================================================================

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	// SharedSecret is an optional secret to validate X-APIGate-Secret header.
	// If empty, secret validation is skipped.
	SharedSecret string

	// UserResolver resolves APIGate user reference IDs to local integer IDs.
	// If nil, UserID in auth context will be 0 (only ReferenceID will be set).
	UserResolver UserResolver

	// Logger for auth middleware logging.
	Logger *slog.Logger
}

// =============================================================================
// Auth Middleware
// =============================================================================

// AuthMiddleware extracts authentication context from APIGate headers
// and stores it in the request context.
type AuthMiddleware struct {
	config AuthConfig
}

// NewAuthMiddleware creates a new auth middleware with the given config.
func NewAuthMiddleware(cfg AuthConfig) *AuthMiddleware {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &AuthMiddleware{config: cfg}
}

// Handler returns the middleware handler function.
// This middleware extracts auth context from APIGate-injected headers and stores it in the request context.
// If a UserResolver is configured, it also resolves the APIGate reference ID to a local integer user ID.
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate shared secret if configured
		if m.config.SharedSecret != "" {
			if r.Header.Get(auth.HeaderAPIGateSecret) != m.config.SharedSecret {
				m.config.Logger.Warn("invalid APIGate secret",
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
				)
				writeJSONError(w, http.StatusForbidden, "Forbidden", "Invalid gateway secret")
				return
			}
		}

		// Extract auth context from headers (sets ReferenceID, not UserID)
		ctx := auth.ExtractFromRequest(r)

		// Resolve integer user ID if authenticated and resolver is available
		if ctx.Authenticated && m.config.UserResolver != nil {
			userID, err := m.config.UserResolver.ResolveUser(
				r.Context(),
				ctx.ReferenceID,
				"", // email — not in headers
				"", // name — not in headers
				ctx.PlanID,
			)
			if err != nil {
				m.config.Logger.Error("failed to resolve user",
					"reference_id", ctx.ReferenceID,
					"error", err,
				)
				writeJSONError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to resolve user identity")
				return
			}
			ctx.UserID = userID
		}

		// Store in request context
		r = r.WithContext(auth.WithContext(r.Context(), ctx))

		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// Require Auth Middleware
// =============================================================================

// RequireAuth is a middleware that requires authentication.
// Use this for protected endpoints that must have a valid user.
// Must be used AFTER AuthMiddleware.
func RequireAuth(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.FromContext(r.Context())

			if !ctx.Authenticated {
				logger.Warn("unauthenticated request to protected endpoint",
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
					"method", r.Method,
				)
				writeJSONError(w, http.StatusUnauthorized, "Unauthorized", "Authentication required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// Optional Auth Middleware
// =============================================================================

// OptionalAuth is a middleware that allows both authenticated and unauthenticated requests.
// The auth context will be available if present, but requests won't be rejected.
// This is the same as just using AuthMiddleware.Handler() - kept for clarity.
func OptionalAuth(next http.Handler) http.Handler {
	return next
}

// =============================================================================
// JSON Error Response
// =============================================================================

// JSONAPIError represents a JSON:API error object.
type JSONAPIError struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// JSONAPIErrorResponse represents a JSON:API error response.
type JSONAPIErrorResponse struct {
	Errors []JSONAPIError `json:"errors"`
}

// writeJSONError writes a JSON:API formatted error response.
func writeJSONError(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(JSONAPIErrorResponse{
		Errors: []JSONAPIError{
			{
				Status: http.StatusText(status),
				Title:  title,
				Detail: detail,
			},
		},
	})
}
