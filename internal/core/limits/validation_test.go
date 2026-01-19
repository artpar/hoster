package limits

import (
	"testing"

	"github.com/artpar/hoster/internal/core/auth"
	"github.com/stretchr/testify/assert"
)

func TestValidateDeploymentCreation_WithinLimits(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 5,
		MaxCPUCores:    4.0,
		MaxMemoryMB:    8192,
		MaxDiskMB:      51200,
	}
	usage := CurrentUsage{
		DeploymentCount: 2,
		TotalCPUCores:   2.0,
		TotalMemoryMB:   4096,
		TotalDiskMB:     20480,
	}
	requested := Resources{
		CPUCores: 1.0,
		MemoryMB: 2048,
		DiskMB:   10240,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.True(t, result.Allowed)
	assert.Empty(t, result.Reason)
	assert.True(t, result.Ok())
	assert.NoError(t, result.Error())
}

func TestValidateDeploymentCreation_DeploymentLimitReached(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 3,
		MaxCPUCores:    4.0,
		MaxMemoryMB:    8192,
		MaxDiskMB:      51200,
	}
	usage := CurrentUsage{
		DeploymentCount: 3, // At limit
		TotalCPUCores:   2.0,
		TotalMemoryMB:   4096,
		TotalDiskMB:     20480,
	}
	requested := Resources{
		CPUCores: 0.5,
		MemoryMB: 512,
		DiskMB:   1024,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "deployment limit reached")
	assert.Contains(t, result.Reason, "3/3")
	assert.False(t, result.Ok())
	assert.Error(t, result.Error())
}

func TestValidateDeploymentCreation_CPULimitExceeded(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 10,
		MaxCPUCores:    4.0,
		MaxMemoryMB:    8192,
		MaxDiskMB:      51200,
	}
	usage := CurrentUsage{
		DeploymentCount: 2,
		TotalCPUCores:   3.5,
		TotalMemoryMB:   4096,
		TotalDiskMB:     20480,
	}
	requested := Resources{
		CPUCores: 1.0, // Would exceed 4.0
		MemoryMB: 512,
		DiskMB:   1024,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "CPU limit would be exceeded")
	assert.Contains(t, result.Reason, "4.5/4.0")
}

func TestValidateDeploymentCreation_MemoryLimitExceeded(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 10,
		MaxCPUCores:    4.0,
		MaxMemoryMB:    4096,
		MaxDiskMB:      51200,
	}
	usage := CurrentUsage{
		DeploymentCount: 2,
		TotalCPUCores:   2.0,
		TotalMemoryMB:   3072,
		TotalDiskMB:     20480,
	}
	requested := Resources{
		CPUCores: 0.5,
		MemoryMB: 2048, // Would exceed 4096
		DiskMB:   1024,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "memory limit would be exceeded")
	assert.Contains(t, result.Reason, "5120/4096")
}

func TestValidateDeploymentCreation_DiskLimitExceeded(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 10,
		MaxCPUCores:    4.0,
		MaxMemoryMB:    8192,
		MaxDiskMB:      10240,
	}
	usage := CurrentUsage{
		DeploymentCount: 2,
		TotalCPUCores:   2.0,
		TotalMemoryMB:   4096,
		TotalDiskMB:     8192,
	}
	requested := Resources{
		CPUCores: 0.5,
		MemoryMB: 512,
		DiskMB:   4096, // Would exceed 10240
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "disk limit would be exceeded")
	assert.Contains(t, result.Reason, "12288/10240")
}

func TestValidateDeploymentCreation_FirstDeployment(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 1,
		MaxCPUCores:    1.0,
		MaxMemoryMB:    1024,
		MaxDiskMB:      5120,
	}
	usage := CurrentUsage{
		DeploymentCount: 0, // No existing deployments
		TotalCPUCores:   0,
		TotalMemoryMB:   0,
		TotalDiskMB:     0,
	}
	requested := Resources{
		CPUCores: 0.5,
		MemoryMB: 512,
		DiskMB:   1024,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.True(t, result.Allowed)
}

func TestValidateDeploymentCreation_ExactlyAtLimits(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 5,
		MaxCPUCores:    4.0,
		MaxMemoryMB:    8192,
		MaxDiskMB:      51200,
	}
	usage := CurrentUsage{
		DeploymentCount: 4, // One slot left
		TotalCPUCores:   3.0,
		TotalMemoryMB:   7168,
		TotalDiskMB:     46080,
	}
	requested := Resources{
		CPUCores: 1.0, // Exactly at 4.0
		MemoryMB: 1024,
		DiskMB:   5120,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.True(t, result.Allowed)
}

func TestValidateDeploymentCreation_ZeroLimits(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 0, // Zero means no deployments allowed
		MaxCPUCores:    0,
		MaxMemoryMB:    0,
		MaxDiskMB:      0,
	}
	usage := CurrentUsage{}
	requested := Resources{
		CPUCores: 0.5,
		MemoryMB: 512,
		DiskMB:   1024,
	}

	result := ValidateDeploymentCreation(limits, usage, requested)

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "deployment limit reached")
}

func TestValidateCapability_Allowed(t *testing.T) {
	limits := auth.PlanLimits{
		AllowedCapabilities: []string{"standard", "high-memory"},
	}

	result := ValidateCapability(limits, "standard")

	assert.True(t, result.Allowed)
}

func TestValidateCapability_NotAllowed(t *testing.T) {
	limits := auth.PlanLimits{
		AllowedCapabilities: []string{"standard"},
	}

	result := ValidateCapability(limits, "gpu")

	assert.False(t, result.Allowed)
	assert.Contains(t, result.Reason, "capability 'gpu' not allowed")
}

func TestValidateCapability_EmptyAllowsAll(t *testing.T) {
	limits := auth.PlanLimits{
		AllowedCapabilities: nil, // Empty
	}

	result := ValidateCapability(limits, "any-capability")

	assert.True(t, result.Allowed)
}

func TestValidateDeploymentStart_AlwaysAllowed(t *testing.T) {
	limits := auth.PlanLimits{
		MaxDeployments: 3,
	}

	result := ValidateDeploymentStart(limits, 3) // Already at max running

	assert.True(t, result.Allowed)
}

func TestValidationResult_Error(t *testing.T) {
	allowed := ValidationResult{Allowed: true}
	assert.NoError(t, allowed.Error())

	denied := ValidationResult{Allowed: false, Reason: "limit exceeded"}
	err := denied.Error()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan limit exceeded")
	assert.Contains(t, err.Error(), "limit exceeded")
}
