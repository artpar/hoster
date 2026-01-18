package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/minion"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// createContainerCmd handles the "create-container" command.
// Reads ContainerSpec JSON from stdin.
func createContainerCmd() error {
	ctx := context.Background()

	// Read spec from stdin
	var spec minion.ContainerSpec
	if err := json.NewDecoder(os.Stdin).Decode(&spec); err != nil {
		outputError("create-container", minion.ErrCodeInvalidInput, "invalid JSON input: "+err.Error())
		return err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("create-container", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

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

	// Create container
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, spec.Name)
	if err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "Conflict") {
			code = minion.ErrCodeAlreadyExists
		} else if strings.Contains(err.Error(), "port is already allocated") {
			code = minion.ErrCodePortConflict
		}
		outputError("create-container", code, err.Error())
		return err
	}

	outputSuccess(minion.CreateResult{ID: resp.ID})
	return nil
}

// startContainerCmd handles the "start-container <id>" command.
func startContainerCmd(args []string) error {
	if len(args) < 1 {
		outputError("start-container", minion.ErrCodeInvalidInput, "usage: start-container <container_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	containerID := args[0]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("start-container", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "No such container") {
			code = minion.ErrCodeNotFound
		} else if strings.Contains(err.Error(), "is already running") {
			code = minion.ErrCodeAlreadyRunning
		}
		outputError("start-container", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}

// stopContainerCmd handles the "stop-container <id> [timeout_ms]" command.
func stopContainerCmd(args []string) error {
	if len(args) < 1 {
		outputError("stop-container", minion.ErrCodeInvalidInput, "usage: stop-container <container_id> [timeout_ms]")
		return errInvalidArgs
	}

	ctx := context.Background()
	containerID := args[0]

	var timeout *int
	if len(args) > 1 {
		ms, err := strconv.Atoi(args[1])
		if err == nil {
			secs := ms / 1000
			timeout = &secs
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("stop-container", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	opts := container.StopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}

	if err := cli.ContainerStop(ctx, containerID, opts); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "No such container") {
			code = minion.ErrCodeNotFound
		} else if strings.Contains(err.Error(), "is not running") {
			code = minion.ErrCodeNotRunning
		}
		outputError("stop-container", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}

// removeContainerCmd handles the "remove-container <id>" command.
// Reads RemoveOptions JSON from stdin (optional).
func removeContainerCmd(args []string) error {
	if len(args) < 1 {
		outputError("remove-container", minion.ErrCodeInvalidInput, "usage: remove-container <container_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	containerID := args[0]

	// Try to read options from stdin (optional)
	var opts minion.RemoveOptions
	_ = json.NewDecoder(os.Stdin).Decode(&opts) // Ignore error - stdin may be empty

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("remove-container", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	removeOpts := container.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.RemoveVolumes,
	}

	if err := cli.ContainerRemove(ctx, containerID, removeOpts); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "No such container") {
			code = minion.ErrCodeNotFound
		}
		outputError("remove-container", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}

// inspectContainerCmd handles the "inspect-container <id>" command.
func inspectContainerCmd(args []string) error {
	if len(args) < 1 {
		outputError("inspect-container", minion.ErrCodeInvalidInput, "usage: inspect-container <container_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	containerID := args[0]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("inspect-container", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "No such container") {
			code = minion.ErrCodeNotFound
		}
		outputError("inspect-container", code, err.Error())
		return err
	}

	info := convertContainerInspect(&inspect)
	outputSuccess(info)
	return nil
}

// listContainersCmd handles the "list-containers" command.
// Reads ListOptions JSON from stdin.
func listContainersCmd() error {
	ctx := context.Background()

	// Read options from stdin
	var opts minion.ListOptions
	_ = json.NewDecoder(os.Stdin).Decode(&opts) // Ignore error - stdin may be empty

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("list-containers", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	listOpts := container.ListOptions{
		All: opts.All,
	}

	// Build filters
	if len(opts.Filters) > 0 {
		f := filters.NewArgs()
		for k, v := range opts.Filters {
			f.Add(k, v)
		}
		listOpts.Filters = f
	}

	containers, err := cli.ContainerList(ctx, listOpts)
	if err != nil {
		outputError("list-containers", minion.ErrCodeInternal, err.Error())
		return err
	}

	// Convert to our format
	result := make([]minion.ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		info := minion.ContainerInfo{
			ID:        c.ID,
			Name:      name,
			Image:     c.Image,
			Status:    c.Status,
			State:     c.State,
			CreatedAt: time.Unix(c.Created, 0),
			Labels:    c.Labels,
		}

		// Convert ports
		for _, p := range c.Ports {
			info.Ports = append(info.Ports, minion.PortBinding{
				ContainerPort: int(p.PrivatePort),
				HostPort:      int(p.PublicPort),
				Protocol:      p.Type,
				HostIP:        p.IP,
			})
		}

		result = append(result, info)
	}

	outputSuccess(result)
	return nil
}

// containerLogsCmd handles the "container-logs <id>" command.
// Reads LogOptions JSON from stdin.
func containerLogsCmd(args []string) error {
	if len(args) < 1 {
		outputError("container-logs", minion.ErrCodeInvalidInput, "usage: container-logs <container_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	containerID := args[0]

	// Read options from stdin
	var opts minion.LogOptions
	_ = json.NewDecoder(os.Stdin).Decode(&opts)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("container-logs", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false, // Never follow in minion (would block)
		Timestamps: opts.Timestamps,
	}

	if opts.Tail != "" {
		logOpts.Tail = opts.Tail
	} else {
		logOpts.Tail = "100" // Default tail
	}

	if !opts.Since.IsZero() {
		logOpts.Since = opts.Since.Format(time.RFC3339)
	}
	if !opts.Until.IsZero() {
		logOpts.Until = opts.Until.Format(time.RFC3339)
	}

	reader, err := cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "No such container") {
			code = minion.ErrCodeNotFound
		}
		outputError("container-logs", code, err.Error())
		return err
	}
	defer reader.Close()

	// Read logs (limit to 64KB to avoid huge responses)
	buf := new(bytes.Buffer)
	_, _ = io.CopyN(buf, reader, 64*1024)

	outputSuccess(minion.LogsResult{Logs: buf.String()})
	return nil
}

// containerStatsCmd handles the "container-stats <id>" command.
func containerStatsCmd(args []string) error {
	if len(args) < 1 {
		outputError("container-stats", minion.ErrCodeInvalidInput, "usage: container-stats <container_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	containerID := args[0]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("container-stats", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	// Get one-shot stats
	statsResp, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "No such container") {
			code = minion.ErrCodeNotFound
		}
		outputError("container-stats", code, err.Error())
		return err
	}
	defer statsResp.Body.Close()

	// Parse stats JSON
	var statsJSON container.StatsResponse
	if err := json.NewDecoder(statsResp.Body).Decode(&statsJSON); err != nil {
		outputError("container-stats", minion.ErrCodeInternal, "failed to parse stats: "+err.Error())
		return err
	}

	stats := calculateStats(&statsJSON)
	outputSuccess(stats)
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

var errInvalidArgs = &commandError{msg: "invalid arguments"}

// convertContainerInspect converts Docker inspect result to our format.
func convertContainerInspect(inspect *container.InspectResponse) *minion.ContainerInfo {
	info := &minion.ContainerInfo{
		ID:     inspect.ID,
		Name:   strings.TrimPrefix(inspect.Name, "/"),
		Image:  inspect.Config.Image,
		State:  inspect.State.Status,
		Status: inspect.State.Status,
		Labels: inspect.Config.Labels,
	}

	// Parse timestamps
	if t, err := time.Parse(time.RFC3339Nano, inspect.Created); err == nil {
		info.CreatedAt = t
	}
	if inspect.State.StartedAt != "" && inspect.State.StartedAt != "0001-01-01T00:00:00Z" {
		if t, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil {
			info.StartedAt = &t
		}
	}
	if inspect.State.FinishedAt != "" && inspect.State.FinishedAt != "0001-01-01T00:00:00Z" {
		if t, err := time.Parse(time.RFC3339Nano, inspect.State.FinishedAt); err == nil {
			info.FinishedAt = &t
		}
	}

	// Health status
	if inspect.State.Health != nil {
		info.Health = inspect.State.Health.Status
	}

	// Exit code
	info.ExitCode = inspect.State.ExitCode

	// Port bindings
	if inspect.NetworkSettings != nil && len(inspect.NetworkSettings.Ports) > 0 {
		for portProto, bindings := range inspect.NetworkSettings.Ports {
			containerPort, _ := strconv.Atoi(portProto.Port())
			proto := portProto.Proto()

			for _, b := range bindings {
				hostPort, _ := strconv.Atoi(b.HostPort)
				info.Ports = append(info.Ports, minion.PortBinding{
					ContainerPort: containerPort,
					HostPort:      hostPort,
					Protocol:      proto,
					HostIP:        b.HostIP,
				})
			}
		}
	}

	return info
}

// calculateStats calculates resource stats from Docker stats response.
func calculateStats(stats *container.StatsResponse) *minion.ContainerResourceStats {
	result := &minion.ContainerResourceStats{}

	// CPU percentage
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpuCount := float64(stats.CPUStats.OnlineCPUs)
	if cpuCount == 0 {
		cpuCount = 1
	}
	if systemDelta > 0 && cpuDelta > 0 {
		result.CPUPercent = (cpuDelta / systemDelta) * cpuCount * 100.0
	}

	// Memory
	result.MemoryUsageBytes = int64(stats.MemoryStats.Usage)
	result.MemoryLimitBytes = int64(stats.MemoryStats.Limit)
	if result.MemoryLimitBytes > 0 {
		result.MemoryPercent = float64(result.MemoryUsageBytes) / float64(result.MemoryLimitBytes) * 100.0
	}

	// Network I/O
	for _, netStats := range stats.Networks {
		result.NetworkRxBytes += int64(netStats.RxBytes)
		result.NetworkTxBytes += int64(netStats.TxBytes)
	}

	// Block I/O
	for _, bioEntry := range stats.BlkioStats.IoServiceBytesRecursive {
		switch bioEntry.Op {
		case "Read", "read":
			result.BlockReadBytes += int64(bioEntry.Value)
		case "Write", "write":
			result.BlockWriteBytes += int64(bioEntry.Value)
		}
	}

	// PIDs
	result.PIDs = int(stats.PidsStats.Current)

	return result
}
