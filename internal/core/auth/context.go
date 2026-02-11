// Package auth provides authentication context and authorization functions.
// Following ADR-005: APIGate Integration for Authentication and Billing
package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// =============================================================================
// Context Key
// =============================================================================

type contextKey string

const authContextKey contextKey = "auth"

// =============================================================================
// Types
// =============================================================================

// Context represents the authentication and authorization context for a request.
// It is extracted from APIGate-injected headers and stored in the request context.
type Context struct {
	// UserID is the local integer PK from the users table (resolved by middleware).
	UserID int

	// ReferenceID is the APIGate user ID string (from X-User-ID header, e.g. "user_bc6849d9ab6dc0e5").
	ReferenceID string

	// PlanID is the user's subscription plan ID (from X-Plan-ID header)
	PlanID string

	// PlanLimits contains resource limits from the user's plan (from X-Plan-Limits header)
	PlanLimits PlanLimits

	// KeyID is the API key ID if API key authentication was used (from X-Key-ID header)
	KeyID string

	// OrganizationID is the organization ID for future multi-tenant support (from X-Organization-ID header)
	OrganizationID string

	// Authenticated indicates whether the request is authenticated
	Authenticated bool
}

// PlanLimits defines the resource limits for a user's subscription plan.
type PlanLimits struct {
	// MaxDeployments is the maximum number of active deployments allowed
	MaxDeployments int `json:"max_deployments"`

	// MaxCPUCores is the maximum total CPU cores across all deployments
	MaxCPUCores float64 `json:"max_cpu_cores"`

	// MaxMemoryMB is the maximum total memory in MB across all deployments
	MaxMemoryMB int64 `json:"max_memory_mb"`

	// MaxDiskMB is the maximum total disk space in MB across all deployments
	MaxDiskMB int64 `json:"max_disk_mb"`

	// AllowedCapabilities lists node capability tags the plan permits
	// e.g., ["standard"] for basic plans, ["standard","gpu","high-memory"] for premium
	AllowedCapabilities []string `json:"allowed_capabilities"`
}

// =============================================================================
// Header Constants
// =============================================================================

const (
	// HeaderUserID is the header containing the authenticated user's ID
	HeaderUserID = "X-User-ID"

	// HeaderPlanID is the header containing the user's plan ID
	HeaderPlanID = "X-Plan-ID"

	// HeaderPlanLimits is the header containing JSON-encoded plan limits
	HeaderPlanLimits = "X-Plan-Limits"

	// HeaderKeyID is the header containing the API key ID
	HeaderKeyID = "X-Key-ID"

	// HeaderOrganizationID is the header containing the organization ID
	HeaderOrganizationID = "X-Organization-ID"

	// HeaderAPIGateSecret is the header containing the shared secret for validation
	HeaderAPIGateSecret = "X-APIGate-Secret"
)

// =============================================================================
// Context Extraction
// =============================================================================

// ExtractFromRequest extracts auth context from HTTP request headers.
// If X-User-ID header is not present, returns an unauthenticated context.
func ExtractFromRequest(r *http.Request) Context {
	return ExtractFromHeaders(headerGetter{r: r})
}

// HeaderGetter is an interface for getting header values.
// This allows testing without requiring an http.Request.
type HeaderGetter interface {
	Get(key string) string
}

type headerGetter struct {
	r *http.Request
}

func (h headerGetter) Get(key string) string {
	return h.r.Header.Get(key)
}

// ExtractFromHeaders extracts auth context from headers using the HeaderGetter interface.
// This is a pure function that can be tested without HTTP dependencies.
// Note: UserID (int) is NOT set here — it must be resolved by the middleware
// via a user upsert against the database. Only ReferenceID is extracted.
//
// Auth sources (checked in order):
//  1. X-User-ID header (injected by APIGate for some routes)
//  2. Authorization: Bearer {jwt} — decode payload, extract sub claim
func ExtractFromHeaders(headers HeaderGetter) Context {
	referenceID := headers.Get(HeaderUserID)

	// Fallback: parse Bearer token payload when X-User-ID is absent.
	// Per architecture spec, Hoster validates Bearer tokens directly.
	// No signature verification — APIGate has already validated the token.
	if referenceID == "" {
		claims := parseBearer(headers.Get("Authorization"))
		if claims == nil || claims.Sub == "" {
			return Context{Authenticated: false}
		}

		limits, err := ParsePlanLimits(headers.Get(HeaderPlanLimits))
		if err != nil {
			limits = DefaultPlanLimits()
		}

		planID := claims.PlanID
		if planID == "" {
			planID = headers.Get(HeaderPlanID)
		}

		return Context{
			ReferenceID:   claims.Sub,
			PlanID:        planID,
			PlanLimits:    limits,
			Authenticated: true,
		}
	}

	limits, err := ParsePlanLimits(headers.Get(HeaderPlanLimits))
	if err != nil {
		// If parsing fails, use default limits
		limits = DefaultPlanLimits()
	}

	return Context{
		ReferenceID:    referenceID,
		PlanID:         headers.Get(HeaderPlanID),
		PlanLimits:     limits,
		KeyID:          headers.Get(HeaderKeyID),
		OrganizationID: headers.Get(HeaderOrganizationID),
		Authenticated:  true,
	}
}

// jwtClaims holds the fields extracted from a JWT payload.
type jwtClaims struct {
	Sub    string `json:"sub"`
	PlanID string `json:"pid"`
}

// parseBearer extracts claims from a Bearer token by base64-decoding the payload.
// No signature verification — Hoster trusts APIGate to have validated the token.
func parseBearer(authHeader string) *jwtClaims {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	parts := strings.Split(authHeader[7:], ".")
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
	return &claims
}

// =============================================================================
// Plan Limits Parsing
// =============================================================================

// ParsePlanLimits parses a JSON string into PlanLimits.
// If the string is empty, returns DefaultPlanLimits.
// Returns an error if the JSON is invalid.
func ParsePlanLimits(jsonStr string) (PlanLimits, error) {
	if jsonStr == "" {
		return DefaultPlanLimits(), nil
	}

	var limits PlanLimits
	if err := json.Unmarshal([]byte(jsonStr), &limits); err != nil {
		return PlanLimits{}, fmt.Errorf("invalid plan limits: %w", err)
	}

	return limits, nil
}

// DefaultPlanLimits returns the default plan limits for users without a specified plan.
// These are conservative limits for free/starter tier.
func DefaultPlanLimits() PlanLimits {
	return PlanLimits{
		MaxDeployments: 1,
		MaxCPUCores:    1.0,
		MaxMemoryMB:    1024,
		MaxDiskMB:      5120,
	}
}

// =============================================================================
// Context Storage
// =============================================================================

// WithContext stores the auth context in the request context.
func WithContext(ctx context.Context, authCtx Context) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

// FromContext retrieves the auth context from the request context.
// If no auth context is found, returns an unauthenticated context.
func FromContext(ctx context.Context) Context {
	if authCtx, ok := ctx.Value(authContextKey).(Context); ok {
		return authCtx
	}
	return Context{Authenticated: false}
}

// =============================================================================
// Helper Types for Testing
// =============================================================================

// MapHeaderGetter wraps a map to implement HeaderGetter interface.
// This is useful for testing without creating http.Request objects.
type MapHeaderGetter map[string]string

func (m MapHeaderGetter) Get(key string) string {
	return m[key]
}
