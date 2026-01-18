// Package deployment provides pure functions for deployment planning.
//
// This package contains the functional core logic for transforming compose
// specifications into Docker execution plans. All functions are pure
// (no I/O, no side effects) and comply with ADR-002 "Values as Boundaries".
//
// # Functions
//
//   - Naming: Generate consistent resource names (NetworkName, VolumeName, ContainerName)
//   - Ordering: Sort services by dependencies (TopologicalSort)
//   - Variables: Substitute environment variable placeholders (SubstituteVariables)
//   - Ports: Convert port bindings to domain types (ConvertPorts)
//   - Container: Build container plans from compose services (BuildContainerPlan)
//
// # Usage
//
// The imperative shell (internal/shell/docker) uses these pure functions
// to plan deployments, then executes the plans via the Docker API.
//
//	networkName := deployment.NetworkName(deploymentID)
//	orderedServices := deployment.TopologicalSort(services)
//	plan := deployment.BuildContainerPlan(params)
package deployment
