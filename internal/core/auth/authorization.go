// Package auth provides authentication context and authorization functions.
// Following ADR-005: APIGate Integration for Authentication and Billing
package auth

import (
	"fmt"

	"github.com/artpar/hoster/internal/core/domain"
)

// =============================================================================
// Template Authorization
// =============================================================================

// CanViewTemplate checks if the user can view a template.
// Published templates are visible to everyone.
// Unpublished templates are only visible to their creator.
func CanViewTemplate(ctx Context, template domain.Template) bool {
	// Published templates are visible to all
	if template.Published {
		return true
	}
	// Unpublished templates only visible to creator
	return ctx.Authenticated && ctx.UserID == template.CreatorID
}

// CanModifyTemplate checks if the user can modify a template.
// Only the creator can modify their templates.
func CanModifyTemplate(ctx Context, template domain.Template) bool {
	return ctx.Authenticated && ctx.UserID == template.CreatorID
}

// CanDeleteTemplate checks if the user can delete a template.
// Only the creator can delete their templates.
func CanDeleteTemplate(ctx Context, template domain.Template) bool {
	return ctx.Authenticated && ctx.UserID == template.CreatorID
}

// CanPublishTemplate checks if the user can publish a template.
// Only the creator can publish their templates.
func CanPublishTemplate(ctx Context, template domain.Template) bool {
	return ctx.Authenticated && ctx.UserID == template.CreatorID
}

// =============================================================================
// Deployment Authorization
// =============================================================================

// CanViewDeployment checks if the user can view a deployment.
// Only the deployment's owner (customer) can view it.
func CanViewDeployment(ctx Context, deployment domain.Deployment) bool {
	return ctx.Authenticated && ctx.UserID == deployment.CustomerID
}

// CanManageDeployment checks if the user can manage a deployment (start/stop/restart).
// Only the deployment's owner (customer) can manage it.
func CanManageDeployment(ctx Context, deployment domain.Deployment) bool {
	return ctx.Authenticated && ctx.UserID == deployment.CustomerID
}

// CanDeleteDeployment checks if the user can delete a deployment.
// Only the deployment's owner (customer) can delete it.
func CanDeleteDeployment(ctx Context, deployment domain.Deployment) bool {
	return ctx.Authenticated && ctx.UserID == deployment.CustomerID
}

// =============================================================================
// Plan Limit Authorization
// =============================================================================

// CanCreateDeployment checks if the user can create another deployment based on plan limits.
// Returns (true, "") if allowed, or (false, reason) if not allowed.
func CanCreateDeployment(ctx Context, currentDeploymentCount int) (bool, string) {
	if !ctx.Authenticated {
		return false, "authentication required"
	}
	if currentDeploymentCount >= ctx.PlanLimits.MaxDeployments {
		return false, fmt.Sprintf("plan limit reached: max %d deployments", ctx.PlanLimits.MaxDeployments)
	}
	return true, ""
}

// Resources mirrors domain.Resources for limit validation.
// This allows the auth package to not depend on domain for simple resource checks.
type Resources struct {
	CPUCores float64
	MemoryMB int64
	DiskMB   int64
}

// ResourcesFromDomain converts domain.Resources to auth.Resources.
func ResourcesFromDomain(r domain.Resources) Resources {
	return Resources{
		CPUCores: r.CPUCores,
		MemoryMB: r.MemoryMB,
		DiskMB:   r.DiskMB,
	}
}

// ValidateResourceLimits checks if the requested resources are within plan limits.
// It validates that currentUsage + requested does not exceed plan limits.
// Returns (true, "") if allowed, or (false, reason) if not allowed.
func ValidateResourceLimits(ctx Context, currentUsage, requested Resources) (bool, string) {
	if !ctx.Authenticated {
		return false, "authentication required"
	}

	total := Resources{
		CPUCores: currentUsage.CPUCores + requested.CPUCores,
		MemoryMB: currentUsage.MemoryMB + requested.MemoryMB,
		DiskMB:   currentUsage.DiskMB + requested.DiskMB,
	}

	if total.CPUCores > ctx.PlanLimits.MaxCPUCores {
		return false, fmt.Sprintf("CPU limit exceeded: %.1f/%.1f cores", total.CPUCores, ctx.PlanLimits.MaxCPUCores)
	}
	if total.MemoryMB > ctx.PlanLimits.MaxMemoryMB {
		return false, fmt.Sprintf("memory limit exceeded: %dMB/%dMB", total.MemoryMB, ctx.PlanLimits.MaxMemoryMB)
	}
	if total.DiskMB > ctx.PlanLimits.MaxDiskMB {
		return false, fmt.Sprintf("disk limit exceeded: %dMB/%dMB", total.DiskMB, ctx.PlanLimits.MaxDiskMB)
	}

	return true, ""
}

// =============================================================================
// Generic Helpers
// =============================================================================

// RequireAuthentication checks if the context is authenticated.
// Returns (true, "") if authenticated, or (false, "authentication required") if not.
func RequireAuthentication(ctx Context) (bool, string) {
	if !ctx.Authenticated {
		return false, "authentication required"
	}
	return true, ""
}
