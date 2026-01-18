// Package domain contains the core domain types and validation logic.
// This is part of the Functional Core - all functions are pure with no I/O.
package domain

import (
	"errors"
	"net"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Node Errors
// =============================================================================

var (
	// Node name validation errors
	ErrNodeNameRequired = errors.New("node name is required")
	ErrNodeNameTooShort = errors.New("node name must be at least 3 characters")
	ErrNodeNameTooLong  = errors.New("node name must be at most 100 characters")

	// SSH validation errors
	ErrSSHHostRequired = errors.New("SSH host is required")
	ErrSSHHostInvalid  = errors.New("SSH host must be a valid hostname or IP address")
	ErrSSHPortInvalid  = errors.New("SSH port must be between 1 and 65535")
	ErrSSHUserRequired = errors.New("SSH user is required")

	// Capabilities validation errors
	ErrCapabilitiesRequired = errors.New("at least one capability is required")
	ErrCapabilityEmpty      = errors.New("capability cannot be empty")

	// Node operation errors
	ErrNodeNotFound    = errors.New("node not found")
	ErrNodeOffline     = errors.New("node is offline")
	ErrNodeMaintenance = errors.New("node is in maintenance mode")
)

// =============================================================================
// Node Status
// =============================================================================

// NodeStatus represents the operational status of a node.
type NodeStatus string

const (
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

// IsValid checks if the node status is valid.
func (s NodeStatus) IsValid() bool {
	switch s {
	case NodeStatusOnline, NodeStatusOffline, NodeStatusMaintenance:
		return true
	default:
		return false
	}
}

// IsAvailable returns true if the node can accept deployments.
func (s NodeStatus) IsAvailable() bool {
	return s == NodeStatusOnline
}

// =============================================================================
// Node Capacity
// =============================================================================

// NodeCapacity represents the resource capacity and usage of a node.
type NodeCapacity struct {
	CPUCores     float64 `json:"cpu_cores"`
	MemoryMB     int64   `json:"memory_mb"`
	DiskMB       int64   `json:"disk_mb"`
	CPUUsed      float64 `json:"cpu_used"`
	MemoryUsedMB int64   `json:"memory_used_mb"`
	DiskUsedMB   int64   `json:"disk_used_mb"`
}

// AvailableCPU returns the available CPU cores.
func (c NodeCapacity) AvailableCPU() float64 {
	avail := c.CPUCores - c.CPUUsed
	if avail < 0 {
		return 0
	}
	return avail
}

// AvailableMemory returns the available memory in MB.
func (c NodeCapacity) AvailableMemory() int64 {
	avail := c.MemoryMB - c.MemoryUsedMB
	if avail < 0 {
		return 0
	}
	return avail
}

// AvailableDisk returns the available disk space in MB.
func (c NodeCapacity) AvailableDisk() int64 {
	avail := c.DiskMB - c.DiskUsedMB
	if avail < 0 {
		return 0
	}
	return avail
}

// CanHandle checks if the node can handle the given resource requirements.
func (c NodeCapacity) CanHandle(required Resources) bool {
	return c.AvailableCPU() >= required.CPUCores &&
		c.AvailableMemory() >= required.MemoryMB &&
		c.AvailableDisk() >= required.DiskMB
}

// UsagePercent returns the overall resource usage percentage (0-100).
func (c NodeCapacity) UsagePercent() float64 {
	if c.CPUCores == 0 && c.MemoryMB == 0 && c.DiskMB == 0 {
		return 0
	}

	cpuPercent := 0.0
	memPercent := 0.0
	diskPercent := 0.0

	if c.CPUCores > 0 {
		cpuPercent = (c.CPUUsed / c.CPUCores) * 100
	}
	if c.MemoryMB > 0 {
		memPercent = (float64(c.MemoryUsedMB) / float64(c.MemoryMB)) * 100
	}
	if c.DiskMB > 0 {
		diskPercent = (float64(c.DiskUsedMB) / float64(c.DiskMB)) * 100
	}

	// Weighted average: memory is most important for containers
	return cpuPercent*0.3 + memPercent*0.4 + diskPercent*0.3
}

// =============================================================================
// Node
// =============================================================================

// Node represents a worker node registered by a creator.
type Node struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	CreatorID       string       `json:"creator_id"`
	SSHHost         string       `json:"ssh_host"`
	SSHPort         int          `json:"ssh_port"`
	SSHUser         string       `json:"ssh_user"`
	SSHKeyID        string       `json:"ssh_key_id,omitempty"`
	DockerSocket    string       `json:"docker_socket"`
	Status          NodeStatus   `json:"status"`
	Capabilities    []string     `json:"capabilities"`
	Capacity        NodeCapacity `json:"capacity"`
	Location        string       `json:"location,omitempty"`
	LastHealthCheck *time.Time   `json:"last_health_check,omitempty"`
	ErrorMessage    string       `json:"error_message,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// GenerateNodeID generates a new node ID with "node_" prefix.
func GenerateNodeID() string {
	return "node_" + uuid.New().String()[:8]
}

// NewNode creates a new node with validated fields.
// Returns error if any validation fails.
func NewNode(creatorID, name, sshHost, sshUser string, sshPort int, capabilities []string) (*Node, error) {
	if err := ValidateNodeName(name); err != nil {
		return nil, err
	}
	if err := ValidateSSHHost(sshHost); err != nil {
		return nil, err
	}
	if err := ValidateSSHPort(sshPort); err != nil {
		return nil, err
	}
	if err := ValidateSSHUser(sshUser); err != nil {
		return nil, err
	}
	if err := ValidateCapabilities(capabilities); err != nil {
		return nil, err
	}

	if creatorID == "" {
		return nil, errors.New("creator ID is required")
	}

	now := time.Now()
	return &Node{
		ID:           GenerateNodeID(),
		Name:         name,
		CreatorID:    creatorID,
		SSHHost:      sshHost,
		SSHPort:      sshPort,
		SSHUser:      sshUser,
		DockerSocket: "/var/run/docker.sock",
		Status:       NodeStatusOffline,
		Capabilities: capabilities,
		Capacity:     NodeCapacity{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// HasCapability checks if the node has a specific capability.
func (n *Node) HasCapability(cap string) bool {
	for _, c := range n.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// HasAllCapabilities checks if the node has all the specified capabilities.
func (n *Node) HasAllCapabilities(caps []string) bool {
	for _, required := range caps {
		if !n.HasCapability(required) {
			return false
		}
	}
	return true
}

// HasAnyCapability checks if the node has any of the specified capabilities.
func (n *Node) HasAnyCapability(caps []string) bool {
	if len(caps) == 0 {
		return true
	}
	for _, cap := range caps {
		if n.HasCapability(cap) {
			return true
		}
	}
	return false
}

// IsAvailable returns true if the node can accept new deployments.
func (n *Node) IsAvailable() bool {
	return n.Status.IsAvailable()
}

// SSHAddress returns the SSH connection address (host:port).
func (n *Node) SSHAddress() string {
	return net.JoinHostPort(n.SSHHost, string(rune(n.SSHPort)))
}

// =============================================================================
// SSH Key
// =============================================================================

// SSHKey represents an encrypted SSH private key.
type SSHKey struct {
	ID                  string    `json:"id"`
	CreatorID           string    `json:"creator_id"`
	Name                string    `json:"name"`
	PrivateKeyEncrypted []byte    `json:"-"` // Never serialize
	Fingerprint         string    `json:"fingerprint"`
	CreatedAt           time.Time `json:"created_at"`
}

// GenerateSSHKeyID generates a new SSH key ID with "sshkey_" prefix.
func GenerateSSHKeyID() string {
	return "sshkey_" + uuid.New().String()[:8]
}

// =============================================================================
// Validation Functions
// =============================================================================

// ValidateNodeName validates a node name.
func ValidateNodeName(name string) error {
	if name == "" {
		return ErrNodeNameRequired
	}
	if len(name) < 3 {
		return ErrNodeNameTooShort
	}
	if len(name) > 100 {
		return ErrNodeNameTooLong
	}
	return nil
}

// ValidateSSHHost validates an SSH host (hostname or IP).
func ValidateSSHHost(host string) error {
	if host == "" {
		return ErrSSHHostRequired
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	// Check if it's a valid hostname
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if hostnameRegex.MatchString(host) {
		return nil
	}

	return ErrSSHHostInvalid
}

// ValidateSSHPort validates an SSH port.
func ValidateSSHPort(port int) error {
	if port < 1 || port > 65535 {
		return ErrSSHPortInvalid
	}
	return nil
}

// ValidateSSHUser validates an SSH username.
func ValidateSSHUser(user string) error {
	if user == "" {
		return ErrSSHUserRequired
	}
	return nil
}

// ValidateCapabilities validates node capabilities.
func ValidateCapabilities(caps []string) error {
	if len(caps) == 0 {
		return ErrCapabilitiesRequired
	}
	for _, cap := range caps {
		if cap == "" {
			return ErrCapabilityEmpty
		}
	}
	return nil
}

// =============================================================================
// Standard Capabilities
// =============================================================================

// StandardCapabilities are predefined capability tags.
var StandardCapabilities = []string{
	"standard",
	"gpu",
	"high-memory",
	"high-cpu",
	"ssd",
	"nvme",
}

// IsStandardCapability checks if a capability is one of the predefined ones.
func IsStandardCapability(cap string) bool {
	for _, std := range StandardCapabilities {
		if std == cap {
			return true
		}
	}
	return false
}

// DefaultCapabilities returns the default capabilities for a new node.
func DefaultCapabilities() []string {
	return []string{"standard"}
}
