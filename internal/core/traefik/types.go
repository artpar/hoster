package traefik

// =============================================================================
// Traefik Label Generation Types
// =============================================================================

// LabelParams contains parameters for generating Traefik labels.
type LabelParams struct {
	// DeploymentID is the unique deployment identifier.
	DeploymentID string

	// ServiceName is the name of the service (e.g., "web", "api").
	ServiceName string

	// Hostname is the domain/hostname for routing (e.g., "myapp.apps.hoster.io").
	Hostname string

	// Port is the container port to route traffic to.
	Port int

	// EnableTLS enables HTTPS routing with TLS termination.
	EnableTLS bool
}
