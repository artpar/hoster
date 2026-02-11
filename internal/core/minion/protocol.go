// Package minion defines the protocol for communication between the hoster
// backend and the minion binary that runs on remote nodes.
//
// The minion binary is deployed to remote nodes and provides direct Docker
// SDK access. Communication happens via SSH exec with JSON input/output.
//
// This package contains pure types with no I/O - following ADR-002.
package minion

import (
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// Version Info
// =============================================================================

// Version is the current minion protocol version.
// Bump MAJOR for breaking changes, MINOR for new commands, PATCH for fixes.
const Version = "1.1.0"

// =============================================================================
// Response Envelope
// =============================================================================

// Response is the standard envelope for all minion command responses.
// All commands return this structure as JSON to stdout.
type Response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *ErrorInfo      `json:"error,omitempty"`
}

// ErrorInfo contains error details when Success is false.
type ErrorInfo struct {
	Command string `json:"command"`          // Command that failed
	Code    string `json:"code,omitempty"`   // Error code (e.g., "not_found")
	Message string `json:"message"`          // Human-readable error message
}

// NewSuccessResponse creates a successful response with data.
func NewSuccessResponse(data interface{}) (*Response, error) {
	var rawData json.RawMessage
	if data != nil {
		bytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("marshal data: %w", err)
		}
		rawData = bytes
	}
	return &Response{
		Success: true,
		Data:    rawData,
	}, nil
}

// NewErrorResponse creates an error response.
func NewErrorResponse(command, code, message string) *Response {
	return &Response{
		Success: false,
		Error: &ErrorInfo{
			Command: command,
			Code:    code,
			Message: message,
		},
	}
}

// ParseResponse parses a JSON response from the minion.
func ParseResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &resp, nil
}

// UnmarshalData unmarshals the response data into the target type.
func (r *Response) UnmarshalData(target interface{}) error {
	if r.Data == nil {
		return nil
	}
	return json.Unmarshal(r.Data, target)
}

// =============================================================================
// Error Codes
// =============================================================================

// Standard error codes for minion responses.
const (
	ErrCodeNotFound        = "not_found"
	ErrCodeAlreadyExists   = "already_exists"
	ErrCodeNotRunning      = "not_running"
	ErrCodeAlreadyRunning  = "already_running"
	ErrCodeInUse           = "in_use"
	ErrCodePortConflict    = "port_conflict"
	ErrCodeConnectionFailed = "connection_failed"
	ErrCodeTimeout         = "timeout"
	ErrCodePullFailed      = "pull_failed"
	ErrCodeInvalidInput    = "invalid_input"
	ErrCodeInternal        = "internal"
)

// =============================================================================
// Command Result Types
// =============================================================================

// VersionInfo is returned by the "version" command.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// PingInfo is returned by the "ping" command.
type PingInfo struct {
	DockerVersion string `json:"docker_version"`
	APIVersion    string `json:"api_version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
}

// SystemInfo is returned by the "system-info" command.
// It contains host-level resource metrics for the node.
type SystemInfo struct {
	CPUCores      float64 `json:"cpu_cores"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	DiskTotalMB   int64   `json:"disk_total_mb"`
	CPUUsedPct    float64 `json:"cpu_used_percent"`
	MemoryUsedMB  int64   `json:"memory_used_mb"`
	DiskUsedMB    int64   `json:"disk_used_mb"`
}

// CreateResult is returned when creating containers, networks, or volumes.
type CreateResult struct {
	ID string `json:"id"`
}

// VolumeCreateResult is returned when creating a volume.
type VolumeCreateResult struct {
	Name string `json:"name"`
}

// ImageExistsResult is returned by "image-exists" command.
type ImageExistsResult struct {
	Exists bool `json:"exists"`
}

// LogsResult is returned by "container-logs" command.
type LogsResult struct {
	Logs string `json:"logs"`
}

// =============================================================================
// Container Types (mirrors docker.ContainerSpec for JSON serialization)
// =============================================================================

// ContainerSpec defines the specification for creating a container.
// This mirrors internal/shell/docker.ContainerSpec but with JSON tags.
type ContainerSpec struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Command       []string          `json:"command,omitempty"`
	Entrypoint    []string          `json:"entrypoint,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Ports         []PortBinding     `json:"ports,omitempty"`
	Volumes       []VolumeMount     `json:"volumes,omitempty"`
	Networks      []string          `json:"networks,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	User          string            `json:"user,omitempty"`
	RestartPolicy RestartPolicy     `json:"restart_policy,omitempty"`
	Resources     ResourceLimits    `json:"resources,omitempty"`
	HealthCheck   *HealthCheck      `json:"health_check,omitempty"`
}

// PortBinding defines a port mapping.
type PortBinding struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port,omitempty"` // 0 for auto-assign
	Protocol      string `json:"protocol,omitempty"`  // "tcp" or "udp"
	HostIP        string `json:"host_ip,omitempty"`   // "" for 0.0.0.0
}

// VolumeMount defines a volume mount.
type VolumeMount struct {
	Source   string `json:"source"`              // Volume name or host path
	Target   string `json:"target"`              // Container path
	ReadOnly bool   `json:"read_only,omitempty"`
}

// RestartPolicy defines the container restart policy.
type RestartPolicy struct {
	Name              string `json:"name,omitempty"` // "no", "always", "on-failure", "unless-stopped"
	MaximumRetryCount int    `json:"maximum_retry_count,omitempty"`
}

// ResourceLimits defines resource constraints.
type ResourceLimits struct {
	CPULimit    float64 `json:"cpu_limit,omitempty"`    // CPU cores
	MemoryLimit int64   `json:"memory_limit,omitempty"` // Bytes
}

// HealthCheck defines container health check configuration.
type HealthCheck struct {
	Test        []string      `json:"test,omitempty"`
	Interval    time.Duration `json:"interval,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	Retries     int           `json:"retries,omitempty"`
	StartPeriod time.Duration `json:"start_period,omitempty"`
}

// ContainerInfo contains information about a container.
type ContainerInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Status     string            `json:"status"` // "created", "running", etc.
	State      string            `json:"state"`
	Health     string            `json:"health,omitempty"` // "healthy", "unhealthy", "starting", ""
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	FinishedAt *time.Time        `json:"finished_at,omitempty"`
	Ports      []PortBinding     `json:"ports,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	ExitCode   int               `json:"exit_code,omitempty"`
}

// ContainerResourceStats represents resource statistics for a container.
type ContainerResourceStats struct {
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

// =============================================================================
// Network and Volume Types
// =============================================================================

// NetworkSpec defines the specification for creating a network.
type NetworkSpec struct {
	Name   string            `json:"name"`
	Driver string            `json:"driver,omitempty"` // "bridge", "overlay", etc.
	Labels map[string]string `json:"labels,omitempty"`
}

// VolumeSpec defines the specification for creating a volume.
type VolumeSpec struct {
	Name   string            `json:"name"`
	Driver string            `json:"driver,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

// =============================================================================
// Options Types
// =============================================================================

// RemoveOptions defines options for removing containers.
type RemoveOptions struct {
	Force         bool `json:"force,omitempty"`
	RemoveVolumes bool `json:"remove_volumes,omitempty"`
}

// ListOptions defines options for listing containers.
type ListOptions struct {
	All     bool              `json:"all,omitempty"`
	Filters map[string]string `json:"filters,omitempty"`
}

// LogOptions defines options for container logs.
type LogOptions struct {
	Follow     bool      `json:"follow,omitempty"`
	Tail       string    `json:"tail,omitempty"` // "all" or number
	Since      time.Time `json:"since,omitempty"`
	Until      time.Time `json:"until,omitempty"`
	Timestamps bool      `json:"timestamps,omitempty"`
}

// PullOptions defines options for pulling images.
type PullOptions struct {
	Platform string `json:"platform,omitempty"` // e.g., "linux/amd64"
}
