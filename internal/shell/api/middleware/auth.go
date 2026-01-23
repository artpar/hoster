// Package middleware provides HTTP middleware for the Hoster API.
// Following ADR-005: APIGate Integration for Authentication and Billing
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/artpar/hoster/internal/core/auth"
)

// =============================================================================
// Auth Configuration
// =============================================================================

// DevSession represents a dev session for session lookup.
type DevSession struct {
	UserID string
	Email  string
	Name   string
}

// DevSessionLookup is a function that looks up a dev session by session ID.
// Returns nil if the session is not found.
type DevSessionLookup func(sessionID string) *DevSession

// AuthConfig holds configuration for the auth middleware.
type AuthConfig struct {
	// Mode determines how authentication is handled.
	// "header" - Extract auth from APIGate headers (production)
	// "none" - Skip auth extraction entirely (unauthenticated requests)
	// "dev" - Auto-authenticate as dev-user (local development)
	Mode string

	// RequireAuth determines if authentication is required for protected endpoints.
	// When true, unauthenticated requests to protected endpoints return 401.
	RequireAuth bool

	// SharedSecret is an optional secret to validate X-APIGate-Secret header.
	// If empty, secret validation is skipped.
	SharedSecret string

	// Logger for auth middleware logging.
	Logger *slog.Logger

	// DevSessionLookup is used in "dev" mode to look up sessions from the dev auth handler.
	// If nil in dev mode, falls back to hardcoded "dev-user".
	DevSessionLookup DevSessionLookup
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
// This middleware extracts auth context from headers and stores it in the request context.
// If Mode is "none", it skips authentication extraction entirely (unauthenticated).
// If Mode is "dev", it auto-authenticates as a dev user for local development.
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In "none" mode, skip auth extraction entirely (unauthenticated)
		if m.config.Mode == "none" {
			// Set an unauthenticated context
			emptyCtx := auth.Context{
				Authenticated: false,
				PlanLimits:    auth.DefaultPlanLimits(),
			}
			r = r.WithContext(auth.WithContext(r.Context(), emptyCtx))
			next.ServeHTTP(w, r)
			return
		}

		// In "dev" mode, check for dev session cookie first
		if m.config.Mode == "dev" {
			userID := "dev-user" // fallback

			// Try to get user ID from dev session cookie
			if m.config.DevSessionLookup != nil {
				if cookie, err := r.Cookie("hoster_dev_session"); err == nil {
					if session := m.config.DevSessionLookup(cookie.Value); session != nil {
						userID = session.UserID
						m.config.Logger.Debug("dev auth: using session user",
							"user_id", userID,
							"email", session.Email,
						)
					}
				}
			}

			devCtx := auth.Context{
				UserID:        userID,
				PlanID:        "dev-plan",
				PlanLimits:    auth.DefaultPlanLimits(),
				Authenticated: true,
			}
			r = r.WithContext(auth.WithContext(r.Context(), devCtx))
			next.ServeHTTP(w, r)
			return
		}

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

		// Extract auth context from headers
		ctx := auth.ExtractFromRequest(r)

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
