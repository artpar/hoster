package auth

import (
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Test Helpers
// =============================================================================

func authenticatedContext(userID string) Context {
	return Context{
		UserID:        userID,
		PlanID:        "plan_default",
		PlanLimits:    DefaultPlanLimits(),
		Authenticated: true,
	}
}

func contextWithLimits(userID string, limits PlanLimits) Context {
	return Context{
		UserID:        userID,
		PlanID:        "plan_custom",
		PlanLimits:    limits,
		Authenticated: true,
	}
}

func unauthenticatedContext() Context {
	return Context{Authenticated: false}
}

func sampleTemplate(creatorID string, published bool) domain.Template {
	return domain.Template{
		ID:        "tmpl_test",
		Name:      "Test Template",
		Slug:      "test-template",
		Version:   "1.0.0",
		CreatorID: creatorID,
		Published: published,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func sampleDeployment(customerID string) domain.Deployment {
	return domain.Deployment{
		ID:         "deploy_test",
		Name:       "Test Deployment",
		CustomerID: customerID,
		TemplateID: "tmpl_test",
		Status:     domain.StatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// =============================================================================
// Template Authorization Tests
// =============================================================================

func TestCanViewTemplate_PublishedVisible(t *testing.T) {
	template := sampleTemplate("creator_123", true)

	// Unauthenticated can view published
	assert.True(t, CanViewTemplate(unauthenticatedContext(), template))

	// Different user can view published
	assert.True(t, CanViewTemplate(authenticatedContext("other_user"), template))

	// Creator can view published
	assert.True(t, CanViewTemplate(authenticatedContext("creator_123"), template))
}

func TestCanViewTemplate_UnpublishedCreatorOnly(t *testing.T) {
	template := sampleTemplate("creator_123", false)

	// Unauthenticated cannot view unpublished
	assert.False(t, CanViewTemplate(unauthenticatedContext(), template))

	// Different user cannot view unpublished
	assert.False(t, CanViewTemplate(authenticatedContext("other_user"), template))

	// Creator can view unpublished
	assert.True(t, CanViewTemplate(authenticatedContext("creator_123"), template))
}

func TestCanModifyTemplate_CreatorOnly(t *testing.T) {
	template := sampleTemplate("creator_123", true)

	// Unauthenticated cannot modify
	assert.False(t, CanModifyTemplate(unauthenticatedContext(), template))

	// Different user cannot modify
	assert.False(t, CanModifyTemplate(authenticatedContext("other_user"), template))

	// Creator can modify
	assert.True(t, CanModifyTemplate(authenticatedContext("creator_123"), template))
}

func TestCanDeleteTemplate_CreatorOnly(t *testing.T) {
	template := sampleTemplate("creator_123", true)

	// Unauthenticated cannot delete
	assert.False(t, CanDeleteTemplate(unauthenticatedContext(), template))

	// Different user cannot delete
	assert.False(t, CanDeleteTemplate(authenticatedContext("other_user"), template))

	// Creator can delete
	assert.True(t, CanDeleteTemplate(authenticatedContext("creator_123"), template))
}

func TestCanPublishTemplate_CreatorOnly(t *testing.T) {
	template := sampleTemplate("creator_123", false)

	// Unauthenticated cannot publish
	assert.False(t, CanPublishTemplate(unauthenticatedContext(), template))

	// Different user cannot publish
	assert.False(t, CanPublishTemplate(authenticatedContext("other_user"), template))

	// Creator can publish
	assert.True(t, CanPublishTemplate(authenticatedContext("creator_123"), template))
}

// =============================================================================
// Deployment Authorization Tests
// =============================================================================

func TestCanViewDeployment_OwnerOnly(t *testing.T) {
	deployment := sampleDeployment("customer_456")

	// Unauthenticated cannot view
	assert.False(t, CanViewDeployment(unauthenticatedContext(), deployment))

	// Different user cannot view
	assert.False(t, CanViewDeployment(authenticatedContext("other_user"), deployment))

	// Owner can view
	assert.True(t, CanViewDeployment(authenticatedContext("customer_456"), deployment))
}

func TestCanManageDeployment_OwnerOnly(t *testing.T) {
	deployment := sampleDeployment("customer_456")

	// Unauthenticated cannot manage
	assert.False(t, CanManageDeployment(unauthenticatedContext(), deployment))

	// Different user cannot manage
	assert.False(t, CanManageDeployment(authenticatedContext("other_user"), deployment))

	// Owner can manage
	assert.True(t, CanManageDeployment(authenticatedContext("customer_456"), deployment))
}

func TestCanDeleteDeployment_OwnerOnly(t *testing.T) {
	deployment := sampleDeployment("customer_456")

	// Unauthenticated cannot delete
	assert.False(t, CanDeleteDeployment(unauthenticatedContext(), deployment))

	// Different user cannot delete
	assert.False(t, CanDeleteDeployment(authenticatedContext("other_user"), deployment))

	// Owner can delete
	assert.True(t, CanDeleteDeployment(authenticatedContext("customer_456"), deployment))
}

// =============================================================================
// Plan Limit Authorization Tests
// =============================================================================

func TestCanCreateDeployment_Unauthenticated(t *testing.T) {
	ok, reason := CanCreateDeployment(unauthenticatedContext(), 0)

	assert.False(t, ok)
	assert.Equal(t, "authentication required", reason)
}

func TestCanCreateDeployment_WithinLimit(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{MaxDeployments: 5})

	ok, reason := CanCreateDeployment(ctx, 0)
	assert.True(t, ok)
	assert.Empty(t, reason)

	ok, reason = CanCreateDeployment(ctx, 4)
	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestCanCreateDeployment_AtLimit(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{MaxDeployments: 5})

	ok, reason := CanCreateDeployment(ctx, 5)

	assert.False(t, ok)
	assert.Equal(t, "plan limit reached: max 5 deployments", reason)
}

func TestCanCreateDeployment_OverLimit(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{MaxDeployments: 5})

	ok, reason := CanCreateDeployment(ctx, 10)

	assert.False(t, ok)
	assert.Contains(t, reason, "plan limit reached")
}

func TestCanCreateDeployment_DefaultLimits(t *testing.T) {
	// Default limits allow only 1 deployment
	ctx := authenticatedContext("user_123")

	ok, _ := CanCreateDeployment(ctx, 0)
	assert.True(t, ok)

	ok, reason := CanCreateDeployment(ctx, 1)
	assert.False(t, ok)
	assert.Equal(t, "plan limit reached: max 1 deployments", reason)
}

// =============================================================================
// Resource Limit Validation Tests
// =============================================================================

func TestValidateResourceLimits_Unauthenticated(t *testing.T) {
	ok, reason := ValidateResourceLimits(unauthenticatedContext(), Resources{}, Resources{})

	assert.False(t, ok)
	assert.Equal(t, "authentication required", reason)
}

func TestValidateResourceLimits_WithinLimits(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{
		MaxCPUCores: 4.0,
		MaxMemoryMB: 8192,
		MaxDiskMB:   51200,
	})

	current := Resources{CPUCores: 1.0, MemoryMB: 2048, DiskMB: 10240}
	requested := Resources{CPUCores: 1.0, MemoryMB: 2048, DiskMB: 10240}

	ok, reason := ValidateResourceLimits(ctx, current, requested)

	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestValidateResourceLimits_ExactlyAtLimits(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{
		MaxCPUCores: 4.0,
		MaxMemoryMB: 8192,
		MaxDiskMB:   51200,
	})

	current := Resources{CPUCores: 2.0, MemoryMB: 4096, DiskMB: 25600}
	requested := Resources{CPUCores: 2.0, MemoryMB: 4096, DiskMB: 25600}

	ok, reason := ValidateResourceLimits(ctx, current, requested)

	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestValidateResourceLimits_CPUExceeded(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{
		MaxCPUCores: 4.0,
		MaxMemoryMB: 8192,
		MaxDiskMB:   51200,
	})

	current := Resources{CPUCores: 2.0, MemoryMB: 2048, DiskMB: 10240}
	requested := Resources{CPUCores: 3.0, MemoryMB: 2048, DiskMB: 10240}

	ok, reason := ValidateResourceLimits(ctx, current, requested)

	assert.False(t, ok)
	assert.Contains(t, reason, "CPU limit exceeded")
	assert.Contains(t, reason, "5.0/4.0 cores")
}

func TestValidateResourceLimits_MemoryExceeded(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{
		MaxCPUCores: 4.0,
		MaxMemoryMB: 8192,
		MaxDiskMB:   51200,
	})

	current := Resources{CPUCores: 1.0, MemoryMB: 4096, DiskMB: 10240}
	requested := Resources{CPUCores: 1.0, MemoryMB: 5000, DiskMB: 10240}

	ok, reason := ValidateResourceLimits(ctx, current, requested)

	assert.False(t, ok)
	assert.Contains(t, reason, "memory limit exceeded")
	assert.Contains(t, reason, "9096MB/8192MB")
}

func TestValidateResourceLimits_DiskExceeded(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{
		MaxCPUCores: 4.0,
		MaxMemoryMB: 8192,
		MaxDiskMB:   51200,
	})

	current := Resources{CPUCores: 1.0, MemoryMB: 2048, DiskMB: 30000}
	requested := Resources{CPUCores: 1.0, MemoryMB: 2048, DiskMB: 30000}

	ok, reason := ValidateResourceLimits(ctx, current, requested)

	assert.False(t, ok)
	assert.Contains(t, reason, "disk limit exceeded")
	assert.Contains(t, reason, "60000MB/51200MB")
}

func TestValidateResourceLimits_ZeroCurrentUsage(t *testing.T) {
	ctx := contextWithLimits("user_123", PlanLimits{
		MaxCPUCores: 2.0,
		MaxMemoryMB: 4096,
		MaxDiskMB:   20480,
	})

	current := Resources{CPUCores: 0, MemoryMB: 0, DiskMB: 0}
	requested := Resources{CPUCores: 1.0, MemoryMB: 2048, DiskMB: 10240}

	ok, reason := ValidateResourceLimits(ctx, current, requested)

	assert.True(t, ok)
	assert.Empty(t, reason)
}

// =============================================================================
// ResourcesFromDomain Tests
// =============================================================================

func TestResourcesFromDomain(t *testing.T) {
	domainRes := domain.Resources{
		CPUCores: 2.5,
		MemoryMB: 4096,
		DiskMB:   20480,
	}

	authRes := ResourcesFromDomain(domainRes)

	assert.Equal(t, 2.5, authRes.CPUCores)
	assert.Equal(t, int64(4096), authRes.MemoryMB)
	assert.Equal(t, int64(20480), authRes.DiskMB)
}

// =============================================================================
// RequireAuthentication Tests
// =============================================================================

func TestRequireAuthentication_Authenticated(t *testing.T) {
	ok, reason := RequireAuthentication(authenticatedContext("user_123"))

	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestRequireAuthentication_Unauthenticated(t *testing.T) {
	ok, reason := RequireAuthentication(unauthenticatedContext())

	assert.False(t, ok)
	assert.Equal(t, "authentication required", reason)
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestEmptyUserID_Template(t *testing.T) {
	// Context with empty UserID should be treated as unauthenticated
	ctx := Context{
		UserID:        "",
		Authenticated: true, // Buggy state - authenticated but no UserID
	}
	template := sampleTemplate("creator_123", false)

	// Should not match creator
	assert.False(t, CanModifyTemplate(ctx, template))
}

func TestEmptyCreatorID_Template(t *testing.T) {
	// Template with empty CreatorID
	template := domain.Template{
		ID:        "tmpl_test",
		CreatorID: "",
		Published: false,
	}
	ctx := authenticatedContext("")

	// Empty matches empty
	assert.True(t, CanModifyTemplate(ctx, template))
}

func TestEmptyCustomerID_Deployment(t *testing.T) {
	// Deployment with empty CustomerID
	deployment := domain.Deployment{
		ID:         "deploy_test",
		CustomerID: "",
	}
	ctx := authenticatedContext("")

	// Empty matches empty
	assert.True(t, CanViewDeployment(ctx, deployment))
}
