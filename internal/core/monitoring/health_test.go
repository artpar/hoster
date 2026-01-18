package monitoring

import (
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// AggregateHealth Tests
// =============================================================================

func TestAggregateHealth_AllHealthy(t *testing.T) {
	containers := []domain.ContainerHealth{
		{Name: "web", Health: domain.HealthStatusHealthy},
		{Name: "db", Health: domain.HealthStatusHealthy},
	}

	result := AggregateHealth(containers)

	assert.Equal(t, domain.HealthStatusHealthy, result)
}

func TestAggregateHealth_OneUnhealthy(t *testing.T) {
	containers := []domain.ContainerHealth{
		{Name: "web", Health: domain.HealthStatusHealthy},
		{Name: "db", Health: domain.HealthStatusUnhealthy},
	}

	result := AggregateHealth(containers)

	assert.Equal(t, domain.HealthStatusDegraded, result)
}

func TestAggregateHealth_AllUnhealthy(t *testing.T) {
	containers := []domain.ContainerHealth{
		{Name: "web", Health: domain.HealthStatusUnhealthy},
		{Name: "db", Health: domain.HealthStatusUnhealthy},
	}

	result := AggregateHealth(containers)

	assert.Equal(t, domain.HealthStatusUnhealthy, result)
}

func TestAggregateHealth_MixedStatus(t *testing.T) {
	tests := []struct {
		name       string
		containers []domain.ContainerHealth
		expected   domain.HealthStatus
	}{
		{
			name: "one degraded",
			containers: []domain.ContainerHealth{
				{Name: "web", Health: domain.HealthStatusHealthy},
				{Name: "db", Health: domain.HealthStatusDegraded},
			},
			expected: domain.HealthStatusDegraded,
		},
		{
			name: "unhealthy and degraded",
			containers: []domain.ContainerHealth{
				{Name: "web", Health: domain.HealthStatusUnhealthy},
				{Name: "db", Health: domain.HealthStatusDegraded},
				{Name: "cache", Health: domain.HealthStatusHealthy},
			},
			expected: domain.HealthStatusDegraded,
		},
		{
			name: "one unknown",
			containers: []domain.ContainerHealth{
				{Name: "web", Health: domain.HealthStatusHealthy},
				{Name: "db", Health: domain.HealthStatusUnknown},
			},
			expected: domain.HealthStatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AggregateHealth(tt.containers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAggregateHealth_EmptyContainers(t *testing.T) {
	result := AggregateHealth([]domain.ContainerHealth{})

	assert.Equal(t, domain.HealthStatusUnknown, result)
}

func TestAggregateHealth_SingleContainer(t *testing.T) {
	tests := []struct {
		name     string
		health   domain.HealthStatus
		expected domain.HealthStatus
	}{
		{"healthy", domain.HealthStatusHealthy, domain.HealthStatusHealthy},
		{"unhealthy", domain.HealthStatusUnhealthy, domain.HealthStatusUnhealthy},
		{"degraded", domain.HealthStatusDegraded, domain.HealthStatusDegraded},
		{"unknown", domain.HealthStatusUnknown, domain.HealthStatusDegraded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containers := []domain.ContainerHealth{
				{Name: "app", Health: tt.health},
			}
			result := AggregateHealth(containers)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// DetermineContainerHealth Tests
// =============================================================================

func TestDetermineContainerHealth_Running(t *testing.T) {
	result := DetermineContainerHealth("running", nil, 0)

	assert.Equal(t, domain.HealthStatusHealthy, result)
}

func TestDetermineContainerHealth_Stopped(t *testing.T) {
	tests := []string{"stopped", "exited", "paused", "dead", "restarting"}

	for _, status := range tests {
		t.Run(status, func(t *testing.T) {
			result := DetermineContainerHealth(status, nil, 0)
			assert.Equal(t, domain.HealthStatusUnhealthy, result)
		})
	}
}

func TestDetermineContainerHealth_HighRestarts(t *testing.T) {
	tests := []struct {
		restarts int
		expected domain.HealthStatus
	}{
		{0, domain.HealthStatusHealthy},
		{1, domain.HealthStatusHealthy},
		{3, domain.HealthStatusHealthy},
		{4, domain.HealthStatusDegraded},
		{10, domain.HealthStatusDegraded},
	}

	for _, tt := range tests {
		t.Run("restarts="+string(rune('0'+tt.restarts)), func(t *testing.T) {
			result := DetermineContainerHealth("running", nil, tt.restarts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineContainerHealth_UnhealthyCheck(t *testing.T) {
	unhealthy := "unhealthy"
	result := DetermineContainerHealth("running", &unhealthy, 0)

	assert.Equal(t, domain.HealthStatusUnhealthy, result)
}

func TestDetermineContainerHealth_HealthyCheck(t *testing.T) {
	healthy := "healthy"
	result := DetermineContainerHealth("running", &healthy, 0)

	assert.Equal(t, domain.HealthStatusHealthy, result)
}

func TestDetermineContainerHealth_StartingCheck(t *testing.T) {
	starting := "starting"
	result := DetermineContainerHealth("running", &starting, 0)

	assert.Equal(t, domain.HealthStatusDegraded, result)
}

func TestDetermineContainerHealth_CombinedFactors(t *testing.T) {
	// Unhealthy check takes precedence over restarts
	unhealthy := "unhealthy"
	result := DetermineContainerHealth("running", &unhealthy, 10)
	assert.Equal(t, domain.HealthStatusUnhealthy, result)

	// Non-running status takes precedence over everything
	result = DetermineContainerHealth("stopped", &unhealthy, 10)
	assert.Equal(t, domain.HealthStatusUnhealthy, result)

	// High restarts still counted when healthy otherwise
	healthy := "healthy"
	result = DetermineContainerHealth("running", &healthy, 5)
	assert.Equal(t, domain.HealthStatusDegraded, result)
}

// =============================================================================
// ContainerEventMessage Tests
// =============================================================================

func TestContainerEventMessage(t *testing.T) {
	tests := []struct {
		eventType domain.ContainerEventType
		container string
		expected  string
	}{
		{domain.EventContainerCreated, "web", "Container web created"},
		{domain.EventContainerStarted, "db", "Container db started successfully"},
		{domain.EventContainerStopped, "cache", "Container cache stopped"},
		{domain.EventContainerRestarted, "app", "Container app restarted"},
		{domain.EventContainerDied, "worker", "Container worker died unexpectedly"},
		{domain.EventContainerOOM, "memhog", "Container memhog killed due to out of memory"},
		{domain.EventHealthUnhealthy, "api", "Container api health check failed"},
		{domain.EventHealthHealthy, "web", "Container web health check passed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			result := ContainerEventMessage(tt.eventType, tt.container)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainerEventMessage_UnknownType(t *testing.T) {
	result := ContainerEventMessage("unknown_event", "app")
	assert.Contains(t, result, "Container app")
	assert.Contains(t, result, "unknown_event")
}
