// Package docker provides a Docker client for container lifecycle management.
package docker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/compose"
	coredeployment "github.com/artpar/hoster/internal/core/deployment"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/traefik"
)

// =============================================================================
// Orchestrator - Manages Deployment Lifecycle
// =============================================================================

// Orchestrator manages the lifecycle of deployments using Docker.
type Orchestrator struct {
	docker    Client
	logger    *slog.Logger
	configDir string // Base directory for storing config files
}

// NewOrchestrator creates a new orchestrator.
// configDir is the base directory for storing deployment config files.
func NewOrchestrator(docker Client, logger *slog.Logger, configDir string) *Orchestrator {
	if logger == nil {
		logger = slog.Default()
	}
	if configDir == "" {
		configDir = "/var/lib/hoster/configs"
	}
	return &Orchestrator{
		docker:    docker,
		logger:    logger,
		configDir: configDir,
	}
}

// =============================================================================
// Start Deployment
// =============================================================================

// StartDeployment creates and starts all containers for a deployment.
// Returns the container info for all started containers.
// configFiles are written to disk and mounted into containers at their specified paths.
func (o *Orchestrator) StartDeployment(ctx context.Context, deployment *domain.Deployment, composeSpec string, configFiles []domain.ConfigFile) ([]domain.ContainerInfo, error) {
	o.logger.Info("starting deployment",
		"deployment_id", deployment.ID,
		"template_id", deployment.TemplateID,
		"config_files", len(configFiles),
	)

	// 1. Write config files to disk
	configMounts, err := o.writeConfigFiles(deployment.ID, configFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to write config files: %w", err)
	}

	// 2. Parse compose spec
	parsedSpec, err := compose.ParseComposeSpec(composeSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose spec: %w", err)
	}

	o.logger.Debug("parsed compose spec",
		"services", len(parsedSpec.Services),
		"networks", len(parsedSpec.Networks),
		"volumes", len(parsedSpec.Volumes),
	)

	// 2. Create network for deployment
	networkName := coredeployment.NetworkName(deployment.ID)
	networkID, err := o.createDeploymentNetwork(ctx, deployment.ID, networkName)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	o.logger.Debug("created network", "network_id", networkID, "network_name", networkName)

	// 3. Create named volumes
	for _, vol := range parsedSpec.Volumes {
		if vol.External {
			continue // Skip external volumes
		}
		volumeName := coredeployment.VolumeName(deployment.ID, vol.Name)
		if _, err := o.createDeploymentVolume(ctx, deployment.ID, volumeName); err != nil {
			// Cleanup network on failure
			_ = o.docker.RemoveNetwork(networkID)
			return nil, fmt.Errorf("failed to create volume %s: %w", vol.Name, err)
		}
		o.logger.Debug("created volume", "volume_name", volumeName)
	}

	// 4. Pull images
	for _, svc := range parsedSpec.Services {
		if svc.Image == "" {
			continue // Skip services with build (not supported yet)
		}
		exists, _ := o.docker.ImageExists(svc.Image)
		if !exists {
			o.logger.Info("pulling image", "image", svc.Image)
			if err := o.docker.PullImage(svc.Image, PullOptions{}); err != nil {
				o.logger.Warn("failed to pull image, trying anyway", "image", svc.Image, "error", err)
			}
		}
	}

	// 5. Check for existing containers (restart case)
	existingContainers, _ := o.docker.ListContainers(ListOptions{
		All: true,
		Filters: map[string]string{
			"label": fmt.Sprintf("%s=%s", LabelDeployment, deployment.ID),
		},
	})

	// 6. Create and start containers (respecting depends_on order)
	var containers []domain.ContainerInfo
	createdContainers := make(map[string]string) // serviceName -> containerID

	// Build map of existing containers by service name
	existingByService := make(map[string]ContainerInfo)
	for _, c := range existingContainers {
		if svc, ok := c.Labels[LabelService]; ok {
			existingByService[svc] = c
		}
	}

	orderedServices := coredeployment.TopologicalSort(parsedSpec.Services)

	for _, svc := range orderedServices {
		var containerID string
		var err error

		// Check if container already exists (restart case)
		if existing, found := existingByService[svc.Name]; found {
			containerID = existing.ID
			o.logger.Debug("using existing container", "service", svc.Name, "container_id", containerID[:12])
		} else {
			// Create new container
			containerName := coredeployment.ContainerName(deployment.ID, svc.Name)
			spec := o.buildContainerSpec(deployment, svc, containerName, networkName, parsedSpec.Volumes, configMounts)

			containerID, err = o.docker.CreateContainer(spec)
			if err != nil {
				// Cleanup on failure
				o.cleanupCreatedContainers(ctx, createdContainers)
				_ = o.docker.RemoveNetwork(networkID)
				return nil, fmt.Errorf("failed to create container %s: %w", svc.Name, err)
			}
			o.logger.Debug("created container", "service", svc.Name, "container_id", containerID[:12])
		}

		createdContainers[svc.Name] = containerID

		// Start the container (works for both new and existing stopped containers)
		if err := o.docker.StartContainer(containerID); err != nil {
			// Ignore error if already running
			if !strings.Contains(err.Error(), "already started") && !strings.Contains(err.Error(), "is already running") {
				o.cleanupCreatedContainers(ctx, createdContainers)
				_ = o.docker.RemoveNetwork(networkID)
				return nil, fmt.Errorf("failed to start container %s: %w", svc.Name, err)
			}
		}
		o.logger.Debug("started container", "service", svc.Name, "container_id", containerID[:12])

		// Get container info
		info, err := o.docker.InspectContainer(containerID)
		if err != nil {
			o.cleanupCreatedContainers(ctx, createdContainers)
			_ = o.docker.RemoveNetwork(networkID)
			return nil, fmt.Errorf("failed to inspect container %s: %w", svc.Name, err)
		}

		containers = append(containers, domain.ContainerInfo{
			ID:          info.ID,
			ServiceName: svc.Name,
			Image:       svc.Image,
			Status:      string(info.Status),
			Ports:       o.convertPorts(info.Ports),
		})
	}

	o.logger.Info("deployment started",
		"deployment_id", deployment.ID,
		"containers", len(containers),
	)

	return containers, nil
}

// =============================================================================
// Wait for Healthy
// =============================================================================

// WaitForHealthy polls containers until all are healthy or timeout.
// Checks every 5 seconds as per CLAUDE.md requirements.
func (o *Orchestrator) WaitForHealthy(ctx context.Context, deployment *domain.Deployment, timeout time.Duration) error {
	o.logger.Info("waiting for containers to be healthy",
		"deployment_id", deployment.ID,
		"timeout", timeout,
	)

	ticker := time.NewTicker(5 * time.Second) // Check every 5s per CLAUDE.md
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			allHealthy, err := o.checkAllContainersHealthy(deployment)
			if err != nil {
				return err
			}
			if allHealthy {
				o.logger.Info("all containers healthy", "deployment_id", deployment.ID)
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for containers to become healthy")
			}
			o.logger.Debug("containers not yet healthy, waiting...", "deployment_id", deployment.ID)
		}
	}
}

// checkAllContainersHealthy checks if all containers in deployment are healthy
func (o *Orchestrator) checkAllContainersHealthy(deployment *domain.Deployment) (bool, error) {
	for _, c := range deployment.Containers {
		info, err := o.docker.InspectContainer(c.ID)
		if err != nil {
			return false, fmt.Errorf("failed to inspect container %s: %w", c.ServiceName, err)
		}

		// If container has health check configured
		if info.Health != "" {
			if info.Health == "unhealthy" {
				return false, fmt.Errorf("container %s is unhealthy", c.ServiceName)
			}
			if info.Health != "healthy" {
				return false, nil // Still waiting
			}
		} else {
			// No health check - just check if running
			if info.Status != ContainerStatusRunning {
				return false, nil
			}
		}
	}
	return true, nil
}

// =============================================================================
// Stop Deployment
// =============================================================================

// StopDeployment stops all containers for a deployment.
func (o *Orchestrator) StopDeployment(ctx context.Context, deployment *domain.Deployment) error {
	o.logger.Info("stopping deployment", "deployment_id", deployment.ID)

	// List containers by label
	containers, err := o.docker.ListContainers(ListOptions{
		All: true,
		Filters: map[string]string{
			"label": fmt.Sprintf("%s=%s", LabelDeployment, deployment.ID),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Stop each container
	timeout := 10 * time.Second
	for _, c := range containers {
		if c.Status == ContainerStatusRunning {
			o.logger.Debug("stopping container", "container_id", c.ID[:12], "name", c.Name)
			if err := o.docker.StopContainer(c.ID, &timeout); err != nil {
				o.logger.Warn("failed to stop container", "container_id", c.ID[:12], "error", err)
				// Continue stopping others
			}
		}
	}

	o.logger.Info("deployment stopped", "deployment_id", deployment.ID, "containers_stopped", len(containers))
	return nil
}

// =============================================================================
// Remove Deployment
// =============================================================================

// RemoveDeployment removes all resources for a deployment.
// Order: containers → network → volumes
func (o *Orchestrator) RemoveDeployment(ctx context.Context, deployment *domain.Deployment) error {
	o.logger.Info("removing deployment", "deployment_id", deployment.ID)

	// 1. List and remove containers
	containers, err := o.docker.ListContainers(ListOptions{
		All: true,
		Filters: map[string]string{
			"label": fmt.Sprintf("%s=%s", LabelDeployment, deployment.ID),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	timeout := 10 * time.Second
	for _, c := range containers {
		// Stop if running
		if c.Status == ContainerStatusRunning {
			_ = o.docker.StopContainer(c.ID, &timeout)
		}
		// Remove container
		if err := o.docker.RemoveContainer(c.ID, RemoveOptions{Force: true, RemoveVolumes: false}); err != nil {
			o.logger.Warn("failed to remove container", "container_id", c.ID[:12], "error", err)
		} else {
			o.logger.Debug("removed container", "container_id", c.ID[:12])
		}
	}

	// 2. Remove network
	networkName := coredeployment.NetworkName(deployment.ID)
	if err := o.docker.RemoveNetwork(networkName); err != nil {
		o.logger.Warn("failed to remove network", "network", networkName, "error", err)
	} else {
		o.logger.Debug("removed network", "network", networkName)
	}

	// 3. Remove volumes (prefixed with deployment ID)
	// Note: We'd need to list volumes by label, but Docker's volume list API is limited
	// For now, we track volumes in deployment.Variables or skip automatic cleanup
	o.logger.Debug("volume cleanup skipped - requires explicit volume list")

	o.logger.Info("deployment removed", "deployment_id", deployment.ID)
	return nil
}

// =============================================================================
// Get Container Logs
// =============================================================================

// GetContainerLogs returns logs for a specific container.
func (o *Orchestrator) GetContainerLogs(ctx context.Context, containerID string, tail string) (string, error) {
	reader, err := o.docker.ContainerLogs(containerID, LogOptions{
		Tail:       tail,
		Timestamps: true,
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buf := make([]byte, 64*1024) // 64KB buffer
	n, _ := reader.Read(buf)
	return string(buf[:n]), nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// createDeploymentNetwork creates a network for a deployment or returns existing one.
func (o *Orchestrator) createDeploymentNetwork(ctx context.Context, deploymentID, networkName string) (string, error) {
	// Try to create the network
	networkID, err := o.docker.CreateNetwork(NetworkSpec{
		Name:   networkName,
		Driver: "bridge",
		Labels: map[string]string{
			LabelManaged:    "true",
			LabelDeployment: deploymentID,
		},
	})
	if err != nil {
		// Check if it's a "network already exists" error - use existing network
		if strings.Contains(err.Error(), "already exists") {
			o.logger.Debug("network already exists, reusing", "network_name", networkName)
			// Return the network name as ID (Docker accepts name or ID)
			return networkName, nil
		}
		return "", err
	}
	return networkID, nil
}

// createDeploymentVolume creates a volume for a deployment or returns existing one.
func (o *Orchestrator) createDeploymentVolume(ctx context.Context, deploymentID, volumeName string) (string, error) {
	volID, err := o.docker.CreateVolume(VolumeSpec{
		Name: volumeName,
		Labels: map[string]string{
			LabelManaged:    "true",
			LabelDeployment: deploymentID,
		},
	})
	if err != nil {
		// Check if it's a "volume already exists" error - use existing volume
		if strings.Contains(err.Error(), "already exists") {
			o.logger.Debug("volume already exists, reusing", "volume_name", volumeName)
			return volumeName, nil
		}
		return "", err
	}
	return volID, nil
}

// buildContainerSpec builds a ContainerSpec from a compose service.
// configMounts maps container paths to host file paths for config file bind mounts.
func (o *Orchestrator) buildContainerSpec(deployment *domain.Deployment, svc compose.Service, containerName, networkName string, volumes []compose.Volume, configMounts map[string]string) ContainerSpec {
	spec := ContainerSpec{
		Name:       containerName,
		Image:      svc.Image,
		Command:    svc.Command,
		Entrypoint: svc.Entrypoint,
		Env:        make(map[string]string),
		Labels: map[string]string{
			LabelManaged:    "true",
			LabelDeployment: deployment.ID,
			LabelTemplate:   deployment.TemplateID,
			LabelService:    svc.Name,
		},
		Networks: []string{networkName},
	}

	// Merge environment: service env + deployment variables
	for k, v := range svc.Environment {
		spec.Env[k] = coredeployment.SubstituteVariables(v, deployment.Variables)
	}

	// Port bindings
	for _, p := range svc.Ports {
		spec.Ports = append(spec.Ports, PortBinding{
			ContainerPort: int(p.Target),
			HostPort:      int(p.Published),
			Protocol:      p.Protocol,
			HostIP:        p.HostIP,
		})
	}

	// Volume mounts
	for _, v := range svc.Volumes {
		source := v.Source
		// Replace named volume with deployment-prefixed name
		if v.Type == compose.VolumeMountTypeVolume {
			source = coredeployment.VolumeName(deployment.ID, v.Source)
		}
		spec.Volumes = append(spec.Volumes, VolumeMount{
			Source:   source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}

	// Config file bind mounts
	for containerPath, hostPath := range configMounts {
		spec.Volumes = append(spec.Volumes, VolumeMount{
			Source:   hostPath,
			Target:   containerPath,
			ReadOnly: true, // Config files are read-only
		})
	}

	// Health check
	if svc.HealthCheck != nil {
		spec.HealthCheck = &HealthCheck{
			Test:    svc.HealthCheck.Test,
			Retries: svc.HealthCheck.Retries,
		}
		if svc.HealthCheck.Interval != "" {
			if d, err := time.ParseDuration(svc.HealthCheck.Interval); err == nil {
				spec.HealthCheck.Interval = d
			}
		}
		if svc.HealthCheck.Timeout != "" {
			if d, err := time.ParseDuration(svc.HealthCheck.Timeout); err == nil {
				spec.HealthCheck.Timeout = d
			}
		}
		if svc.HealthCheck.StartPeriod != "" {
			if d, err := time.ParseDuration(svc.HealthCheck.StartPeriod); err == nil {
				spec.HealthCheck.StartPeriod = d
			}
		}
	}

	// Resource limits
	if svc.Resources.CPULimit > 0 {
		spec.Resources.CPULimit = svc.Resources.CPULimit
	}
	if svc.Resources.MemoryLimit > 0 {
		spec.Resources.MemoryLimit = svc.Resources.MemoryLimit
	}

	// Restart policy
	switch svc.Restart {
	case compose.RestartAlways:
		spec.RestartPolicy = RestartPolicy{Name: "always"}
	case compose.RestartOnFailure:
		spec.RestartPolicy = RestartPolicy{Name: "on-failure"}
	case compose.RestartUnlessStopped:
		spec.RestartPolicy = RestartPolicy{Name: "unless-stopped"}
	default:
		spec.RestartPolicy = RestartPolicy{Name: "no"}
	}

	// Copy service labels
	for k, v := range svc.Labels {
		spec.Labels[k] = v
	}

	// Add Traefik labels if deployment has domains and service has ports
	if len(deployment.Domains) > 0 && len(svc.Ports) > 0 {
		traefikLabels := traefik.GenerateLabels(traefik.LabelParams{
			DeploymentID: deployment.ID,
			ServiceName:  svc.Name,
			Hostname:     deployment.Domains[0].Hostname,
			Port:         int(svc.Ports[0].Target),
			EnableTLS:    deployment.Domains[0].SSLEnabled,
		})
		for k, v := range traefikLabels {
			spec.Labels[k] = v
		}
	}

	return spec
}

// cleanupCreatedContainers stops and removes all created containers.
func (o *Orchestrator) cleanupCreatedContainers(ctx context.Context, containers map[string]string) {
	timeout := 5 * time.Second
	for name, id := range containers {
		_ = o.docker.StopContainer(id, &timeout)
		_ = o.docker.RemoveContainer(id, RemoveOptions{Force: true})
		o.logger.Debug("cleaned up container", "service", name, "container_id", id[:12])
	}
}

// convertPorts converts Docker port bindings to domain port mappings.
func (o *Orchestrator) convertPorts(ports []PortBinding) []domain.PortMapping {
	var result []domain.PortMapping
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

// =============================================================================
// Refresh Container Info
// =============================================================================

// RefreshContainerInfo refreshes the container info for a deployment.
func (o *Orchestrator) RefreshContainerInfo(ctx context.Context, deployment *domain.Deployment) ([]domain.ContainerInfo, error) {
	containers, err := o.docker.ListContainers(ListOptions{
		All: true,
		Filters: map[string]string{
			"label": fmt.Sprintf("%s=%s", LabelDeployment, deployment.ID),
		},
	})
	if err != nil {
		return nil, err
	}

	var result []domain.ContainerInfo
	for _, c := range containers {
		serviceName := ""
		if svc, ok := c.Labels[LabelService]; ok {
			serviceName = svc
		} else {
			// Extract from container name
			parts := strings.Split(c.Name, "_")
			if len(parts) >= 3 {
				serviceName = parts[len(parts)-1]
			}
		}

		result = append(result, domain.ContainerInfo{
			ID:          c.ID,
			ServiceName: serviceName,
			Image:       c.Image,
			Status:      string(c.Status),
			Ports:       o.convertPorts(c.Ports),
		})
	}

	return result, nil
}

// =============================================================================
// Config File Management
// =============================================================================

// writeConfigFiles writes config files to the host filesystem and returns a map
// of container paths to host paths for bind mounting.
func (o *Orchestrator) writeConfigFiles(deploymentID string, configFiles []domain.ConfigFile) (map[string]string, error) {
	mounts := make(map[string]string)

	if len(configFiles) == 0 {
		return mounts, nil
	}

	// Ensure config directory is an absolute path (required for Docker bind mounts)
	configDir := o.configDir
	if !filepath.IsAbs(configDir) {
		absDir, err := filepath.Abs(configDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for config dir: %w", err)
		}
		configDir = absDir
	}

	// Create deployment config directory
	deploymentDir := filepath.Join(configDir, deploymentID)
	if err := os.MkdirAll(deploymentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	for _, cf := range configFiles {
		// Sanitize the config file name for the host filesystem
		// Use a hash or sanitized version of the path as the filename
		hostFileName := sanitizeFileName(cf.Name)
		if hostFileName == "" {
			hostFileName = sanitizeFileName(filepath.Base(cf.Path))
		}
		hostPath := filepath.Join(deploymentDir, hostFileName)

		// Parse file mode (default to 0644)
		fileMode := os.FileMode(0644)
		if cf.Mode != "" {
			var mode uint32
			if _, err := fmt.Sscanf(cf.Mode, "%o", &mode); err == nil {
				fileMode = os.FileMode(mode)
			}
		}

		// Write the config file
		if err := os.WriteFile(hostPath, []byte(cf.Content), fileMode); err != nil {
			return nil, fmt.Errorf("failed to write config file %s: %w", cf.Name, err)
		}

		o.logger.Debug("wrote config file",
			"name", cf.Name,
			"host_path", hostPath,
			"container_path", cf.Path,
			"mode", cf.Mode,
		)

		// Map container path to host path
		mounts[cf.Path] = hostPath
	}

	return mounts, nil
}

// sanitizeFileName makes a filename safe for the filesystem.
func sanitizeFileName(name string) string {
	// Replace unsafe characters with underscores
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove leading/trailing underscores
	result = strings.Trim(result, "_")
	return result
}

// CleanupConfigFiles removes config files for a deployment.
func (o *Orchestrator) CleanupConfigFiles(deploymentID string) error {
	deploymentDir := filepath.Join(o.configDir, deploymentID)
	if err := os.RemoveAll(deploymentDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to cleanup config files: %w", err)
	}
	o.logger.Debug("cleaned up config files", "deployment_id", deploymentID)
	return nil
}
