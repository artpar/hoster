package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
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
			planID := r.Header.Get(HeaderPlanID)

			// Fallback: extract from JWT Bearer token when APIGate
			// doesn't inject identity headers (no request_transform configured).
			if referenceID == "" {
				if claims := parseJWTClaims(r); claims != nil {
					referenceID = claims.UserID
					if planID == "" {
						planID = claims.PlanID
					}
				}
			}

			if referenceID == "" {
				// Unauthenticated — continue with empty AuthContext
				next.ServeHTTP(w, r)
				return
			}

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

			// Parse plan limits from header or derive from plan ID
			if limitsJSON := r.Header.Get(HeaderPlanLimits); limitsJSON != "" {
				var limits PlanLimits
				if err := json.Unmarshal([]byte(limitsJSON), &limits); err == nil {
					ac.PlanLimits = limits
				}
			} else if ac.PlanID != "" {
				ac.PlanLimits = DefaultPlanLimits(ac.PlanID)
			}

			r = r.WithContext(WithAuth(r.Context(), ac))
			next.ServeHTTP(w, r)
		})
	}
}

// jwtClaims represents the relevant fields from an APIGate JWT payload.
type jwtClaims struct {
	UserID string `json:"uid"`
	PlanID string `json:"pid"`
	Exp    int64  `json:"exp"`
}

// parseJWTClaims extracts user identity from the Authorization Bearer token.
// Signature verification is skipped because APIGate already validated the JWT
// and Hoster is only reachable via APIGate on localhost.
func parseJWTClaims(r *http.Request) *jwtClaims {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil
	}
	if claims.UserID == "" {
		return nil
	}
	return &claims
}

// PlanLimits defines resource limits for a user's subscription plan.
type PlanLimits struct {
	MaxDeployments      int      `json:"max_deployments"`
	MaxCPUCores         float64  `json:"max_cpu_cores"`
	MaxMemoryMB         int64    `json:"max_memory_mb"`
	MaxDiskMB           int64    `json:"max_disk_mb"`
	AllowedCapabilities []string `json:"allowed_capabilities"`
}

// DefaultPlanLimits returns the default limits for a plan ID when
// the X-Plan-Limits header is not injected by APIGate.
func DefaultPlanLimits(planID string) PlanLimits {
	switch planID {
	case "free":
		return PlanLimits{
			MaxDeployments: 1,
			MaxCPUCores:    1,
			MaxMemoryMB:    1024,
			MaxDiskMB:      5120,
		}
	case "starter":
		return PlanLimits{
			MaxDeployments: 5,
			MaxCPUCores:    4,
			MaxMemoryMB:    4096,
			MaxDiskMB:      20480,
		}
	case "pro":
		return PlanLimits{
			MaxDeployments: 20,
			MaxCPUCores:    16,
			MaxMemoryMB:    16384,
			MaxDiskMB:      102400,
		}
	default:
		return PlanLimits{}
	}
}
