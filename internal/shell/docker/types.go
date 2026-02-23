// Package docker provides a Docker client for container lifecycle management.
package docker

import (
	"io"
	"time"
)

// =============================================================================
// Container Types
// =============================================================================

// ContainerSpec defines the specification for creating a container.
type ContainerSpec struct {
	Name          string
	Image         string
	Command       []string
	Entrypoint    []string
	Env           map[string]string
	Labels        map[string]string
	Ports         []PortBinding
	Volumes       []VolumeMount
	Networks        []string
	NetworkAliases  map[string][]string // network name → aliases (e.g., service name for DNS)
	WorkingDir      string
	User          string
	RestartPolicy RestartPolicy
	Resources     ResourceLimits
	HealthCheck   *HealthCheck
}

// PortBinding defines a port mapping.
type PortBinding struct {
	ContainerPort int
	HostPort      int    // 0 for auto-assign
	Protocol      string // "tcp" or "udp"
	HostIP        string // "" for 0.0.0.0
}

// VolumeMount defines a volume mount.
type VolumeMount struct {
	Source   string // Volume name or host path
	Target   string // Container path
	ReadOnly bool
}

// RestartPolicy defines the container restart policy.
type RestartPolicy struct {
	Name              string // "no", "always", "on-failure", "unless-stopped"
	MaximumRetryCount int
}

// ResourceLimits defines resource constraints.
type ResourceLimits struct {
	CPULimit    float64 // CPU cores
	MemoryLimit int64   // Bytes
}

// HealthCheck defines container health check configuration.
type HealthCheck struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int
	StartPeriod time.Duration
}

// =============================================================================
// Container Info
// =============================================================================

// ContainerStatus represents the container status.
type ContainerStatus string

const (
	ContainerStatusCreated    ContainerStatus = "created"
	ContainerStatusRunning    ContainerStatus = "running"
	ContainerStatusPaused     ContainerStatus = "paused"
	ContainerStatusRestarting ContainerStatus = "restarting"
	ContainerStatusRemoving   ContainerStatus = "removing"
	ContainerStatusExited     ContainerStatus = "exited"
	ContainerStatusDead       ContainerStatus = "dead"
)

// ContainerInfo contains information about a container.
type ContainerInfo struct {
	ID         string
	Name       string
	Image      string
	Status     ContainerStatus
	State      string // "running", "exited", "created", etc.
	Health     string // "healthy", "unhealthy", "starting", ""
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	Ports      []PortBinding
	Labels     map[string]string
	ExitCode   int
}

// =============================================================================
// Network Types
// =============================================================================

// NetworkSpec defines the specification for creating a network.
type NetworkSpec struct {
	Name   string
	Driver string // "bridge", "overlay", etc.
	Labels map[string]string
}

// =============================================================================
// Volume Types
// =============================================================================

// VolumeSpec defines the specification for creating a volume.
type VolumeSpec struct {
	Name   string
	Driver string
	Labels map[string]string
}

// =============================================================================
// Options
// =============================================================================

// RemoveOptions defines options for removing containers.
type RemoveOptions struct {
	Force         bool
	RemoveVolumes bool
}

// ListOptions defines options for listing containers.
type ListOptions struct {
	All     bool              // Include stopped containers
	Filters map[string]string // e.g., {"label": "com.hoster.deployment=xyz"}
}

// LogOptions defines options for container logs.
type LogOptions struct {
	Follow     bool
	Tail       string // "all" or number
	Since      time.Time
	Until      time.Time
	Timestamps bool
}

// PullOptions defines options for pulling images.
type PullOptions struct {
	Platform string // e.g., "linux/amd64"
}

// =============================================================================
// Client Interface
// =============================================================================

// Client defines the Docker client interface.
type Client interface {
	// Container operations
	CreateContainer(spec ContainerSpec) (containerID string, err error)
	StartContainer(containerID string) error
	StopContainer(containerID string, timeout *time.Duration) error
	RemoveContainer(containerID string, opts RemoveOptions) error
	InspectContainer(containerID string) (*ContainerInfo, error)
	ListContainers(opts ListOptions) ([]ContainerInfo, error)
	ContainerLogs(containerID string, opts LogOptions) (io.ReadCloser, error)
	ContainerStats(containerID string) (*ContainerResourceStats, error) // F010: Monitoring

	// Network operations
	CreateNetwork(spec NetworkSpec) (networkID string, err error)
	RemoveNetwork(networkID string) error
	ConnectNetwork(networkID, containerID string) error
	DisconnectNetwork(networkID, containerID string, force bool) error

	// Volume operations
	CreateVolume(spec VolumeSpec) (volumeName string, err error)
	RemoveVolume(volumeName string, force bool) error

	// Image operations
	PullImage(image string, opts PullOptions) error
	ImageExists(image string) (bool, error)

	// Health operations
	Ping() error
	Close() error
}

// ContainerResourceStats represents resource statistics for a container.
// Used by F010: Monitoring Dashboard
type ContainerResourceStats struct {
	CPUPercent       float64
	MemoryUsageBytes int64
	MemoryLimitBytes int64
	MemoryPercent    float64
	NetworkRxBytes   int64
	NetworkTxBytes   int64
	BlockReadBytes   int64
	BlockWriteBytes  int64
	PIDs             int
}

// =============================================================================
// Label Constants
// =============================================================================

const (
	LabelManaged    = "com.hoster.managed"
	LabelDeployment = "com.hoster.deployment"
	LabelTemplate   = "com.hoster.template"
	LabelService    = "com.hoster.service"
)
