package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ExtractFromHeaders Tests
// =============================================================================

func TestExtractFromHeaders_Unauthenticated(t *testing.T) {
	// Empty headers means unauthenticated
	headers := MapHeaderGetter{}
	ctx := ExtractFromHeaders(headers)

	assert.False(t, ctx.Authenticated)
	assert.Equal(t, 0, ctx.UserID)
	assert.Empty(t, ctx.ReferenceID)
}

func TestExtractFromHeaders_EmptyUserID(t *testing.T) {
	headers := MapHeaderGetter{
		HeaderUserID: "",
	}
	ctx := ExtractFromHeaders(headers)

	assert.False(t, ctx.Authenticated)
}

func TestExtractFromHeaders_Authenticated(t *testing.T) {
	headers := MapHeaderGetter{
		HeaderUserID:         "user_12345",
		HeaderPlanID:         "plan_premium",
		HeaderKeyID:          "key_67890",
		HeaderOrganizationID: "org_abc",
	}
	ctx := ExtractFromHeaders(headers)

	assert.True(t, ctx.Authenticated)
	assert.Equal(t, "user_12345", ctx.ReferenceID)
	assert.Equal(t, 0, ctx.UserID) // UserID is resolved by middleware, not extraction
	assert.Equal(t, "plan_premium", ctx.PlanID)
	assert.Equal(t, "key_67890", ctx.KeyID)
	assert.Equal(t, "org_abc", ctx.OrganizationID)
}

func TestExtractFromHeaders_WithPlanLimits(t *testing.T) {
	headers := MapHeaderGetter{
		HeaderUserID:     "user_12345",
		HeaderPlanID:     "plan_premium",
		HeaderPlanLimits: `{"max_deployments": 10, "max_cpu_cores": 8.0, "max_memory_mb": 16384, "max_disk_mb": 102400}`,
	}
	ctx := ExtractFromHeaders(headers)

	assert.True(t, ctx.Authenticated)
	assert.Equal(t, 10, ctx.PlanLimits.MaxDeployments)
	assert.Equal(t, 8.0, ctx.PlanLimits.MaxCPUCores)
	assert.Equal(t, int64(16384), ctx.PlanLimits.MaxMemoryMB)
	assert.Equal(t, int64(102400), ctx.PlanLimits.MaxDiskMB)
}

func TestExtractFromHeaders_InvalidPlanLimits(t *testing.T) {
	// Invalid JSON should result in default limits
	headers := MapHeaderGetter{
		HeaderUserID:     "user_12345",
		HeaderPlanLimits: "not valid json",
	}
	ctx := ExtractFromHeaders(headers)

	assert.True(t, ctx.Authenticated)
	// Should use default limits when parsing fails
	assert.Equal(t, 1, ctx.PlanLimits.MaxDeployments)
}

func TestExtractFromHeaders_EmptyPlanLimits(t *testing.T) {
	headers := MapHeaderGetter{
		HeaderUserID:     "user_12345",
		HeaderPlanLimits: "",
	}
	ctx := ExtractFromHeaders(headers)

	assert.True(t, ctx.Authenticated)
	// Should use default limits when empty
	defaults := DefaultPlanLimits()
	assert.Equal(t, defaults.MaxDeployments, ctx.PlanLimits.MaxDeployments)
	assert.Equal(t, defaults.MaxCPUCores, ctx.PlanLimits.MaxCPUCores)
	assert.Equal(t, defaults.MaxMemoryMB, ctx.PlanLimits.MaxMemoryMB)
	assert.Equal(t, defaults.MaxDiskMB, ctx.PlanLimits.MaxDiskMB)
}

func TestExtractFromHeaders_PartialHeaders(t *testing.T) {
	// Only UserID header is required for authentication
	headers := MapHeaderGetter{
		HeaderUserID: "user_12345",
	}
	ctx := ExtractFromHeaders(headers)

	assert.True(t, ctx.Authenticated)
	assert.Equal(t, "user_12345", ctx.ReferenceID)
	assert.Equal(t, 0, ctx.UserID) // Resolved by middleware
	assert.Empty(t, ctx.PlanID)
	assert.Empty(t, ctx.KeyID)
	assert.Empty(t, ctx.OrganizationID)
}

// =============================================================================
// ParsePlanLimits Tests
// =============================================================================

func TestParsePlanLimits_ValidJSON(t *testing.T) {
	json := `{"max_deployments": 5, "max_cpu_cores": 4.0, "max_memory_mb": 8192, "max_disk_mb": 51200}`
	limits, err := ParsePlanLimits(json)

	require.NoError(t, err)
	assert.Equal(t, 5, limits.MaxDeployments)
	assert.Equal(t, 4.0, limits.MaxCPUCores)
	assert.Equal(t, int64(8192), limits.MaxMemoryMB)
	assert.Equal(t, int64(51200), limits.MaxDiskMB)
}

func TestParsePlanLimits_EmptyString(t *testing.T) {
	limits, err := ParsePlanLimits("")

	require.NoError(t, err)
	defaults := DefaultPlanLimits()
	assert.Equal(t, defaults, limits)
}

func TestParsePlanLimits_InvalidJSON(t *testing.T) {
	_, err := ParsePlanLimits("not json")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plan limits")
}

func TestParsePlanLimits_PartialJSON(t *testing.T) {
	// JSON with only some fields - missing fields should be zero
	json := `{"max_deployments": 3}`
	limits, err := ParsePlanLimits(json)

	require.NoError(t, err)
	assert.Equal(t, 3, limits.MaxDeployments)
	assert.Equal(t, 0.0, limits.MaxCPUCores)
	assert.Equal(t, int64(0), limits.MaxMemoryMB)
	assert.Equal(t, int64(0), limits.MaxDiskMB)
}

func TestParsePlanLimits_ZeroValues(t *testing.T) {
	json := `{"max_deployments": 0, "max_cpu_cores": 0, "max_memory_mb": 0, "max_disk_mb": 0}`
	limits, err := ParsePlanLimits(json)

	require.NoError(t, err)
	assert.Equal(t, 0, limits.MaxDeployments)
	assert.Equal(t, 0.0, limits.MaxCPUCores)
	assert.Equal(t, int64(0), limits.MaxMemoryMB)
	assert.Equal(t, int64(0), limits.MaxDiskMB)
}

// =============================================================================
// DefaultPlanLimits Tests
// =============================================================================

func TestDefaultPlanLimits(t *testing.T) {
	limits := DefaultPlanLimits()

	assert.Equal(t, 1, limits.MaxDeployments)
	assert.Equal(t, 1.0, limits.MaxCPUCores)
	assert.Equal(t, int64(1024), limits.MaxMemoryMB)
	assert.Equal(t, int64(5120), limits.MaxDiskMB)
}

// =============================================================================
// Context Storage Tests
// =============================================================================

func TestWithContext_AndFromContext(t *testing.T) {
	authCtx := Context{
		UserID:        1,
		ReferenceID:   "user_12345",
		PlanID:        "plan_premium",
		Authenticated: true,
	}

	ctx := context.Background()
	ctx = WithContext(ctx, authCtx)

	retrieved := FromContext(ctx)

	assert.True(t, retrieved.Authenticated)
	assert.Equal(t, 1, retrieved.UserID)
	assert.Equal(t, "plan_premium", retrieved.PlanID)
}

func TestFromContext_NotFound(t *testing.T) {
	ctx := context.Background()
	retrieved := FromContext(ctx)

	assert.False(t, retrieved.Authenticated)
	assert.Equal(t, 0, retrieved.UserID)
}

func TestFromContext_WrongType(t *testing.T) {
	// Store wrong type with same key
	ctx := context.WithValue(context.Background(), authContextKey, "wrong type")
	retrieved := FromContext(ctx)

	assert.False(t, retrieved.Authenticated)
}

// =============================================================================
// Context Full Round Trip
// =============================================================================

func TestContext_FullRoundTrip(t *testing.T) {
	// Simulate full flow: headers -> extract -> store -> retrieve
	headers := MapHeaderGetter{
		HeaderUserID:     "user_complete",
		HeaderPlanID:     "plan_enterprise",
		HeaderPlanLimits: `{"max_deployments": 100, "max_cpu_cores": 32.0, "max_memory_mb": 65536, "max_disk_mb": 1048576}`,
		HeaderKeyID:      "key_api",
		HeaderOrganizationID: "org_enterprise",
	}

	// Extract from headers
	authCtx := ExtractFromHeaders(headers)

	// Store in context
	ctx := WithContext(context.Background(), authCtx)

	// Retrieve from context
	retrieved := FromContext(ctx)

	// Verify all fields
	assert.True(t, retrieved.Authenticated)
	assert.Equal(t, 0, retrieved.UserID) // ExtractFromHeaders does not set UserID (resolved by middleware)
	assert.Equal(t, "user_complete", retrieved.ReferenceID)
	assert.Equal(t, "plan_enterprise", retrieved.PlanID)
	assert.Equal(t, "key_api", retrieved.KeyID)
	assert.Equal(t, "org_enterprise", retrieved.OrganizationID)
	assert.Equal(t, 100, retrieved.PlanLimits.MaxDeployments)
	assert.Equal(t, 32.0, retrieved.PlanLimits.MaxCPUCores)
	assert.Equal(t, int64(65536), retrieved.PlanLimits.MaxMemoryMB)
	assert.Equal(t, int64(1048576), retrieved.PlanLimits.MaxDiskMB)
}

// =============================================================================
// MapHeaderGetter Tests
// =============================================================================

func TestMapHeaderGetter_Get(t *testing.T) {
	m := MapHeaderGetter{
		"X-Custom-Header": "value",
	}

	assert.Equal(t, "value", m.Get("X-Custom-Header"))
	assert.Empty(t, m.Get("X-Missing"))
}
