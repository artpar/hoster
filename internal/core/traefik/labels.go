package traefik

import "fmt"

// =============================================================================
// Traefik Label Generation Functions
// =============================================================================

// GenerateLabels generates Traefik reverse proxy labels for a service.
//
// The generated labels configure Traefik to route HTTP(S) traffic to the container:
//   - Enables Traefik for the container
//   - Creates a router with Host rule for the specified hostname
//   - Configures the service loadbalancer port
//   - If TLS is enabled, creates an additional secure router
//
// Router and service names follow the pattern: {deploymentID}-{serviceName}
// This ensures uniqueness across all deployments.
//
// Example (HTTP only):
//
//	labels := GenerateLabels(LabelParams{
//	    DeploymentID: "abc123",
//	    ServiceName:  "web",
//	    Hostname:     "myapp.apps.hoster.io",
//	    Port:         80,
//	    EnableTLS:    false,
//	})
//	// Returns:
//	// {
//	//   "traefik.enable": "true",
//	//   "traefik.http.routers.abc123-web.rule": "Host(`myapp.apps.hoster.io`)",
//	//   "traefik.http.routers.abc123-web.entrypoints": "web",
//	//   "traefik.http.services.abc123-web.loadbalancer.server.port": "80",
//	// }
func GenerateLabels(params LabelParams) map[string]string {
	// Router/service name: {deploymentID}-{serviceName}
	name := fmt.Sprintf("%s-%s", params.DeploymentID, params.ServiceName)

	labels := map[string]string{
		// Enable Traefik for this container
		"traefik.enable": "true",

		// HTTP router
		fmt.Sprintf("traefik.http.routers.%s.rule", name):        fmt.Sprintf("Host(`%s`)", params.Hostname),
		fmt.Sprintf("traefik.http.routers.%s.entrypoints", name): "web",

		// Service (loadbalancer port)
		fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", name): fmt.Sprintf("%d", params.Port),
	}

	// Add HTTPS router if TLS is enabled
	if params.EnableTLS {
		secureName := name + "-secure"
		labels[fmt.Sprintf("traefik.http.routers.%s.rule", secureName)] = fmt.Sprintf("Host(`%s`)", params.Hostname)
		labels[fmt.Sprintf("traefik.http.routers.%s.entrypoints", secureName)] = "websecure"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls", secureName)] = "true"
		labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", secureName)] = "letsencrypt"
	}

	return labels
}
