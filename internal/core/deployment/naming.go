package deployment

import "fmt"

// =============================================================================
// Resource Naming Functions
// =============================================================================

// NetworkName generates a network name for a deployment.
// Pattern: hoster_{deploymentID}
//
// Example:
//
//	NetworkName("abc123") // returns "hoster_abc123"
func NetworkName(deploymentID string) string {
	return fmt.Sprintf("hoster_%s", deploymentID)
}

// VolumeName generates a volume name for a deployment.
// Pattern: hoster_{deploymentID}_{volumeName}
//
// Example:
//
//	VolumeName("abc123", "data") // returns "hoster_abc123_data"
func VolumeName(deploymentID, volumeName string) string {
	return fmt.Sprintf("hoster_%s_%s", deploymentID, volumeName)
}

// ContainerName generates a container name for a service in a deployment.
// Pattern: hoster_{deploymentID}_{serviceName}
//
// Example:
//
//	ContainerName("abc123", "web") // returns "hoster_abc123_web"
func ContainerName(deploymentID, serviceName string) string {
	return fmt.Sprintf("hoster_%s_%s", deploymentID, serviceName)
}
