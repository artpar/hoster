package compose

// =============================================================================
// ParsedSpec - Main Output Type
// =============================================================================

// ParsedSpec represents a fully parsed Docker Compose specification.
// This is the Hoster-specific representation, decoupled from compose-go types.
type ParsedSpec struct {
	Services []Service `json:"services"`
	Networks []Network `json:"networks,omitempty"`
	Volumes  []Volume  `json:"volumes,omitempty"`
}

// =============================================================================
// Service Types
// =============================================================================

// Service represents a single service definition.
type Service struct {
	Name        string            `json:"name"`
	Image       string            `json:"image,omitempty"`
	Build       *BuildConfig      `json:"build,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty"`
	Ports       []Port            `json:"ports,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Volumes     []VolumeMount     `json:"volumes,omitempty"`
	Networks    []string          `json:"networks,omitempty"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Restart     RestartPolicy     `json:"restart,omitempty"`
	Resources   ServiceResources  `json:"resources"`
	HealthCheck *HealthCheck      `json:"healthcheck,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// BuildConfig represents build configuration (optional).
type BuildConfig struct {
	Context    string `json:"context"`
	Dockerfile string `json:"dockerfile,omitempty"`
}

// Port represents a port mapping.
type Port struct {
	Target    uint32 `json:"target"`              // Container port
	Published uint32 `json:"published,omitempty"` // Host port (0 = dynamic)
	Protocol  string `json:"protocol,omitempty"`  // tcp, udp
	HostIP    string `json:"host_ip,omitempty"`   // Bind IP
}

// VolumeMount represents a volume mount in a service.
type VolumeMount struct {
	Type     VolumeMountType `json:"type"`     // bind, volume, tmpfs
	Source   string          `json:"source"`   // Path or volume name
	Target   string          `json:"target"`   // Container path
	ReadOnly bool            `json:"readonly"`
}

// VolumeMountType represents the type of volume mount.
type VolumeMountType string

const (
	VolumeMountTypeBind   VolumeMountType = "bind"
	VolumeMountTypeVolume VolumeMountType = "volume"
	VolumeMountTypeTmpfs  VolumeMountType = "tmpfs"
)

// ServiceResources represents resource limits/reservations for a service.
type ServiceResources struct {
	CPULimit          float64 `json:"cpu_limit"`
	CPUReservation    float64 `json:"cpu_reservation"`
	MemoryLimit       int64   `json:"memory_limit"`       // Bytes
	MemoryReservation int64   `json:"memory_reservation"` // Bytes
}

// RestartPolicy represents the restart policy.
type RestartPolicy string

const (
	RestartNo            RestartPolicy = "no"
	RestartAlways        RestartPolicy = "always"
	RestartOnFailure     RestartPolicy = "on-failure"
	RestartUnlessStopped RestartPolicy = "unless-stopped"
)

// HealthCheck represents health check configuration.
type HealthCheck struct {
	Test        []string `json:"test"`
	Interval    string   `json:"interval,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod string   `json:"start_period,omitempty"`
}

// =============================================================================
// Network Types
// =============================================================================

// Network represents a network definition.
type Network struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver,omitempty"`
	External   bool              `json:"external"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	Labels     map[string]string `json:"labels,omitempty"`
	IPAM       *IPAM             `json:"ipam,omitempty"`
}

// IPAM represents IP address management configuration.
type IPAM struct {
	Driver string       `json:"driver,omitempty"`
	Config []IPAMConfig `json:"config,omitempty"`
}

// IPAMConfig represents IPAM configuration.
type IPAMConfig struct {
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

// =============================================================================
// Volume Types
// =============================================================================

// Volume represents a named volume definition.
type Volume struct {
	Name     string            `json:"name"`
	Driver   string            `json:"driver,omitempty"`
	External bool              `json:"external"`
	Labels   map[string]string `json:"labels,omitempty"`
}
