package deployment

import (
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// BuildContainerPlan Tests
// =============================================================================

func TestBuildContainerPlan_BasicService(t *testing.T) {
	service := compose.Service{
		Name:  "web",
		Image: "nginx:latest",
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "web",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{},
	}

	plan := BuildContainerPlan(params)

	assert.Equal(t, "hoster_deploy-123_web", plan.Name)
	assert.Equal(t, "nginx:latest", plan.Image)
	assert.Contains(t, plan.Networks, "hoster_deploy-123")
	assert.Equal(t, "true", plan.Labels[LabelManaged])
	assert.Equal(t, "deploy-123", plan.Labels[LabelDeployment])
	assert.Equal(t, "tmpl-456", plan.Labels[LabelTemplate])
	assert.Equal(t, "web", plan.Labels[LabelService])
}

func TestBuildContainerPlan_WithEnvironment(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
		Environment: map[string]string{
			"DB_HOST": "${DB_HOST}",
			"PORT":    "3000",
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{"DB_HOST": "localhost"},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{},
	}

	plan := BuildContainerPlan(params)

	assert.Equal(t, "localhost", plan.Env["DB_HOST"])
	assert.Equal(t, "3000", plan.Env["PORT"])
}

func TestBuildContainerPlan_WithVolumes(t *testing.T) {
	service := compose.Service{
		Name:  "db",
		Image: "postgres:15",
		Volumes: []compose.VolumeMount{
			{Type: compose.VolumeMountTypeVolume, Source: "pgdata", Target: "/var/lib/postgresql/data"},
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "db",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{{Name: "pgdata"}},
	}

	plan := BuildContainerPlan(params)

	require.Len(t, plan.Volumes, 1)
	assert.Equal(t, "hoster_deploy-123_pgdata", plan.Volumes[0].Source)
	assert.Equal(t, "/var/lib/postgresql/data", plan.Volumes[0].Target)
}

func TestBuildContainerPlan_WithBindMount(t *testing.T) {
	service := compose.Service{
		Name:  "web",
		Image: "nginx:latest",
		Volumes: []compose.VolumeMount{
			{Type: compose.VolumeMountTypeBind, Source: "./config", Target: "/etc/nginx/conf.d"},
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "web",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{},
	}

	plan := BuildContainerPlan(params)

	require.Len(t, plan.Volumes, 1)
	// Bind mounts should NOT be prefixed
	assert.Equal(t, "./config", plan.Volumes[0].Source)
	assert.Equal(t, "/etc/nginx/conf.d", plan.Volumes[0].Target)
}

func TestBuildContainerPlan_WithHealthCheck(t *testing.T) {
	service := compose.Service{
		Name:  "web",
		Image: "nginx:latest",
		HealthCheck: &compose.HealthCheck{
			Test:        []string{"CMD", "curl", "-f", "http://localhost"},
			Interval:    "30s",
			Timeout:     "10s",
			Retries:     3,
			StartPeriod: "5s",
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "web",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{},
	}

	plan := BuildContainerPlan(params)

	require.NotNil(t, plan.HealthCheck)
	assert.Equal(t, []string{"CMD", "curl", "-f", "http://localhost"}, plan.HealthCheck.Test)
	assert.Equal(t, 30*time.Second, plan.HealthCheck.Interval)
	assert.Equal(t, 10*time.Second, plan.HealthCheck.Timeout)
	assert.Equal(t, 3, plan.HealthCheck.Retries)
	assert.Equal(t, 5*time.Second, plan.HealthCheck.StartPeriod)
}

func TestBuildContainerPlan_NoHealthCheck(t *testing.T) {
	service := compose.Service{
		Name:  "web",
		Image: "nginx:latest",
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "web",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	assert.Nil(t, plan.HealthCheck)
}

func TestBuildContainerPlan_RestartPolicies(t *testing.T) {
	tests := []struct {
		name           string
		composeRestart compose.RestartPolicy
		expectedName   string
	}{
		{"always", compose.RestartAlways, "always"},
		{"on-failure", compose.RestartOnFailure, "on-failure"},
		{"unless-stopped", compose.RestartUnlessStopped, "unless-stopped"},
		{"no", compose.RestartNo, "no"},
		{"empty", "", "no"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := compose.Service{
				Name:    "app",
				Image:   "nginx",
				Restart: tt.composeRestart,
			}
			params := BuildContainerPlanParams{
				DeploymentID: "deploy-123",
				TemplateID:   "tmpl-456",
				ServiceName:  "app",
				Service:      service,
				Variables:    map[string]string{},
				NetworkName:  "hoster_deploy-123",
			}

			plan := BuildContainerPlan(params)
			assert.Equal(t, tt.expectedName, plan.RestartPolicy.Name)
		})
	}
}

func TestBuildContainerPlan_WithResources(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
		Resources: compose.ServiceResources{
			CPULimit:    2.0,
			MemoryLimit: 536870912, // 512MB
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	assert.Equal(t, 2.0, plan.Resources.CPULimit)
	assert.Equal(t, int64(536870912), plan.Resources.MemoryLimit)
}

func TestBuildContainerPlan_NoResources(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	assert.Equal(t, float64(0), plan.Resources.CPULimit)
	assert.Equal(t, int64(0), plan.Resources.MemoryLimit)
}

func TestBuildContainerPlan_WithPorts(t *testing.T) {
	service := compose.Service{
		Name:  "web",
		Image: "nginx:latest",
		Ports: []compose.Port{
			{Target: 80, Published: 8080, Protocol: "tcp"},
			{Target: 443, Published: 8443, Protocol: "tcp"},
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "web",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	require.Len(t, plan.Ports, 2)
	assert.Equal(t, 80, plan.Ports[0].ContainerPort)
	assert.Equal(t, 8080, plan.Ports[0].HostPort)
	assert.Equal(t, "tcp", plan.Ports[0].Protocol)
	assert.Equal(t, 443, plan.Ports[1].ContainerPort)
	assert.Equal(t, 8443, plan.Ports[1].HostPort)
}

func TestBuildContainerPlan_Labels(t *testing.T) {
	service := compose.Service{
		Name:  "web",
		Image: "nginx:latest",
		Labels: map[string]string{
			"custom.label":  "value",
			"another.label": "another-value",
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "web",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	// Hoster labels
	assert.Equal(t, "true", plan.Labels[LabelManaged])
	assert.Equal(t, "deploy-123", plan.Labels[LabelDeployment])
	assert.Equal(t, "tmpl-456", plan.Labels[LabelTemplate])
	assert.Equal(t, "web", plan.Labels[LabelService])
	// Custom labels
	assert.Equal(t, "value", plan.Labels["custom.label"])
	assert.Equal(t, "another-value", plan.Labels["another.label"])
}

func TestBuildContainerPlan_CommandAndEntrypoint(t *testing.T) {
	service := compose.Service{
		Name:       "app",
		Image:      "myapp:1.0",
		Command:    []string{"npm", "start"},
		Entrypoint: []string{"/docker-entrypoint.sh"},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	assert.Equal(t, []string{"npm", "start"}, plan.Command)
	assert.Equal(t, []string{"/docker-entrypoint.sh"}, plan.Entrypoint)
}

func TestBuildContainerPlan_EnvironmentSubstitution(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
		Environment: map[string]string{
			"DATABASE_URL": "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT:-5432}/${DB_NAME}",
			"DEBUG":        "${DEBUG:-false}",
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables: map[string]string{
			"DB_USER": "admin",
			"DB_PASS": "secret",
			"DB_HOST": "localhost",
			"DB_NAME": "mydb",
		},
		NetworkName: "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	assert.Equal(t, "postgres://admin:secret@localhost:5432/mydb", plan.Env["DATABASE_URL"])
	assert.Equal(t, "false", plan.Env["DEBUG"])
}

func TestBuildContainerPlan_ReadOnlyVolume(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
		Volumes: []compose.VolumeMount{
			{Type: compose.VolumeMountTypeVolume, Source: "config", Target: "/app/config", ReadOnly: true},
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{{Name: "config"}},
	}

	plan := BuildContainerPlan(params)

	require.Len(t, plan.Volumes, 1)
	assert.True(t, plan.Volumes[0].ReadOnly)
}

func TestBuildContainerPlan_MultipleVolumes(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
		Volumes: []compose.VolumeMount{
			{Type: compose.VolumeMountTypeVolume, Source: "data", Target: "/app/data"},
			{Type: compose.VolumeMountTypeVolume, Source: "logs", Target: "/app/logs"},
			{Type: compose.VolumeMountTypeBind, Source: "./config", Target: "/app/config"},
		},
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
		Volumes:      []compose.Volume{{Name: "data"}, {Name: "logs"}},
	}

	plan := BuildContainerPlan(params)

	require.Len(t, plan.Volumes, 3)
	assert.Equal(t, "hoster_deploy-123_data", plan.Volumes[0].Source)
	assert.Equal(t, "hoster_deploy-123_logs", plan.Volumes[1].Source)
	assert.Equal(t, "./config", plan.Volumes[2].Source) // Bind mount not prefixed
}

func TestBuildContainerPlan_EmptyEnvironment(t *testing.T) {
	service := compose.Service{
		Name:  "app",
		Image: "myapp:1.0",
	}
	params := BuildContainerPlanParams{
		DeploymentID: "deploy-123",
		TemplateID:   "tmpl-456",
		ServiceName:  "app",
		Service:      service,
		Variables:    map[string]string{},
		NetworkName:  "hoster_deploy-123",
	}

	plan := BuildContainerPlan(params)

	assert.NotNil(t, plan.Env)
	assert.Empty(t, plan.Env)
}
