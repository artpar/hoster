package deployment

import (
	"github.com/artpar/hoster/internal/core/domain"
)

// =============================================================================
// Port Conversion Functions
// =============================================================================

// PortBinding represents a Docker port binding.
// This type mirrors the shell PortBinding type for conversion purposes.
type PortBinding struct {
	ContainerPort int
	HostPort      int
	Protocol      string
	HostIP        string
}

// ConvertPorts converts port bindings to domain port mappings.
// Default protocol is "tcp" if empty.
//
// Example:
//
//	ports := []PortBinding{{ContainerPort: 80, HostPort: 8080, Protocol: ""}}
//	mappings := ConvertPorts(ports)
//	// Result: []domain.PortMapping{{ContainerPort: 80, HostPort: 8080, Protocol: "tcp"}}
func ConvertPorts(ports []PortBinding) []domain.PortMapping {
	if len(ports) == 0 {
		return []domain.PortMapping{}
	}

	result := make([]domain.PortMapping, 0, len(ports))
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		result = append(result, domain.PortMapping{
			ContainerPort: p.ContainerPort,
			HostPort:      p.HostPort,
			Protocol:      proto,
		})
	}
	return result
}
