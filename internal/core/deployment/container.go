package deployment

import (
	"time"

	"github.com/artpar/hoster/internal/core/compose"
)

// =============================================================================
// Container Plan Building Functions
// =============================================================================

// BuildContainerPlan builds a ContainerPlan from a compose service and deployment data.
//
// This is a pure function that transforms compose service definitions and deployment
// parameters into a container plan that the shell can execute via Docker API.
//
// The function:
//   - Generates the container name using ContainerName()
//   - Copies image, command, and entrypoint from service
//   - Merges and substitutes environment variables
//   - Prefixes named volumes with deployment ID
//   - Parses health check durations
//   - Maps restart policy to Docker format
//   - Copies and merges labels
//
// Example:
//
//	params := BuildContainerPlanParams{
//	    DeploymentID: "abc123",
//	    TemplateID:   "tmpl-456",
//	    ServiceName:  "web",
//	    Service:      compose.Service{Name: "web", Image: "nginx:latest"},
//	    Variables:    map[string]string{"PORT": "8080"},
//	    NetworkName:  "hoster_abc123",
//	    Volumes:      []compose.Volume{},
//	}
//	plan := BuildContainerPlan(params)
func BuildContainerPlan(params BuildContainerPlanParams) ContainerPlan {
	svc := params.Service

	plan := ContainerPlan{
		Name:       ContainerName(params.DeploymentID, params.ServiceName),
		Image:      svc.Image,
		Command:    svc.Command,
		Entrypoint: svc.Entrypoint,
		Env:        make(map[string]string),
		Labels: map[string]string{
			LabelManaged:    "true",
			LabelDeployment: params.DeploymentID,
			LabelTemplate:   params.TemplateID,
			LabelService:    params.ServiceName,
		},
		Networks: []string{params.NetworkName},
	}

	// Merge environment: service env + deployment variables
	for k, v := range svc.Environment {
		plan.Env[k] = SubstituteVariables(v, params.Variables)
	}

	// Port bindings
	for _, p := range svc.Ports {
		plan.Ports = append(plan.Ports, PortPlan{
			ContainerPort: int(p.Target),
			HostPort:      int(p.Published),
			Protocol:      p.Protocol,
			HostIP:        p.HostIP,
		})
	}

	// Volume mounts
	for _, v := range svc.Volumes {
		source := v.Source
		// Replace named volume with deployment-prefixed name
		if v.Type == compose.VolumeMountTypeVolume {
			source = VolumeName(params.DeploymentID, v.Source)
		}
		plan.Volumes = append(plan.Volumes, VolumePlan{
			Source:   source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}

	// Health check
	if svc.HealthCheck != nil {
		plan.HealthCheck = &HealthCheckPlan{
			Test:    svc.HealthCheck.Test,
			Retries: svc.HealthCheck.Retries,
		}
		if svc.HealthCheck.Interval != "" {
			if d, err := time.ParseDuration(svc.HealthCheck.Interval); err == nil {
				plan.HealthCheck.Interval = d
			}
		}
		if svc.HealthCheck.Timeout != "" {
			if d, err := time.ParseDuration(svc.HealthCheck.Timeout); err == nil {
				plan.HealthCheck.Timeout = d
			}
		}
		if svc.HealthCheck.StartPeriod != "" {
			if d, err := time.ParseDuration(svc.HealthCheck.StartPeriod); err == nil {
				plan.HealthCheck.StartPeriod = d
			}
		}
	}

	// Resource limits
	if svc.Resources.CPULimit > 0 {
		plan.Resources.CPULimit = svc.Resources.CPULimit
	}
	if svc.Resources.MemoryLimit > 0 {
		plan.Resources.MemoryLimit = svc.Resources.MemoryLimit
	}

	// Restart policy
	plan.RestartPolicy = mapRestartPolicy(svc.Restart)

	// Copy service labels
	for k, v := range svc.Labels {
		plan.Labels[k] = v
	}

	return plan
}

// mapRestartPolicy maps compose restart policy to Docker restart policy name.
func mapRestartPolicy(policy compose.RestartPolicy) RestartPolicyPlan {
	switch policy {
	case compose.RestartAlways:
		return RestartPolicyPlan{Name: "always"}
	case compose.RestartOnFailure:
		return RestartPolicyPlan{Name: "on-failure"}
	case compose.RestartUnlessStopped:
		return RestartPolicyPlan{Name: "unless-stopped"}
	default:
		return RestartPolicyPlan{Name: "no"}
	}
}
