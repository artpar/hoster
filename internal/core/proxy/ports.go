package proxy

import "errors"

// PortRange defines the available port range for deployments.
type PortRange struct {
	Start int // Inclusive, e.g., 30000
	End   int // Inclusive, e.g., 39999
}

// DefaultPortRange returns the default port range.
func DefaultPortRange() PortRange {
	return PortRange{Start: 30000, End: 39999}
}

// AllocatePort finds the first available port in the range.
// Pure function - takes used ports as input, returns allocated port.
func AllocatePort(usedPorts []int, portRange PortRange) (int, error) {
	// Build a set of used ports for O(1) lookup
	used := make(map[int]bool, len(usedPorts))
	for _, p := range usedPorts {
		used[p] = true
	}

	// Find first available port
	for port := portRange.Start; port <= portRange.End; port++ {
		if !used[port] {
			return port, nil
		}
	}

	return 0, errors.New("no available ports in range")
}

// ValidatePort checks if a port is within the allowed range.
func ValidatePort(port int, portRange PortRange) bool {
	return port >= portRange.Start && port <= portRange.End
}
