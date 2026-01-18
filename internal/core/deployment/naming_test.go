package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// NetworkName Tests
// =============================================================================

func TestNetworkName_Simple(t *testing.T) {
	got := NetworkName("abc123")
	assert.Equal(t, "hoster_abc123", got)
}

func TestNetworkName_UUID(t *testing.T) {
	got := NetworkName("550e8400-e29b-41d4-a716-446655440000")
	assert.Equal(t, "hoster_550e8400-e29b-41d4-a716-446655440000", got)
}

func TestNetworkName_Empty(t *testing.T) {
	got := NetworkName("")
	assert.Equal(t, "hoster_", got)
}

// =============================================================================
// VolumeName Tests
// =============================================================================

func TestVolumeName_Simple(t *testing.T) {
	got := VolumeName("abc123", "data")
	assert.Equal(t, "hoster_abc123_data", got)
}

func TestVolumeName_WithUnderscore(t *testing.T) {
	got := VolumeName("abc123", "postgres_data")
	assert.Equal(t, "hoster_abc123_postgres_data", got)
}

func TestVolumeName_EmptyDeploymentID(t *testing.T) {
	got := VolumeName("", "data")
	assert.Equal(t, "hoster__data", got)
}

func TestVolumeName_EmptyVolumeName(t *testing.T) {
	got := VolumeName("abc123", "")
	assert.Equal(t, "hoster_abc123_", got)
}

// =============================================================================
// ContainerName Tests
// =============================================================================

func TestContainerName_Simple(t *testing.T) {
	got := ContainerName("abc123", "web")
	assert.Equal(t, "hoster_abc123_web", got)
}

func TestContainerName_DBService(t *testing.T) {
	got := ContainerName("abc123", "postgres")
	assert.Equal(t, "hoster_abc123_postgres", got)
}

func TestContainerName_WithHyphen(t *testing.T) {
	got := ContainerName("deploy-456", "my-service")
	assert.Equal(t, "hoster_deploy-456_my-service", got)
}

func TestContainerName_UUID(t *testing.T) {
	got := ContainerName("550e8400-e29b-41d4-a716-446655440000", "api")
	assert.Equal(t, "hoster_550e8400-e29b-41d4-a716-446655440000_api", got)
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestNetworkName_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		deploymentID string
		want         string
	}{
		{"simple", "abc123", "hoster_abc123"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000", "hoster_550e8400-e29b-41d4-a716-446655440000"},
		{"empty", "", "hoster_"},
		{"with-hyphen", "deploy-123", "hoster_deploy-123"},
		{"short-id", "a", "hoster_a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NetworkName(tt.deploymentID)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVolumeName_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		deploymentID string
		volumeName   string
		want         string
	}{
		{"simple", "abc123", "data", "hoster_abc123_data"},
		{"with_underscore", "abc123", "postgres_data", "hoster_abc123_postgres_data"},
		{"with-hyphen", "deploy-123", "my-volume", "hoster_deploy-123_my-volume"},
		{"long-names", "very-long-deployment-id-here", "also-long-volume-name", "hoster_very-long-deployment-id-here_also-long-volume-name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VolumeName(tt.deploymentID, tt.volumeName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainerName_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		deploymentID string
		serviceName  string
		want         string
	}{
		{"simple", "abc123", "web", "hoster_abc123_web"},
		{"db_service", "abc123", "postgres", "hoster_abc123_postgres"},
		{"with-hyphen", "deploy-123", "my-service", "hoster_deploy-123_my-service"},
		{"redis", "abc123", "redis", "hoster_abc123_redis"},
		{"nginx", "abc123", "nginx", "hoster_abc123_nginx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainerName(tt.deploymentID, tt.serviceName)
			assert.Equal(t, tt.want, got)
		})
	}
}
