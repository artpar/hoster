// Package proxy provides pure types and functions for the App Proxy feature.
// This package has no I/O dependencies and is tested with values in/out.
package proxy

import "fmt"

// ProxyTarget represents the destination for a proxied request.
// This is a pure data type with no I/O.
type ProxyTarget struct {
	// DeploymentID is the deployment this target belongs to
	DeploymentID string

	// NodeID is the node where the container runs ("" or "local" for local node)
	NodeID string

	// Port is the host port the container is bound to
	Port int

	// Status is the deployment status (running, stopped, etc.)
	Status string

	// CustomerID is the owner of the deployment
	CustomerID string
}

// CanRoute returns true if the target can accept traffic.
// Only running deployments with a valid port can accept traffic.
func (t ProxyTarget) CanRoute() bool {
	return t.Status == "running" && t.Port > 0
}

// IsLocal returns true if the target is on the local node.
func (t ProxyTarget) IsLocal() bool {
	return t.NodeID == "" || t.NodeID == "local"
}

// LocalAddress returns the target address for local containers.
// For remote containers, use NodePool to get tunneled address.
func (t ProxyTarget) LocalAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", t.Port)
}
