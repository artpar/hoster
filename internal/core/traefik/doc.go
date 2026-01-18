// Package traefik provides pure functions for generating Traefik reverse proxy labels.
//
// This package contains the functional core logic for generating Docker container
// labels that configure Traefik routing. All functions are pure (no I/O, no side
// effects) and comply with ADR-002 "Values as Boundaries".
//
// # Functions
//
//   - GenerateLabels: Generate Traefik labels for HTTP/HTTPS routing
//
// # Usage
//
// The deployment planning stage uses these labels to enable external HTTP access:
//
//	labels := traefik.GenerateLabels(traefik.LabelParams{
//	    DeploymentID: deployment.ID,
//	    ServiceName:  service.Name,
//	    Hostname:     domain.Hostname,
//	    Port:         80,
//	    EnableTLS:    true,
//	})
//	for k, v := range labels {
//	    containerPlan.Labels[k] = v
//	}
package traefik
