package deployment

import (
	"time"

	"github.com/artpar/hoster/internal/core/compose"
)

// =============================================================================
// Container Plan Types
// =============================================================================

// ContainerPlan represents a planned container configuration.
// This is the pure output of planning, ready for the shell to execute.
type ContainerPlan struct {
	Name          string
	Image         string
	Command       []string
	Entrypoint    []string
	Env           map[string]string
	Labels        map[string]string
	Ports         []PortPlan
	Volumes       []VolumePlan
	Networks      []string
	RestartPolicy RestartPolicyPlan
	Resources     ResourcePlan
	HealthCheck   *HealthCheckPlan
}

// PortPlan represents a planned port binding.
type PortPlan struct {
	ContainerPort int
	HostPort      int
	Protocol      string
	HostIP        string
}

// VolumePlan represents a planned volume mount.
type VolumePlan struct {
	Source   string
	Target   string
	ReadOnly bool
}

// RestartPolicyPlan represents a restart policy.
type RestartPolicyPlan struct {
	Name              string
	MaximumRetryCount int
}

// ResourcePlan represents resource limits.
type ResourcePlan struct {
	CPULimit    float64
	MemoryLimit int64
}

// HealthCheckPlan represents a health check configuration.
type HealthCheckPlan struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int
	StartPeriod time.Duration
}

// =============================================================================
// Builder Parameter Types
// =============================================================================

// BuildContainerPlanParams contains all inputs for building a container plan.
type BuildContainerPlanParams struct {
	DeploymentID string
	TemplateID   string
	ServiceName  string
	Service      compose.Service
	Variables    map[string]string
	NetworkName  string
	Volumes      []compose.Volume
}

// =============================================================================
// Hoster Container Labels
// =============================================================================

// Label keys used for Hoster container identification.
const (
	LabelManaged    = "com.hoster.managed"
	LabelDeployment = "com.hoster.deployment"
	LabelTemplate   = "com.hoster.template"
	LabelService    = "com.hoster.service"
)
