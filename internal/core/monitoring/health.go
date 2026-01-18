// Package monitoring provides pure functions for deployment monitoring logic.
// Following ADR-002: Values as Boundaries - this package contains NO I/O.
package monitoring

import "github.com/artpar/hoster/internal/core/domain"

// =============================================================================
// Health Aggregation (Pure Functions)
// =============================================================================

// AggregateHealth determines overall deployment health from container states.
// This is a pure function - it takes container health values and returns a status.
func AggregateHealth(containers []domain.ContainerHealth) domain.HealthStatus {
	if len(containers) == 0 {
		return domain.HealthStatusUnknown
	}

	unhealthy := 0
	degraded := 0

	for _, c := range containers {
		switch c.Health {
		case domain.HealthStatusUnhealthy:
			unhealthy++
		case domain.HealthStatusDegraded:
			degraded++
		case domain.HealthStatusUnknown:
			// Unknown containers count as degraded
			degraded++
		}
	}

	// All unhealthy = unhealthy
	if unhealthy == len(containers) {
		return domain.HealthStatusUnhealthy
	}
	// Any unhealthy or degraded = degraded
	if unhealthy > 0 || degraded > 0 {
		return domain.HealthStatusDegraded
	}
	// All healthy = healthy
	return domain.HealthStatusHealthy
}

// DetermineContainerHealth determines health from container state and metrics.
// This is a pure function that maps container state to health status.
//
// Parameters:
// - status: Container status (running, stopped, paused, restarting, exited)
// - healthCheck: Docker health check result if available (healthy, unhealthy, starting)
// - restarts: Number of restarts since container creation
func DetermineContainerHealth(status string, healthCheck *string, restarts int) domain.HealthStatus {
	// Non-running containers are unhealthy
	if status != "running" {
		return domain.HealthStatusUnhealthy
	}

	// If Docker health check reports unhealthy
	if healthCheck != nil && *healthCheck == "unhealthy" {
		return domain.HealthStatusUnhealthy
	}

	// Many restarts indicate instability
	if restarts > 3 {
		return domain.HealthStatusDegraded
	}

	// Health check still starting
	if healthCheck != nil && *healthCheck == "starting" {
		return domain.HealthStatusDegraded
	}

	return domain.HealthStatusHealthy
}

// =============================================================================
// Event Message Generation (Pure Functions)
// =============================================================================

// ContainerEventMessage generates a human-readable message for container events.
func ContainerEventMessage(eventType domain.ContainerEventType, containerName string) string {
	switch eventType {
	case domain.EventContainerCreated:
		return "Container " + containerName + " created"
	case domain.EventContainerStarted:
		return "Container " + containerName + " started successfully"
	case domain.EventContainerStopped:
		return "Container " + containerName + " stopped"
	case domain.EventContainerRestarted:
		return "Container " + containerName + " restarted"
	case domain.EventContainerDied:
		return "Container " + containerName + " died unexpectedly"
	case domain.EventContainerOOM:
		return "Container " + containerName + " killed due to out of memory"
	case domain.EventHealthUnhealthy:
		return "Container " + containerName + " health check failed"
	case domain.EventHealthHealthy:
		return "Container " + containerName + " health check passed"
	default:
		return "Container " + containerName + " event: " + string(eventType)
	}
}
