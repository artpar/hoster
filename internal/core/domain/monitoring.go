// Package domain contains the core domain types for Hoster.
package domain

import "time"

// =============================================================================
// Health Types (F010: Monitoring Dashboard)
// =============================================================================

// HealthStatus represents the overall health of a deployment or container.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// DeploymentHealth represents the aggregated health of a deployment.
type DeploymentHealth struct {
	Status     HealthStatus      `json:"status"`
	Containers []ContainerHealth `json:"containers"`
	CheckedAt  time.Time         `json:"checked_at"`
}

// ContainerHealth represents the health status of a single container.
type ContainerHealth struct {
	Name      string       `json:"name"`
	Status    string       `json:"status"` // running, stopped, paused, restarting, exited
	Health    HealthStatus `json:"health"`
	StartedAt *time.Time   `json:"started_at,omitempty"`
	Restarts  int          `json:"restarts"`
}

// =============================================================================
// Stats Types
// =============================================================================

// ContainerStats represents resource usage statistics for a container.
type ContainerStats struct {
	Name             string  `json:"name"`
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`
	NetworkRxBytes   int64   `json:"network_rx_bytes"`
	NetworkTxBytes   int64   `json:"network_tx_bytes"`
	BlockReadBytes   int64   `json:"block_read_bytes"`
	BlockWriteBytes  int64   `json:"block_write_bytes"`
	PIDs             int     `json:"pids"`
}

// DeploymentStats represents aggregated stats for a deployment.
type DeploymentStats struct {
	Containers  []ContainerStats `json:"containers"`
	CollectedAt time.Time        `json:"collected_at"`
}

// =============================================================================
// Log Types
// =============================================================================

// ContainerLog represents a single log entry from a container.
type ContainerLog struct {
	Container string    `json:"container"`
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"` // stdout, stderr
	Message   string    `json:"message"`
}

// DeploymentLogs represents logs from a deployment.
type DeploymentLogs struct {
	Logs            []ContainerLog `json:"logs"`
	ContainerFilter *string        `json:"container_filter,omitempty"`
	Tail            int            `json:"tail"`
	Since           *time.Time     `json:"since,omitempty"`
}

// =============================================================================
// Event Types (Container Lifecycle)
// =============================================================================

// ContainerEventType represents the type of container lifecycle event.
type ContainerEventType string

const (
	EventImagePulling       ContainerEventType = "image_pulling"
	EventImagePulled        ContainerEventType = "image_pulled"
	EventContainerCreated   ContainerEventType = "container_created"
	EventContainerStarted   ContainerEventType = "container_started"
	EventContainerStopped   ContainerEventType = "container_stopped"
	EventContainerRestarted ContainerEventType = "container_restarted"
	EventContainerDied      ContainerEventType = "container_died"
	EventContainerOOM       ContainerEventType = "container_oom"
	EventHealthUnhealthy    ContainerEventType = "health_unhealthy"
	EventHealthHealthy      ContainerEventType = "health_healthy"
)

// ContainerEvent represents a container lifecycle event.
type ContainerEvent struct {
	ID           int                `json:"-"`
	ReferenceID  string             `json:"id"`
	DeploymentID int                `json:"-"`
	Type         ContainerEventType `json:"type"`
	Container    string             `json:"container"`
	Message      string             `json:"message"`
	Timestamp    time.Time          `json:"timestamp"`
	CreatedAt    time.Time          `json:"created_at"`
}

// NewContainerEvent creates a new container event.
func NewContainerEvent(referenceID string, deploymentID int, eventType ContainerEventType, container, message string) ContainerEvent {
	now := time.Now()
	return ContainerEvent{
		ReferenceID:  referenceID,
		DeploymentID: deploymentID,
		Type:         eventType,
		Container:    container,
		Message:      message,
		Timestamp:    now,
		CreatedAt:    now,
	}
}
