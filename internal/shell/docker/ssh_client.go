package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/minion"
	"golang.org/x/crypto/ssh"
)

// SSHDockerClient implements the Client interface by executing minion commands via SSH.
// The minion binary must be deployed to the remote node.
type SSHDockerClient struct {
	node       *domain.Node
	sshClient  *ssh.Client
	signer     ssh.Signer
	minionPath string        // Path to minion binary on remote node
	timeout    time.Duration // Command timeout
	mu         sync.Mutex    // Protects sshClient
}

// SSHClientConfig configures the SSH Docker client.
type SSHClientConfig struct {
	MinionPath     string        // Default: ~/.hoster/minion
	CommandTimeout time.Duration // Default: 30 seconds
	ConnectTimeout time.Duration // Default: 10 seconds
}

// DefaultSSHClientConfig returns the default configuration.
func DefaultSSHClientConfig() SSHClientConfig {
	return SSHClientConfig{
		MinionPath:     "~/.hoster/minion",
		CommandTimeout: 30 * time.Second,
		ConnectTimeout: 10 * time.Second,
	}
}

// NewSSHDockerClient creates a new SSH-based Docker client.
// The privateKey should be the decrypted SSH private key.
func NewSSHDockerClient(node *domain.Node, privateKey []byte, config SSHClientConfig) (*SSHDockerClient, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse SSH private key: %w", err)
	}

	if config.MinionPath == "" {
		config.MinionPath = "~/.hoster/minion"
	}
	if config.CommandTimeout == 0 {
		config.CommandTimeout = 30 * time.Second
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = 10 * time.Second
	}

	return &SSHDockerClient{
		node:       node,
		signer:     signer,
		minionPath: config.MinionPath,
		timeout:    config.CommandTimeout,
	}, nil
}

// =============================================================================
// Connection Management
// =============================================================================

// connect establishes SSH connection if not already connected.
func (c *SSHDockerClient) connect(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sshClient != nil {
		// Check if connection is still alive
		_, _, err := c.sshClient.SendRequest("keepalive@hoster", true, nil)
		if err == nil {
			return nil
		}
		// Connection dead, reconnect
		c.sshClient.Close()
		c.sshClient = nil
	}

	config := &ssh.ClientConfig{
		User:            c.node.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(c.signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Store and verify host keys
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(c.node.SSHHost, strconv.Itoa(c.node.SSHPort))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	c.sshClient = client
	return nil
}

// Close closes the SSH connection.
func (c *SSHDockerClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sshClient != nil {
		err := c.sshClient.Close()
		c.sshClient = nil
		return err
	}
	return nil
}

// =============================================================================
// Minion Deployment
// =============================================================================

// EnsureMinion ensures the minion binary is deployed and up-to-date on the remote node.
// It checks if the minion exists and matches the expected version, uploading if needed.
func (c *SSHDockerClient) EnsureMinion(ctx context.Context, minionBinary []byte, expectedVersion string) error {
	if err := c.connect(ctx); err != nil {
		return err
	}

	// Check if minion exists and get version
	currentVersion, err := c.getMinionVersion(ctx)
	if err == nil && currentVersion == expectedVersion {
		// Minion exists and is up-to-date
		return nil
	}

	// Deploy minion binary
	return c.deployMinion(ctx, minionBinary)
}

// getMinionVersion returns the version of the minion binary on the remote node.
func (c *SSHDockerClient) getMinionVersion(ctx context.Context) (string, error) {
	c.mu.Lock()
	session, err := c.sshClient.NewSession()
	c.mu.Unlock()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	done := make(chan error, 1)
	go func() {
		done <- session.Run(c.minionPath + " version")
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(5 * time.Second):
		return "", fmt.Errorf("timeout checking minion version")
	case err := <-done:
		if err != nil {
			return "", err
		}
	}

	resp, err := minion.ParseResponse(stdout.Bytes())
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", fmt.Errorf("minion version check failed")
	}

	var version minion.VersionInfo
	if err := resp.UnmarshalData(&version); err != nil {
		return "", err
	}

	return version.Version, nil
}

// deployMinion uploads the minion binary to the remote node.
func (c *SSHDockerClient) deployMinion(ctx context.Context, binary []byte) error {
	c.mu.Lock()
	session, err := c.sshClient.NewSession()
	c.mu.Unlock()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Expand ~ to home directory using a simple mkdir command
	minionDir := "~/.hoster"
	minionPath := minionDir + "/minion"

	// Create directory and write file using cat
	// This avoids issues with tilde expansion
	cmd := fmt.Sprintf("mkdir -p %s && cat > %s && chmod +x %s", minionDir, minionPath, minionPath)

	session.Stdin = bytes.NewReader(binary)

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(60 * time.Second): // Allow more time for upload
		return fmt.Errorf("timeout deploying minion binary")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("deploy minion: %w", err)
		}
	}

	return nil
}

// =============================================================================
// Minion Execution
// =============================================================================

// execMinion executes a minion command via SSH and returns the response.
func (c *SSHDockerClient) execMinion(ctx context.Context, command string, args []string, input any) (*minion.Response, error) {
	if err := c.connect(ctx); err != nil {
		return nil, err
	}

	c.mu.Lock()
	session, err := c.sshClient.NewSession()
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("create SSH session: %w", err)
	}
	defer session.Close()

	// Build command
	cmdParts := []string{c.minionPath, command}
	cmdParts = append(cmdParts, args...)
	cmdStr := strings.Join(cmdParts, " ")

	// Set up stdin if input is provided
	var stdin io.Reader
	if input != nil {
		inputJSON, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("marshal input: %w", err)
		}
		stdin = bytes.NewReader(inputJSON)
		session.Stdin = stdin
	}

	// Capture stdout
	var stdout bytes.Buffer
	session.Stdout = &stdout

	// Run command with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmdStr)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("command timeout after %v", c.timeout)
	case err := <-done:
		// Parse response even if there was an exit error - minion writes JSON errors
		resp, parseErr := minion.ParseResponse(stdout.Bytes())
		if parseErr != nil {
			if err != nil {
				return nil, fmt.Errorf("command failed: %w, output: %s", err, stdout.String())
			}
			return nil, fmt.Errorf("parse response: %w", parseErr)
		}
		return resp, nil
	}
}

// translateError converts a minion error to a Docker error.
func (c *SSHDockerClient) translateError(errInfo *minion.ErrorInfo) error {
	switch errInfo.Code {
	case minion.ErrCodeNotFound:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrContainerNotFound)
	case minion.ErrCodeAlreadyExists:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrContainerAlreadyExists)
	case minion.ErrCodeNotRunning:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrContainerNotRunning)
	case minion.ErrCodeAlreadyRunning:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrContainerAlreadyRunning)
	case minion.ErrCodeInUse:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrNetworkInUse)
	case minion.ErrCodePortConflict:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrPortAlreadyAllocated)
	case minion.ErrCodeConnectionFailed:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrConnectionFailed)
	case minion.ErrCodePullFailed:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, ErrImagePullFailed)
	default:
		return NewDockerError(errInfo.Command, "", "", errInfo.Message, nil)
	}
}

// =============================================================================
// Container Operations
// =============================================================================

// CreateContainer creates a new container from the given spec.
func (c *SSHDockerClient) CreateContainer(spec ContainerSpec) (string, error) {
	ctx := context.Background()

	// Convert to minion spec
	mSpec := toMinionContainerSpec(spec)

	resp, err := c.execMinion(ctx, "create-container", nil, mSpec)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", c.translateError(resp.Error)
	}

	var result minion.CreateResult
	if err := resp.UnmarshalData(&result); err != nil {
		return "", fmt.Errorf("unmarshal result: %w", err)
	}
	return result.ID, nil
}

// StartContainer starts a stopped container.
func (c *SSHDockerClient) StartContainer(containerID string) error {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "start-container", []string{containerID}, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// StopContainer stops a running container.
func (c *SSHDockerClient) StopContainer(containerID string, timeout *time.Duration) error {
	ctx := context.Background()

	args := []string{containerID}
	if timeout != nil {
		args = append(args, strconv.FormatInt(timeout.Milliseconds(), 10))
	}

	resp, err := c.execMinion(ctx, "stop-container", args, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// RemoveContainer removes a container.
func (c *SSHDockerClient) RemoveContainer(containerID string, opts RemoveOptions) error {
	ctx := context.Background()

	mOpts := minion.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.RemoveVolumes,
	}

	resp, err := c.execMinion(ctx, "remove-container", []string{containerID}, mOpts)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// InspectContainer returns information about a container.
func (c *SSHDockerClient) InspectContainer(containerID string) (*ContainerInfo, error) {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "inspect-container", []string{containerID}, nil)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, c.translateError(resp.Error)
	}

	var mInfo minion.ContainerInfo
	if err := resp.UnmarshalData(&mInfo); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return fromMinionContainerInfo(&mInfo), nil
}

// ListContainers lists containers matching the options.
func (c *SSHDockerClient) ListContainers(opts ListOptions) ([]ContainerInfo, error) {
	ctx := context.Background()

	mOpts := minion.ListOptions{
		All:     opts.All,
		Filters: opts.Filters,
	}

	resp, err := c.execMinion(ctx, "list-containers", nil, mOpts)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, c.translateError(resp.Error)
	}

	var mInfos []minion.ContainerInfo
	if err := resp.UnmarshalData(&mInfos); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	result := make([]ContainerInfo, 0, len(mInfos))
	for _, m := range mInfos {
		result = append(result, *fromMinionContainerInfo(&m))
	}
	return result, nil
}

// ContainerLogs returns logs from a container.
func (c *SSHDockerClient) ContainerLogs(containerID string, opts LogOptions) (io.ReadCloser, error) {
	ctx := context.Background()

	mOpts := minion.LogOptions{
		Follow:     false, // Never follow in SSH mode
		Tail:       opts.Tail,
		Since:      opts.Since,
		Until:      opts.Until,
		Timestamps: opts.Timestamps,
	}

	resp, err := c.execMinion(ctx, "container-logs", []string{containerID}, mOpts)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, c.translateError(resp.Error)
	}

	var result minion.LogsResult
	if err := resp.UnmarshalData(&result); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return io.NopCloser(strings.NewReader(result.Logs)), nil
}

// ContainerStats returns resource statistics for a container.
func (c *SSHDockerClient) ContainerStats(containerID string) (*ContainerResourceStats, error) {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "container-stats", []string{containerID}, nil)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, c.translateError(resp.Error)
	}

	var mStats minion.ContainerResourceStats
	if err := resp.UnmarshalData(&mStats); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	return &ContainerResourceStats{
		CPUPercent:       mStats.CPUPercent,
		MemoryUsageBytes: mStats.MemoryUsageBytes,
		MemoryLimitBytes: mStats.MemoryLimitBytes,
		MemoryPercent:    mStats.MemoryPercent,
		NetworkRxBytes:   mStats.NetworkRxBytes,
		NetworkTxBytes:   mStats.NetworkTxBytes,
		BlockReadBytes:   mStats.BlockReadBytes,
		BlockWriteBytes:  mStats.BlockWriteBytes,
		PIDs:             mStats.PIDs,
	}, nil
}

// =============================================================================
// Network Operations
// =============================================================================

// CreateNetwork creates a new network.
func (c *SSHDockerClient) CreateNetwork(spec NetworkSpec) (string, error) {
	ctx := context.Background()

	mSpec := minion.NetworkSpec{
		Name:   spec.Name,
		Driver: spec.Driver,
		Labels: spec.Labels,
	}

	resp, err := c.execMinion(ctx, "create-network", nil, mSpec)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", c.translateError(resp.Error)
	}

	var result minion.CreateResult
	if err := resp.UnmarshalData(&result); err != nil {
		return "", fmt.Errorf("unmarshal result: %w", err)
	}
	return result.ID, nil
}

// RemoveNetwork removes a network.
func (c *SSHDockerClient) RemoveNetwork(networkID string) error {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "remove-network", []string{networkID}, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// ConnectNetwork connects a container to a network.
func (c *SSHDockerClient) ConnectNetwork(networkID, containerID string) error {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "connect-network", []string{networkID, containerID}, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// DisconnectNetwork disconnects a container from a network.
func (c *SSHDockerClient) DisconnectNetwork(networkID, containerID string, force bool) error {
	ctx := context.Background()

	args := []string{networkID, containerID}
	if force {
		args = append(args, "--force")
	}

	resp, err := c.execMinion(ctx, "disconnect-network", args, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// =============================================================================
// Volume Operations
// =============================================================================

// CreateVolume creates a new volume.
func (c *SSHDockerClient) CreateVolume(spec VolumeSpec) (string, error) {
	ctx := context.Background()

	mSpec := minion.VolumeSpec{
		Name:   spec.Name,
		Driver: spec.Driver,
		Labels: spec.Labels,
	}

	resp, err := c.execMinion(ctx, "create-volume", nil, mSpec)
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", c.translateError(resp.Error)
	}

	var result minion.VolumeCreateResult
	if err := resp.UnmarshalData(&result); err != nil {
		return "", fmt.Errorf("unmarshal result: %w", err)
	}
	return result.Name, nil
}

// RemoveVolume removes a volume.
func (c *SSHDockerClient) RemoveVolume(volumeName string, force bool) error {
	ctx := context.Background()

	args := []string{volumeName}
	if force {
		args = append(args, "--force")
	}

	resp, err := c.execMinion(ctx, "remove-volume", args, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// =============================================================================
// Image Operations
// =============================================================================

// PullImage pulls an image from a registry.
func (c *SSHDockerClient) PullImage(imageName string, opts PullOptions) error {
	ctx := context.Background()

	args := []string{imageName}
	if opts.Platform != "" {
		args = append(args, opts.Platform)
	}

	resp, err := c.execMinion(ctx, "pull-image", args, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// ImageExists checks if an image exists locally.
func (c *SSHDockerClient) ImageExists(imageName string) (bool, error) {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "image-exists", []string{imageName}, nil)
	if err != nil {
		return false, err
	}

	if !resp.Success {
		return false, c.translateError(resp.Error)
	}

	var result minion.ImageExistsResult
	if err := resp.UnmarshalData(&result); err != nil {
		return false, fmt.Errorf("unmarshal result: %w", err)
	}
	return result.Exists, nil
}

// =============================================================================
// Health Operations
// =============================================================================

// Ping checks if the Docker daemon is reachable.
func (c *SSHDockerClient) Ping() error {
	ctx := context.Background()

	resp, err := c.execMinion(ctx, "ping", nil, nil)
	if err != nil {
		return err
	}

	if !resp.Success {
		return c.translateError(resp.Error)
	}
	return nil
}

// =============================================================================
// Type Conversions
// =============================================================================

// toMinionContainerSpec converts our ContainerSpec to minion format.
func toMinionContainerSpec(spec ContainerSpec) minion.ContainerSpec {
	mSpec := minion.ContainerSpec{
		Name:       spec.Name,
		Image:      spec.Image,
		Command:    spec.Command,
		Entrypoint: spec.Entrypoint,
		Env:        spec.Env,
		Labels:     spec.Labels,
		Networks:   spec.Networks,
		WorkingDir: spec.WorkingDir,
		User:       spec.User,
		RestartPolicy: minion.RestartPolicy{
			Name:              spec.RestartPolicy.Name,
			MaximumRetryCount: spec.RestartPolicy.MaximumRetryCount,
		},
		Resources: minion.ResourceLimits{
			CPULimit:    spec.Resources.CPULimit,
			MemoryLimit: spec.Resources.MemoryLimit,
		},
	}

	for _, p := range spec.Ports {
		mSpec.Ports = append(mSpec.Ports, minion.PortBinding{
			ContainerPort: p.ContainerPort,
			HostPort:      p.HostPort,
			Protocol:      p.Protocol,
			HostIP:        p.HostIP,
		})
	}

	for _, v := range spec.Volumes {
		mSpec.Volumes = append(mSpec.Volumes, minion.VolumeMount{
			Source:   v.Source,
			Target:   v.Target,
			ReadOnly: v.ReadOnly,
		})
	}

	if spec.HealthCheck != nil {
		mSpec.HealthCheck = &minion.HealthCheck{
			Test:        spec.HealthCheck.Test,
			Interval:    spec.HealthCheck.Interval,
			Timeout:     spec.HealthCheck.Timeout,
			Retries:     spec.HealthCheck.Retries,
			StartPeriod: spec.HealthCheck.StartPeriod,
		}
	}

	return mSpec
}

// fromMinionContainerInfo converts minion ContainerInfo to our format.
func fromMinionContainerInfo(m *minion.ContainerInfo) *ContainerInfo {
	info := &ContainerInfo{
		ID:         m.ID,
		Name:       m.Name,
		Image:      m.Image,
		Status:     ContainerStatus(m.Status),
		State:      m.State,
		Health:     m.Health,
		CreatedAt:  m.CreatedAt,
		StartedAt:  m.StartedAt,
		FinishedAt: m.FinishedAt,
		Labels:     m.Labels,
		ExitCode:   m.ExitCode,
	}

	for _, p := range m.Ports {
		info.Ports = append(info.Ports, PortBinding{
			ContainerPort: p.ContainerPort,
			HostPort:      p.HostPort,
			Protocol:      p.Protocol,
			HostIP:        p.HostIP,
		})
	}

	return info
}
