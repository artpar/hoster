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

func authenticatedContext(userID int) Context {
	return Context{
		UserID:        userID,
		PlanID:        "plan_default",
		PlanLimits:    DefaultPlanLimits(),
		Authenticated: true,
	}
}

func contextWithLimits(userID int, limits PlanLimits) Context {
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

func sampleTemplate(creatorID int, published bool) domain.Template {
	return domain.Template{
		ReferenceID: "tmpl_test",
		Name:        "Test Template",
		Slug:        "test-template",
		Version:     "1.0.0",
		CreatorID:   creatorID,
		Published:   published,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func sampleDeployment(customerID int) domain.Deployment {
	return domain.Deployment{
		ReferenceID: "deploy_test",
		Name:        "Test Deployment",
		CustomerID:  customerID,
		TemplateID:  1,
		Status:      domain.StatusRunning,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// =============================================================================
// Template Authorization Tests
// =============================================================================

func TestCanViewTemplate_PublishedVisible(t *testing.T) {
	template := sampleTemplate(123, true)

	// Unauthenticated can view published
	assert.True(t, CanViewTemplate(unauthenticatedContext(), template))

	// Different user can view published
	assert.True(t, CanViewTemplate(authenticatedContext(999), template))

	// Creator can view published
	assert.True(t, CanViewTemplate(authenticatedContext(123), template))
}

func TestCanViewTemplate_UnpublishedCreatorOnly(t *testing.T) {
	template := sampleTemplate(123, false)

	// Unauthenticated cannot view unpublished
	assert.False(t, CanViewTemplate(unauthenticatedContext(), template))

	// Different user cannot view unpublished
	assert.False(t, CanViewTemplate(authenticatedContext(999), template))

	// Creator can view unpublished
	assert.True(t, CanViewTemplate(authenticatedContext(123), template))
}

func TestCanModifyTemplate_CreatorOnly(t *testing.T) {
	template := sampleTemplate(123, true)

	// Unauthenticated cannot modify
	assert.False(t, CanModifyTemplate(unauthenticatedContext(), template))

	// Different user cannot modify
	assert.False(t, CanModifyTemplate(authenticatedContext(999), template))

	// Creator can modify
	assert.True(t, CanModifyTemplate(authenticatedContext(123), template))
}

func TestCanDeleteTemplate_CreatorOnly(t *testing.T) {
	template := sampleTemplate(123, true)

	// Unauthenticated cannot delete
	assert.False(t, CanDeleteTemplate(unauthenticatedContext(), template))

	// Different user cannot delete
	assert.False(t, CanDeleteTemplate(authenticatedContext(999), template))

	// Creator can delete
	assert.True(t, CanDeleteTemplate(authenticatedContext(123), template))
}

func TestCanPublishTemplate_CreatorOnly(t *testing.T) {
	template := sampleTemplate(123, false)

	// Unauthenticated cannot publish
	assert.False(t, CanPublishTemplate(unauthenticatedContext(), template))

	// Different user cannot publish
	assert.False(t, CanPublishTemplate(authenticatedContext(999), template))

	// Creator can publish
	assert.True(t, CanPublishTemplate(authenticatedContext(123), template))
}

// =============================================================================
// Deployment Authorization Tests
// =============================================================================

func TestCanViewDeployment_OwnerOnly(t *testing.T) {
	deployment := sampleDeployment(456)

	// Unauthenticated cannot view
	assert.False(t, CanViewDeployment(unauthenticatedContext(), deployment))

	// Different user cannot view
	assert.False(t, CanViewDeployment(authenticatedContext(999), deployment))

	// Owner can view
	assert.True(t, CanViewDeployment(authenticatedContext(456), deployment))
}

func TestCanManageDeployment_OwnerOnly(t *testing.T) {
	deployment := sampleDeployment(456)

	// Unauthenticated cannot manage
	assert.False(t, CanManageDeployment(unauthenticatedContext(), deployment))

	// Different user cannot manage
	assert.False(t, CanManageDeployment(authenticatedContext(999), deployment))

	// Owner can manage
	assert.True(t, CanManageDeployment(authenticatedContext(456), deployment))
}

func TestCanDeleteDeployment_OwnerOnly(t *testing.T) {
	deployment := sampleDeployment(456)

	// Unauthenticated cannot delete
	assert.False(t, CanDeleteDeployment(unauthenticatedContext(), deployment))

	// Different user cannot delete
	assert.False(t, CanDeleteDeployment(authenticatedContext(999), deployment))

	// Owner can delete
	assert.True(t, CanDeleteDeployment(authenticatedContext(456), deployment))
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
	ctx := contextWithLimits(123, PlanLimits{MaxDeployments: 5})

	ok, reason := CanCreateDeployment(ctx, 0)
	assert.True(t, ok)
	assert.Empty(t, reason)

	ok, reason = CanCreateDeployment(ctx, 4)
	assert.True(t, ok)
	assert.Empty(t, reason)
}

func TestCanCreateDeployment_AtLimit(t *testing.T) {
	ctx := contextWithLimits(123, PlanLimits{MaxDeployments: 5})

	ok, reason := CanCreateDeployment(ctx, 5)

	assert.False(t, ok)
	assert.Equal(t, "plan limit reached: max 5 deployments", reason)
}

func TestCanCreateDeployment_OverLimit(t *testing.T) {
	ctx := contextWithLimits(123, PlanLimits{MaxDeployments: 5})

	ok, reason := CanCreateDeployment(ctx, 10)

	assert.False(t, ok)
	assert.Contains(t, reason, "plan limit reached")
}

func TestCanCreateDeployment_DefaultLimits(t *testing.T) {
	// Default limits allow only 1 deployment
	ctx := authenticatedContext(123)

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
	ctx := contextWithLimits(123, PlanLimits{
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
	ctx := contextWithLimits(123, PlanLimits{
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
	ctx := contextWithLimits(123, PlanLimits{
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
	ctx := contextWithLimits(123, PlanLimits{
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
	ctx := contextWithLimits(123, PlanLimits{
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
	ctx := contextWithLimits(123, PlanLimits{
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
	ok, reason := RequireAuthentication(authenticatedContext(123))

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
	// Context with zero UserID should be treated as unauthenticated
	ctx := Context{
		UserID:        0,
		Authenticated: true, // Buggy state - authenticated but no UserID
	}
	template := sampleTemplate(123, false)

	// Should not match creator
	assert.False(t, CanModifyTemplate(ctx, template))
}

func TestEmptyCreatorID_Template(t *testing.T) {
	// Template with zero CreatorID
	template := domain.Template{
		ReferenceID: "tmpl_test",
		CreatorID:   0,
		Published:   false,
	}
	ctx := authenticatedContext(0)

	// Zero matches zero
	assert.True(t, CanModifyTemplate(ctx, template))
}

func TestEmptyCustomerID_Deployment(t *testing.T) {
	// Deployment with zero CustomerID
	deployment := domain.Deployment{
		ReferenceID: "deploy_test",
		CustomerID:  0,
	}
	ctx := authenticatedContext(0)

	// Zero matches zero
	assert.True(t, CanViewDeployment(ctx, deployment))
}

// =============================================================================
// Node Authorization Tests
// =============================================================================

func sampleNode(creatorID int) domain.Node {
	return domain.Node{
		ReferenceID:  "node_test",
		Name:         "Test Node",
		CreatorID:    creatorID,
		SSHHost:      "192.168.1.100",
		SSHPort:      22,
		SSHUser:      "deploy",
		Status:       domain.NodeStatusOnline,
		Capabilities: []string{"standard"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func sampleSSHKey(creatorID int) domain.SSHKey {
	return domain.SSHKey{
		ReferenceID: "sshkey_test",
		CreatorID:   creatorID,
		Name:        "Test Key",
		Fingerprint: "SHA256:abc123",
		CreatedAt:   time.Now(),
	}
}

func TestCanViewNode_CreatorOnly(t *testing.T) {
	node := sampleNode(123)

	// Unauthenticated cannot view
	assert.False(t, CanViewNode(unauthenticatedContext(), node))

	// Different user cannot view
	assert.False(t, CanViewNode(authenticatedContext(999), node))

	// Creator can view
	assert.True(t, CanViewNode(authenticatedContext(123), node))
}

func TestCanManageNode_CreatorOnly(t *testing.T) {
	node := sampleNode(123)

	// Unauthenticated cannot manage
	assert.False(t, CanManageNode(unauthenticatedContext(), node))

	// Different user cannot manage
	assert.False(t, CanManageNode(authenticatedContext(999), node))

	// Creator can manage
	assert.True(t, CanManageNode(authenticatedContext(123), node))
}

func TestCanCreateNode_AuthenticatedOnly(t *testing.T) {
	// Unauthenticated cannot create
	assert.False(t, CanCreateNode(unauthenticatedContext()))

	// Any authenticated user can create
	assert.True(t, CanCreateNode(authenticatedContext(1)))
}

// =============================================================================
// SSH Key Authorization Tests
// =============================================================================

func TestCanViewSSHKey_CreatorOnly(t *testing.T) {
	key := sampleSSHKey(123)

	// Unauthenticated cannot view
	assert.False(t, CanViewSSHKey(unauthenticatedContext(), key))

	// Different user cannot view
	assert.False(t, CanViewSSHKey(authenticatedContext(999), key))

	// Creator can view
	assert.True(t, CanViewSSHKey(authenticatedContext(123), key))
}

func TestCanManageSSHKey_CreatorOnly(t *testing.T) {
	key := sampleSSHKey(123)

	// Unauthenticated cannot manage
	assert.False(t, CanManageSSHKey(unauthenticatedContext(), key))

	// Different user cannot manage
	assert.False(t, CanManageSSHKey(authenticatedContext(999), key))

	// Creator can manage
	assert.True(t, CanManageSSHKey(authenticatedContext(123), key))
}

func TestCanCreateSSHKey_AuthenticatedOnly(t *testing.T) {
	// Unauthenticated cannot create
	assert.False(t, CanCreateSSHKey(unauthenticatedContext()))

	// Any authenticated user can create
	assert.True(t, CanCreateSSHKey(authenticatedContext(1)))
}
