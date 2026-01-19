// Package limits provides plan limit validation functions.
// Following F009: Billing Integration for plan limit enforcement.
// All functions are pure (no I/O) per ADR-002: Values as Boundaries.
package limits

import (
	"fmt"

	"github.com/artpar/hoster/internal/core/auth"
)

// =============================================================================
// Types
// =============================================================================

// ValidationResult represents the outcome of a limit validation check.
type ValidationResult struct {
	// Allowed indicates whether the operation is permitted within plan limits
	Allowed bool

	// Reason explains why the operation was rejected (empty if Allowed is true)
	Reason string
}

// Resources represents requested compute resources for a deployment.
type Resources struct {
	CPUCores float64
	MemoryMB int64
	DiskMB   int64
}

// CurrentUsage represents the user's current resource usage.
type CurrentUsage struct {
	// DeploymentCount is the number of active deployments
	DeploymentCount int

	// TotalCPUCores is the total CPU cores across all deployments
	TotalCPUCores float64

	// TotalMemoryMB is the total memory in MB across all deployments
	TotalMemoryMB int64

	// TotalDiskMB is the total disk space in MB across all deployments
	TotalDiskMB int64
}

// =============================================================================
// Validation Functions
// =============================================================================

// ValidateDeploymentCreation checks if a user can create a new deployment
// given their plan limits and current usage.
func ValidateDeploymentCreation(
	limits auth.PlanLimits,
	usage CurrentUsage,
	requested Resources,
) ValidationResult {
	// Check deployment count limit
	if usage.DeploymentCount >= limits.MaxDeployments {
		return ValidationResult{
			Allowed: false,
			Reason:  fmt.Sprintf("deployment limit reached: %d/%d", usage.DeploymentCount, limits.MaxDeployments),
		}
	}

	// Check CPU limit
	newTotalCPU := usage.TotalCPUCores + requested.CPUCores
	if newTotalCPU > limits.MaxCPUCores {
		return ValidationResult{
			Allowed: false,
			Reason:  fmt.Sprintf("CPU limit would be exceeded: %.1f/%.1f cores", newTotalCPU, limits.MaxCPUCores),
		}
	}

	// Check memory limit
	newTotalMemory := usage.TotalMemoryMB + requested.MemoryMB
	if newTotalMemory > limits.MaxMemoryMB {
		return ValidationResult{
			Allowed: false,
			Reason:  fmt.Sprintf("memory limit would be exceeded: %d/%d MB", newTotalMemory, limits.MaxMemoryMB),
		}
	}

	// Check disk limit
	newTotalDisk := usage.TotalDiskMB + requested.DiskMB
	if newTotalDisk > limits.MaxDiskMB {
		return ValidationResult{
			Allowed: false,
			Reason:  fmt.Sprintf("disk limit would be exceeded: %d/%d MB", newTotalDisk, limits.MaxDiskMB),
		}
	}

	return ValidationResult{Allowed: true}
}

// ValidateDeploymentStart checks if a user can start a deployment
// given their plan limits and current usage.
// This is a lighter check than creation - only validates running deployments.
func ValidateDeploymentStart(
	limits auth.PlanLimits,
	runningDeployments int,
) ValidationResult {
	// Note: MaxDeployments is for total deployments (created), not running.
	// We allow starting any created deployment within the plan limits.
	// If more granular control is needed (e.g., max running), add MaxRunningDeployments to PlanLimits.
	return ValidationResult{Allowed: true}
}

// ValidateCapability checks if a user's plan allows a specific node capability.
func ValidateCapability(limits auth.PlanLimits, capability string) ValidationResult {
	if len(limits.AllowedCapabilities) == 0 {
		// If no capabilities specified, allow all (backward compatibility)
		return ValidationResult{Allowed: true}
	}

	for _, allowed := range limits.AllowedCapabilities {
		if allowed == capability {
			return ValidationResult{Allowed: true}
		}
	}

	return ValidationResult{
		Allowed: false,
		Reason:  fmt.Sprintf("capability '%s' not allowed by plan", capability),
	}
}

// =============================================================================
// Convenience Methods
// =============================================================================

// Ok returns true if the validation passed.
func (r ValidationResult) Ok() bool {
	return r.Allowed
}

// Error returns the reason as an error if validation failed, nil otherwise.
func (r ValidationResult) Error() error {
	if r.Allowed {
		return nil
	}
	return fmt.Errorf("plan limit exceeded: %s", r.Reason)
}
