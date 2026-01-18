package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artpar/hoster/internal/core/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// testHandler is a simple handler that returns the auth context from request.
func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": ctx.Authenticated,
			"user_id":       ctx.UserID,
			"plan_id":       ctx.PlanID,
		})
	})
}

// =============================================================================
// AuthMiddleware Tests
// =============================================================================

func TestAuthMiddleware_ModeNone_SkipsAuth(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode: "none",
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Context should be unauthenticated since mode=none doesn't extract
	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["authenticated"])
}

func TestAuthMiddleware_HeaderMode_ExtractsContext(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode: "header",
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-User-ID", "user_123")
	req.Header.Set("X-Plan-ID", "plan_premium")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["authenticated"])
	assert.Equal(t, "user_123", resp["user_id"])
	assert.Equal(t, "plan_premium", resp["plan_id"])
}

func TestAuthMiddleware_HeaderMode_NoHeaders(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode: "header",
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["authenticated"])
}

func TestAuthMiddleware_SharedSecret_Valid(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode:         "header",
		SharedSecret: "my-secret-key",
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-APIGate-Secret", "my-secret-key")
	req.Header.Set("X-User-ID", "user_123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_SharedSecret_Invalid(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode:         "header",
		SharedSecret: "my-secret-key",
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-APIGate-Secret", "wrong-secret")
	req.Header.Set("X-User-ID", "user_123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/vnd.api+json")
}

func TestAuthMiddleware_SharedSecret_Missing(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode:         "header",
		SharedSecret: "my-secret-key",
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-User-ID", "user_123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuthMiddleware_EmptyMode_DefaultsToHeader(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode: "", // empty mode
	})

	handler := middleware.Handler(testHandler())
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-User-ID", "user_456")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, true, resp["authenticated"])
}

// =============================================================================
// RequireAuth Middleware Tests
// =============================================================================

func TestRequireAuth_Authenticated(t *testing.T) {
	// Setup auth middleware first
	authMW := NewAuthMiddleware(AuthConfig{Mode: "header"})
	requireMW := RequireAuth(nil)

	handler := authMW.Handler(requireMW(testHandler()))
	req := httptest.NewRequest("GET", "/api/v1/protected", nil)
	req.Header.Set("X-User-ID", "user_123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAuth_Unauthenticated(t *testing.T) {
	// Setup auth middleware first
	authMW := NewAuthMiddleware(AuthConfig{Mode: "header"})
	requireMW := RequireAuth(nil)

	handler := authMW.Handler(requireMW(testHandler()))
	req := httptest.NewRequest("GET", "/api/v1/protected", nil)
	// No X-User-ID header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/vnd.api+json")

	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "Authentication required")
}

func TestRequireAuth_ModeNone_StillRequires(t *testing.T) {
	// Even with mode=none, RequireAuth should check the context
	authMW := NewAuthMiddleware(AuthConfig{Mode: "none"})
	requireMW := RequireAuth(nil)

	handler := authMW.Handler(requireMW(testHandler()))
	req := httptest.NewRequest("GET", "/api/v1/protected", nil)
	req.Header.Set("X-User-ID", "user_123") // This won't be extracted in none mode
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should fail because mode=none doesn't extract context
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// =============================================================================
// JSON Error Response Tests
// =============================================================================

func TestWriteJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSONError(rec, http.StatusNotFound, "Not Found", "Resource not found")

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "application/vnd.api+json", rec.Header().Get("Content-Type"))

	var resp JSONAPIErrorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, "Not Found", resp.Errors[0].Title)
	assert.Equal(t, "Resource not found", resp.Errors[0].Detail)
}

// =============================================================================
// Plan Limits Extraction Tests
// =============================================================================

func TestAuthMiddleware_ExtractsPlanLimits(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode: "header",
	})

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"max_deployments": ctx.PlanLimits.MaxDeployments,
			"max_cpu_cores":   ctx.PlanLimits.MaxCPUCores,
		})
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-User-ID", "user_123")
	req.Header.Set("X-Plan-Limits", `{"max_deployments": 10, "max_cpu_cores": 8.0}`)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(10), resp["max_deployments"])
	assert.Equal(t, float64(8.0), resp["max_cpu_cores"])
}

func TestAuthMiddleware_DefaultPlanLimits(t *testing.T) {
	middleware := NewAuthMiddleware(AuthConfig{
		Mode: "header",
	})

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.FromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"max_deployments": ctx.PlanLimits.MaxDeployments,
		})
	}))

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-User-ID", "user_123")
	// No X-Plan-Limits header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	// Should have default limit
	assert.Equal(t, float64(1), resp["max_deployments"])
}
