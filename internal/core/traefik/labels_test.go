package traefik

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// GenerateLabels Tests
// =============================================================================

func TestGenerateLabels_Basic(t *testing.T) {
	params := LabelParams{
		DeploymentID: "deploy-123",
		ServiceName:  "web",
		Hostname:     "myapp-abc123.apps.hoster.io",
		Port:         80,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	assert.Equal(t, "true", labels["traefik.enable"])
	assert.Equal(t, "Host(`myapp-abc123.apps.hoster.io`)", labels["traefik.http.routers.deploy-123-web.rule"])
	assert.Equal(t, "web", labels["traefik.http.routers.deploy-123-web.entrypoints"])
	assert.Equal(t, "80", labels["traefik.http.services.deploy-123-web.loadbalancer.server.port"])
}

func TestGenerateLabels_NoTLSLabels(t *testing.T) {
	params := LabelParams{
		DeploymentID: "deploy-123",
		ServiceName:  "web",
		Hostname:     "myapp.example.com",
		Port:         80,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	// Should NOT have TLS-related labels
	_, hasTLS := labels["traefik.http.routers.deploy-123-web-secure.rule"]
	assert.False(t, hasTLS)
	_, hasTLSEnabled := labels["traefik.http.routers.deploy-123-web-secure.tls"]
	assert.False(t, hasTLSEnabled)
}

func TestGenerateLabels_WithTLS(t *testing.T) {
	params := LabelParams{
		DeploymentID: "deploy-456",
		ServiceName:  "api",
		Hostname:     "api.example.com",
		Port:         3000,
		EnableTLS:    true,
	}

	labels := GenerateLabels(params)

	// Should have both HTTP and HTTPS routes
	// HTTP router
	assert.Equal(t, "true", labels["traefik.enable"])
	assert.Equal(t, "Host(`api.example.com`)", labels["traefik.http.routers.deploy-456-api.rule"])
	assert.Equal(t, "web", labels["traefik.http.routers.deploy-456-api.entrypoints"])

	// HTTPS router
	assert.Equal(t, "Host(`api.example.com`)", labels["traefik.http.routers.deploy-456-api-secure.rule"])
	assert.Equal(t, "websecure", labels["traefik.http.routers.deploy-456-api-secure.entrypoints"])
	assert.Equal(t, "true", labels["traefik.http.routers.deploy-456-api-secure.tls"])
	assert.Equal(t, "letsencrypt", labels["traefik.http.routers.deploy-456-api-secure.tls.certresolver"])

	// Service (shared by both routes)
	assert.Equal(t, "3000", labels["traefik.http.services.deploy-456-api.loadbalancer.server.port"])
}

func TestGenerateLabels_CustomPort(t *testing.T) {
	params := LabelParams{
		DeploymentID: "deploy-789",
		ServiceName:  "app",
		Hostname:     "app.example.com",
		Port:         8080,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	assert.Equal(t, "8080", labels["traefik.http.services.deploy-789-app.loadbalancer.server.port"])
}

func TestGenerateLabels_RouterNaming(t *testing.T) {
	params := LabelParams{
		DeploymentID: "abc123",
		ServiceName:  "web",
		Hostname:     "test.example.com",
		Port:         80,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	// Router name should be {deploymentID}-{serviceName}
	_, hasRouter := labels["traefik.http.routers.abc123-web.rule"]
	assert.True(t, hasRouter)
}

func TestGenerateLabels_ServiceNaming(t *testing.T) {
	params := LabelParams{
		DeploymentID: "def456",
		ServiceName:  "api",
		Hostname:     "test.example.com",
		Port:         3000,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	// Service name should be {deploymentID}-{serviceName}
	_, hasService := labels["traefik.http.services.def456-api.loadbalancer.server.port"]
	assert.True(t, hasService)
}

func TestGenerateLabels_SpecialCharactersInHostname(t *testing.T) {
	params := LabelParams{
		DeploymentID: "deploy-123",
		ServiceName:  "web",
		Hostname:     "my-app.subdomain.example.com",
		Port:         80,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	assert.Equal(t, "Host(`my-app.subdomain.example.com`)", labels["traefik.http.routers.deploy-123-web.rule"])
}

func TestGenerateLabels_HighPort(t *testing.T) {
	params := LabelParams{
		DeploymentID: "deploy-123",
		ServiceName:  "web",
		Hostname:     "test.example.com",
		Port:         65535,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	assert.Equal(t, "65535", labels["traefik.http.services.deploy-123-web.loadbalancer.server.port"])
}

func TestGenerateLabels_ZeroPort(t *testing.T) {
	// Edge case: port 0 should be passed through (validation at caller level)
	params := LabelParams{
		DeploymentID: "deploy-123",
		ServiceName:  "web",
		Hostname:     "test.example.com",
		Port:         0,
		EnableTLS:    false,
	}

	labels := GenerateLabels(params)

	assert.Equal(t, "0", labels["traefik.http.services.deploy-123-web.loadbalancer.server.port"])
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestGenerateLabels_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		params         LabelParams
		expectedLabels map[string]string
	}{
		{
			name: "basic http service",
			params: LabelParams{
				DeploymentID: "d1",
				ServiceName:  "web",
				Hostname:     "test.com",
				Port:         80,
				EnableTLS:    false,
			},
			expectedLabels: map[string]string{
				"traefik.enable":                                          "true",
				"traefik.http.routers.d1-web.rule":                        "Host(`test.com`)",
				"traefik.http.routers.d1-web.entrypoints":                 "web",
				"traefik.http.services.d1-web.loadbalancer.server.port":   "80",
			},
		},
		{
			name: "service with TLS",
			params: LabelParams{
				DeploymentID: "d2",
				ServiceName:  "api",
				Hostname:     "api.test.com",
				Port:         3000,
				EnableTLS:    true,
			},
			expectedLabels: map[string]string{
				"traefik.enable":                                            "true",
				"traefik.http.routers.d2-api.rule":                          "Host(`api.test.com`)",
				"traefik.http.routers.d2-api.entrypoints":                   "web",
				"traefik.http.routers.d2-api-secure.rule":                   "Host(`api.test.com`)",
				"traefik.http.routers.d2-api-secure.entrypoints":            "websecure",
				"traefik.http.routers.d2-api-secure.tls":                    "true",
				"traefik.http.routers.d2-api-secure.tls.certresolver":       "letsencrypt",
				"traefik.http.services.d2-api.loadbalancer.server.port":     "3000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := GenerateLabels(tt.params)

			for key, expectedValue := range tt.expectedLabels {
				assert.Equal(t, expectedValue, labels[key], "label %s", key)
			}
		})
	}
}

func TestGenerateLabels_LabelCount(t *testing.T) {
	// Without TLS: 4 labels
	paramsNoTLS := LabelParams{
		DeploymentID: "d1",
		ServiceName:  "web",
		Hostname:     "test.com",
		Port:         80,
		EnableTLS:    false,
	}
	labelsNoTLS := GenerateLabels(paramsNoTLS)
	assert.Len(t, labelsNoTLS, 4)

	// With TLS: 8 labels
	paramsWithTLS := LabelParams{
		DeploymentID: "d1",
		ServiceName:  "web",
		Hostname:     "test.com",
		Port:         80,
		EnableTLS:    true,
	}
	labelsWithTLS := GenerateLabels(paramsWithTLS)
	assert.Len(t, labelsWithTLS, 8)
}
