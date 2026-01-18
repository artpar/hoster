package docker

import (
	"errors"
	"fmt"
)

// =============================================================================
// Error Types
// =============================================================================

var (
	// Container errors
	ErrContainerNotFound       = errors.New("container not found")
	ErrContainerAlreadyExists  = errors.New("container already exists")
	ErrContainerNotRunning     = errors.New("container is not running")
	ErrContainerAlreadyRunning = errors.New("container is already running")

	// Network errors
	ErrNetworkNotFound      = errors.New("network not found")
	ErrNetworkAlreadyExists = errors.New("network already exists")
	ErrNetworkInUse         = errors.New("network has active endpoints")

	// Volume errors
	ErrVolumeNotFound = errors.New("volume not found")
	ErrVolumeInUse    = errors.New("volume is in use")

	// Image errors
	ErrImageNotFound   = errors.New("image not found")
	ErrImagePullFailed = errors.New("image pull failed")

	// Connection errors
	ErrPortAlreadyAllocated = errors.New("port is already allocated")
	ErrConnectionFailed     = errors.New("docker connection failed")
	ErrTimeout              = errors.New("operation timed out")
)

// DockerError wraps errors with additional context.
type DockerError struct {
	Op      string // Operation that failed
	Entity  string // Entity type (container, network, volume, image)
	ID      string // Entity ID if applicable
	Message string
	Err     error
}

func (e *DockerError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s %s %s: %s", e.Op, e.Entity, e.ID, e.Message)
	}
	if e.Entity != "" {
		return fmt.Sprintf("%s %s: %s", e.Op, e.Entity, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *DockerError) Unwrap() error {
	return e.Err
}

// NewDockerError creates a new DockerError.
func NewDockerError(op, entity, id, message string, err error) *DockerError {
	return &DockerError{
		Op:      op,
		Entity:  entity,
		ID:      id,
		Message: message,
		Err:     err,
	}
}
