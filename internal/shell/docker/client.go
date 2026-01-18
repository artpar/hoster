// Package docker provides a Docker client for container lifecycle management.
package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// =============================================================================
// Docker Client Implementation
// =============================================================================

// DockerClient implements the Client interface using the Docker SDK.
type DockerClient struct {
	cli *client.Client
}

// NewDockerClient creates a new Docker client.
// If host is empty, it uses the default Docker host from environment.
// On macOS with Docker Desktop, it automatically detects the correct socket.
func NewDockerClient(host string) (*DockerClient, error) {
	var opts []client.Opt
	opts = append(opts, client.FromEnv)
	opts = append(opts, client.WithAPIVersionNegotiation())

	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, NewDockerError("NewDockerClient", "", "", "failed to create client", ErrConnectionFailed)
	}

	// Try to ping with default settings
	ctx := context.Background()
	if _, pingErr := cli.Ping(ctx); pingErr != nil {
		// If default socket fails, try Docker Desktop socket on macOS
		homeDir, _ := os.UserHomeDir()
		dockerDesktopSocket := "unix://" + homeDir + "/.docker/run/docker.sock"

		// Try Docker Desktop socket
		cli2, err2 := client.NewClientWithOpts(
			client.WithHost(dockerDesktopSocket),
			client.WithAPIVersionNegotiation(),
		)
		if err2 == nil {
			if _, pingErr2 := cli2.Ping(ctx); pingErr2 == nil {
				// Docker Desktop socket works
				cli.Close()
				return &DockerClient{cli: cli2}, nil
			}
			cli2.Close()
		}
	}

	return &DockerClient{cli: cli}, nil
}

// Ping checks if Docker daemon is reachable.
func (d *DockerClient) Ping() error {
	ctx := context.Background()
	_, err := d.cli.Ping(ctx)
	if err != nil {
		return NewDockerError("Ping", "", "", fmt.Sprintf("failed to ping docker: %v", err), ErrConnectionFailed)
	}
	return nil
}

// Close closes the Docker client connection.
func (d *DockerClient) Close() error {
	return d.cli.Close()
}

// =============================================================================
// Container Operations
// =============================================================================

// CreateContainer creates a new container from the given spec.
func (d *DockerClient) CreateContainer(spec ContainerSpec) (string, error) {
	ctx := context.Background()

	// Build container config
	config := &container.Config{
		Image:      spec.Image,
		Cmd:        spec.Command,
		Entrypoint: spec.Entrypoint,
		WorkingDir: spec.WorkingDir,
		User:       spec.User,
		Labels:     spec.Labels,
	}

	// Set environment variables
	if len(spec.Env) > 0 {
		for k, v := range spec.Env {
			config.Env = append(config.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Build host config
	hostConfig := &container.HostConfig{}

	// Port bindings
	if len(spec.Ports) > 0 {
		portBindings := nat.PortMap{}
		exposedPorts := nat.PortSet{}

		for _, p := range spec.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			containerPort := nat.Port(fmt.Sprintf("%d/%s", p.ContainerPort, proto))
			exposedPorts[containerPort] = struct{}{}

			hostPort := ""
			if p.HostPort != 0 {
				hostPort = fmt.Sprintf("%d", p.HostPort)
			}

			portBindings[containerPort] = []nat.PortBinding{
				{
					HostIP:   p.HostIP,
					HostPort: hostPort,
				},
			}
		}

		config.ExposedPorts = exposedPorts
		hostConfig.PortBindings = portBindings
	}

	// Volume mounts
	if len(spec.Volumes) > 0 {
		for _, v := range spec.Volumes {
			var mountType mount.Type
			if strings.HasPrefix(v.Source, "/") {
				mountType = mount.TypeBind
			} else {
				mountType = mount.TypeVolume
			}

			hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
				Type:     mountType,
				Source:   v.Source,
				Target:   v.Target,
				ReadOnly: v.ReadOnly,
			})
		}
	}

	// Resource limits
	if spec.Resources.CPULimit > 0 {
		hostConfig.NanoCPUs = int64(spec.Resources.CPULimit * 1e9)
	}
	if spec.Resources.MemoryLimit > 0 {
		hostConfig.Memory = spec.Resources.MemoryLimit
	}

	// Restart policy
	if spec.RestartPolicy.Name != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{
			Name:              container.RestartPolicyMode(spec.RestartPolicy.Name),
			MaximumRetryCount: spec.RestartPolicy.MaximumRetryCount,
		}
	}

	// Health check
	if spec.HealthCheck != nil {
		config.Healthcheck = &container.HealthConfig{
			Test:        spec.HealthCheck.Test,
			Interval:    spec.HealthCheck.Interval,
			Timeout:     spec.HealthCheck.Timeout,
			Retries:     spec.HealthCheck.Retries,
			StartPeriod: spec.HealthCheck.StartPeriod,
		}
	}

	// Network config
	var networkConfig *network.NetworkingConfig
	if len(spec.Networks) > 0 {
		networkConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{},
		}
		for _, n := range spec.Networks {
			networkConfig.EndpointsConfig[n] = &network.EndpointSettings{}
		}
	}

	// Create the container
	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, spec.Name)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict") {
			return "", NewDockerError("CreateContainer", "container", spec.Name, "container already exists", ErrContainerAlreadyExists)
		}
		if strings.Contains(err.Error(), "port is already allocated") {
			return "", NewDockerError("CreateContainer", "container", spec.Name, err.Error(), ErrPortAlreadyAllocated)
		}
		return "", NewDockerError("CreateContainer", "container", spec.Name, err.Error(), err)
	}

	return resp.ID, nil
}

// StartContainer starts a stopped container.
func (d *DockerClient) StartContainer(containerID string) error {
	ctx := context.Background()
	err := d.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		if client.IsErrNotFound(err) {
			return NewDockerError("StartContainer", "container", containerID, "container not found", ErrContainerNotFound)
		}
		if strings.Contains(err.Error(), "is already running") {
			return NewDockerError("StartContainer", "container", containerID, "container is already running", ErrContainerAlreadyRunning)
		}
		return NewDockerError("StartContainer", "container", containerID, err.Error(), err)
	}
	return nil
}

// StopContainer stops a running container.
func (d *DockerClient) StopContainer(containerID string, timeout *time.Duration) error {
	ctx := context.Background()

	stopOptions := container.StopOptions{}
	if timeout != nil {
		seconds := int(timeout.Seconds())
		stopOptions.Timeout = &seconds
	}

	err := d.cli.ContainerStop(ctx, containerID, stopOptions)
	if err != nil {
		if client.IsErrNotFound(err) {
			return NewDockerError("StopContainer", "container", containerID, "container not found", ErrContainerNotFound)
		}
		if strings.Contains(err.Error(), "is not running") {
			return NewDockerError("StopContainer", "container", containerID, "container is not running", ErrContainerNotRunning)
		}
		return NewDockerError("StopContainer", "container", containerID, err.Error(), err)
	}
	return nil
}

// RemoveContainer removes a container.
func (d *DockerClient) RemoveContainer(containerID string, opts RemoveOptions) error {
	ctx := context.Background()

	removeOpts := container.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.RemoveVolumes,
	}

	err := d.cli.ContainerRemove(ctx, containerID, removeOpts)
	if err != nil {
		if client.IsErrNotFound(err) {
			return NewDockerError("RemoveContainer", "container", containerID, "container not found", ErrContainerNotFound)
		}
		return NewDockerError("RemoveContainer", "container", containerID, err.Error(), err)
	}
	return nil
}

// InspectContainer returns detailed information about a container.
func (d *DockerClient) InspectContainer(containerID string) (*ContainerInfo, error) {
	ctx := context.Background()

	resp, err := d.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, NewDockerError("InspectContainer", "container", containerID, "container not found", ErrContainerNotFound)
		}
		return nil, NewDockerError("InspectContainer", "container", containerID, err.Error(), err)
	}

	// Parse timestamps
	createdAt, _ := time.Parse(time.RFC3339Nano, resp.Created)

	var startedAt, finishedAt *time.Time
	if resp.State.StartedAt != "" && resp.State.StartedAt != "0001-01-01T00:00:00Z" {
		t, _ := time.Parse(time.RFC3339Nano, resp.State.StartedAt)
		startedAt = &t
	}
	if resp.State.FinishedAt != "" && resp.State.FinishedAt != "0001-01-01T00:00:00Z" {
		t, _ := time.Parse(time.RFC3339Nano, resp.State.FinishedAt)
		finishedAt = &t
	}

	// Get port bindings
	var ports []PortBinding
	for containerPort, bindings := range resp.NetworkSettings.Ports {
		port, proto := nat.Port(containerPort).Port(), nat.Port(containerPort).Proto()
		for _, binding := range bindings {
			var hostPort int
			if binding.HostPort != "" {
				fmt.Sscanf(binding.HostPort, "%d", &hostPort)
			}
			var containerPortInt int
			fmt.Sscanf(port, "%d", &containerPortInt)
			ports = append(ports, PortBinding{
				ContainerPort: containerPortInt,
				HostPort:      hostPort,
				Protocol:      proto,
				HostIP:        binding.HostIP,
			})
		}
	}

	// Determine health status
	health := ""
	if resp.State.Health != nil {
		health = resp.State.Health.Status
	}

	return &ContainerInfo{
		ID:         resp.ID,
		Name:       strings.TrimPrefix(resp.Name, "/"),
		Image:      resp.Config.Image,
		Status:     ContainerStatus(resp.State.Status),
		State:      resp.State.Status,
		Health:     health,
		CreatedAt:  createdAt,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Ports:      ports,
		Labels:     resp.Config.Labels,
		ExitCode:   resp.State.ExitCode,
	}, nil
}

// ListContainers returns a list of containers matching the given options.
func (d *DockerClient) ListContainers(opts ListOptions) ([]ContainerInfo, error) {
	ctx := context.Background()

	listOpts := container.ListOptions{
		All: opts.All,
	}

	if len(opts.Filters) > 0 {
		f := filters.NewArgs()
		for k, v := range opts.Filters {
			f.Add(k, v)
		}
		listOpts.Filters = f
	}

	containers, err := d.cli.ContainerList(ctx, listOpts)
	if err != nil {
		return nil, NewDockerError("ListContainers", "container", "", err.Error(), err)
	}

	var result []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		var ports []PortBinding
		for _, p := range c.Ports {
			ports = append(ports, PortBinding{
				ContainerPort: int(p.PrivatePort),
				HostPort:      int(p.PublicPort),
				Protocol:      p.Type,
				HostIP:        p.IP,
			})
		}

		result = append(result, ContainerInfo{
			ID:        c.ID,
			Name:      name,
			Image:     c.Image,
			Status:    ContainerStatus(c.State),
			State:     c.State,
			CreatedAt: time.Unix(c.Created, 0),
			Ports:     ports,
			Labels:    c.Labels,
		})
	}

	return result, nil
}

// ContainerLogs returns logs from a container.
func (d *DockerClient) ContainerLogs(containerID string, opts LogOptions) (io.ReadCloser, error) {
	ctx := context.Background()

	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Tail:       opts.Tail,
		Timestamps: opts.Timestamps,
	}

	if !opts.Since.IsZero() {
		logOpts.Since = opts.Since.Format(time.RFC3339)
	}
	if !opts.Until.IsZero() {
		logOpts.Until = opts.Until.Format(time.RFC3339)
	}

	reader, err := d.cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil, NewDockerError("ContainerLogs", "container", containerID, "container not found", ErrContainerNotFound)
		}
		return nil, NewDockerError("ContainerLogs", "container", containerID, err.Error(), err)
	}

	return reader, nil
}

// =============================================================================
// Network Operations
// =============================================================================

// CreateNetwork creates a new Docker network.
func (d *DockerClient) CreateNetwork(spec NetworkSpec) (string, error) {
	ctx := context.Background()

	driver := spec.Driver
	if driver == "" {
		driver = "bridge"
	}

	resp, err := d.cli.NetworkCreate(ctx, spec.Name, network.CreateOptions{
		Driver: driver,
		Labels: spec.Labels,
	})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return "", NewDockerError("CreateNetwork", "network", spec.Name, "network already exists", ErrNetworkAlreadyExists)
		}
		return "", NewDockerError("CreateNetwork", "network", spec.Name, err.Error(), err)
	}

	return resp.ID, nil
}

// RemoveNetwork removes a Docker network.
func (d *DockerClient) RemoveNetwork(networkID string) error {
	ctx := context.Background()

	err := d.cli.NetworkRemove(ctx, networkID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return NewDockerError("RemoveNetwork", "network", networkID, "network not found", ErrNetworkNotFound)
		}
		if strings.Contains(err.Error(), "has active endpoints") {
			return NewDockerError("RemoveNetwork", "network", networkID, "network has active endpoints", ErrNetworkInUse)
		}
		return NewDockerError("RemoveNetwork", "network", networkID, err.Error(), err)
	}
	return nil
}

// ConnectNetwork connects a container to a network.
func (d *DockerClient) ConnectNetwork(networkID, containerID string) error {
	ctx := context.Background()

	err := d.cli.NetworkConnect(ctx, networkID, containerID, nil)
	if err != nil {
		if client.IsErrNotFound(err) {
			if strings.Contains(err.Error(), "network") {
				return NewDockerError("ConnectNetwork", "network", networkID, "network not found", ErrNetworkNotFound)
			}
			return NewDockerError("ConnectNetwork", "container", containerID, "container not found", ErrContainerNotFound)
		}
		return NewDockerError("ConnectNetwork", "network", networkID, err.Error(), err)
	}
	return nil
}

// DisconnectNetwork disconnects a container from a network.
func (d *DockerClient) DisconnectNetwork(networkID, containerID string, force bool) error {
	ctx := context.Background()

	err := d.cli.NetworkDisconnect(ctx, networkID, containerID, force)
	if err != nil {
		if client.IsErrNotFound(err) {
			if strings.Contains(err.Error(), "network") {
				return NewDockerError("DisconnectNetwork", "network", networkID, "network not found", ErrNetworkNotFound)
			}
			return NewDockerError("DisconnectNetwork", "container", containerID, "container not found", ErrContainerNotFound)
		}
		return NewDockerError("DisconnectNetwork", "network", networkID, err.Error(), err)
	}
	return nil
}

// =============================================================================
// Volume Operations
// =============================================================================

// CreateVolume creates a new Docker volume.
func (d *DockerClient) CreateVolume(spec VolumeSpec) (string, error) {
	ctx := context.Background()

	driver := spec.Driver
	if driver == "" {
		driver = "local"
	}

	resp, err := d.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   spec.Name,
		Driver: driver,
		Labels: spec.Labels,
	})
	if err != nil {
		return "", NewDockerError("CreateVolume", "volume", spec.Name, err.Error(), err)
	}

	return resp.Name, nil
}

// RemoveVolume removes a Docker volume.
func (d *DockerClient) RemoveVolume(volumeName string, force bool) error {
	ctx := context.Background()

	err := d.cli.VolumeRemove(ctx, volumeName, force)
	if err != nil {
		if client.IsErrNotFound(err) {
			return NewDockerError("RemoveVolume", "volume", volumeName, "volume not found", ErrVolumeNotFound)
		}
		if strings.Contains(err.Error(), "in use") {
			return NewDockerError("RemoveVolume", "volume", volumeName, "volume is in use", ErrVolumeInUse)
		}
		return NewDockerError("RemoveVolume", "volume", volumeName, err.Error(), err)
	}
	return nil
}

// =============================================================================
// Image Operations
// =============================================================================

// PullImage pulls an image from the registry.
func (d *DockerClient) PullImage(imageName string, opts PullOptions) error {
	ctx := context.Background()

	pullOpts := image.PullOptions{}
	if opts.Platform != "" {
		pullOpts.Platform = opts.Platform
	}

	reader, err := d.cli.ImagePull(ctx, imageName, pullOpts)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "manifest unknown") ||
			strings.Contains(errStr, "repository does not exist") ||
			strings.Contains(errStr, "pull access denied") {
			return NewDockerError("PullImage", "image", imageName, "image not found", ErrImageNotFound)
		}
		return NewDockerError("PullImage", "image", imageName, err.Error(), ErrImagePullFailed)
	}
	defer reader.Close()

	// Drain the reader to complete the pull
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return NewDockerError("PullImage", "image", imageName, err.Error(), ErrImagePullFailed)
	}

	return nil
}

// ImageExists checks if an image exists locally.
func (d *DockerClient) ImageExists(imageName string) (bool, error) {
	ctx := context.Background()

	_, _, err := d.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, NewDockerError("ImageExists", "image", imageName, err.Error(), err)
	}

	return true, nil
}
