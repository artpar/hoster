package docker

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

func skipIfNoDocker(t *testing.T) Client {
	t.Helper()
	cli, err := NewDockerClient("")
	if err != nil {
		t.Skip("Docker not available:", err)
	}
	if err := cli.Ping(); err != nil {
		cli.Close()
		t.Skip("Docker not reachable:", err)
	}
	return cli
}

func cleanupContainer(t *testing.T, cli Client, containerID string) {
	t.Helper()
	timeout := 5 * time.Second
	cli.StopContainer(containerID, &timeout)
	cli.RemoveContainer(containerID, RemoveOptions{Force: true, RemoveVolumes: true})
}

func cleanupNetwork(t *testing.T, cli Client, networkID string) {
	t.Helper()
	cli.RemoveNetwork(networkID)
}

func cleanupVolume(t *testing.T, cli Client, volumeName string) {
	t.Helper()
	cli.RemoveVolume(volumeName, true)
}

// Test container name prefix to identify test containers
const testPrefix = "hoster-test-"

// =============================================================================
// Connection Tests
// =============================================================================

func TestNewDockerClient_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// Client was created successfully
	assert.NotNil(t, cli)
}

func TestPing_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	err := cli.Ping()
	assert.NoError(t, err)
}

func TestClose_Success(t *testing.T) {
	cli := skipIfNoDocker(t)

	err := cli.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Container Creation Tests
// =============================================================================

func TestCreateContainer_Minimal(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "minimal",
		Image: "alpine:latest",
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

func TestCreateContainer_WithCommand(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:    testPrefix + "with-cmd",
		Image:   "alpine:latest",
		Command: []string{"echo", "hello"},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

func TestCreateContainer_WithEnv(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "with-env",
		Image: "alpine:latest",
		Env: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

func TestCreateContainer_WithLabels(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "with-labels",
		Image: "alpine:latest",
		Labels: map[string]string{
			LabelManaged:    "true",
			LabelDeployment: "test-deployment",
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	// Verify labels
	info, err := cli.InspectContainer(containerID)
	require.NoError(t, err)
	assert.Equal(t, "true", info.Labels[LabelManaged])
	assert.Equal(t, "test-deployment", info.Labels[LabelDeployment])
}

func TestCreateContainer_WithPorts(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "with-ports",
		Image: "alpine:latest",
		Ports: []PortBinding{
			{ContainerPort: 80, HostPort: 0, Protocol: "tcp"},
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

func TestCreateContainer_DuplicateName(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "duplicate",
		Image: "alpine:latest",
	}

	// Create first container
	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	// Try to create second with same name
	_, err = cli.CreateContainer(spec)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContainerAlreadyExists)
}

// =============================================================================
// Container Lifecycle Tests
// =============================================================================

func TestStartContainer_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:    testPrefix + "start",
		Image:   "alpine:latest",
		Command: []string{"sleep", "30"},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	err = cli.StartContainer(containerID)
	require.NoError(t, err)

	// Verify it's running
	info, err := cli.InspectContainer(containerID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStatusRunning, info.Status)
}

func TestStartContainer_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	err := cli.StartContainer("nonexistent-container-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

func TestStopContainer_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:    testPrefix + "stop",
		Image:   "alpine:latest",
		Command: []string{"sleep", "300"},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	err = cli.StartContainer(containerID)
	require.NoError(t, err)

	timeout := 5 * time.Second
	err = cli.StopContainer(containerID, &timeout)
	require.NoError(t, err)

	// Verify it's stopped
	info, err := cli.InspectContainer(containerID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStatusExited, info.Status)
}

func TestStopContainer_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	timeout := 5 * time.Second
	err := cli.StopContainer("nonexistent-container-id", &timeout)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

func TestRemoveContainer_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "remove",
		Image: "alpine:latest",
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)

	err = cli.RemoveContainer(containerID, RemoveOptions{})
	require.NoError(t, err)

	// Verify it's gone
	_, err = cli.InspectContainer(containerID)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

func TestRemoveContainer_ForceRunning(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:    testPrefix + "force-remove",
		Image:   "alpine:latest",
		Command: []string{"sleep", "300"},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)

	err = cli.StartContainer(containerID)
	require.NoError(t, err)

	// Remove with force
	err = cli.RemoveContainer(containerID, RemoveOptions{Force: true})
	require.NoError(t, err)

	// Verify it's gone
	_, err = cli.InspectContainer(containerID)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

func TestRemoveContainer_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	err := cli.RemoveContainer("nonexistent-container-id", RemoveOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

// =============================================================================
// Container Inspection Tests
// =============================================================================

func TestInspectContainer_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "inspect",
		Image: "alpine:latest",
		Labels: map[string]string{
			"test": "value",
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	info, err := cli.InspectContainer(containerID)
	require.NoError(t, err)

	assert.Equal(t, containerID, info.ID)
	assert.Contains(t, info.Name, testPrefix+"inspect")
	assert.Equal(t, "alpine:latest", info.Image)
	assert.Equal(t, "value", info.Labels["test"])
}

func TestInspectContainer_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	_, err := cli.InspectContainer("nonexistent-container-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

// =============================================================================
// Container List Tests
// =============================================================================

func TestListContainers_Empty(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// List with a filter that won't match anything
	containers, err := cli.ListContainers(ListOptions{
		All: true,
		Filters: map[string]string{
			"label": "com.hoster.test=nonexistent-unique-value",
		},
	})
	require.NoError(t, err)
	assert.Empty(t, containers)
}

func TestListContainers_WithFilter(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	uniqueLabel := "com.hoster.test=" + testPrefix + "list"

	spec := ContainerSpec{
		Name:  testPrefix + "list",
		Image: "alpine:latest",
		Labels: map[string]string{
			"com.hoster.test": testPrefix + "list",
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	containers, err := cli.ListContainers(ListOptions{
		All: true,
		Filters: map[string]string{
			"label": uniqueLabel,
		},
	})
	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, containerID, containers[0].ID)
}

// =============================================================================
// Container Logs Tests
// =============================================================================

func TestContainerLogs_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:    testPrefix + "logs",
		Image:   "alpine:latest",
		Command: []string{"echo", "hello from container"},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	err = cli.StartContainer(containerID)
	require.NoError(t, err)

	// Wait for container to finish
	time.Sleep(2 * time.Second)

	logs, err := cli.ContainerLogs(containerID, LogOptions{Tail: "10"})
	require.NoError(t, err)
	defer logs.Close()

	output, err := io.ReadAll(logs)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello from container")
}

// =============================================================================
// Network Tests
// =============================================================================

func TestCreateNetwork_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := NetworkSpec{
		Name:   testPrefix + "network",
		Driver: "bridge",
		Labels: map[string]string{
			LabelManaged: "true",
		},
	}

	networkID, err := cli.CreateNetwork(spec)
	require.NoError(t, err)
	defer cleanupNetwork(t, cli, networkID)

	assert.NotEmpty(t, networkID)
}

func TestRemoveNetwork_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := NetworkSpec{
		Name:   testPrefix + "network-remove",
		Driver: "bridge",
	}

	networkID, err := cli.CreateNetwork(spec)
	require.NoError(t, err)

	err = cli.RemoveNetwork(networkID)
	require.NoError(t, err)
}

func TestRemoveNetwork_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	err := cli.RemoveNetwork("nonexistent-network-id")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNetworkNotFound)
}

func TestConnectNetwork_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// Create network
	netSpec := NetworkSpec{
		Name:   testPrefix + "connect-net",
		Driver: "bridge",
	}
	networkID, err := cli.CreateNetwork(netSpec)
	require.NoError(t, err)
	defer cleanupNetwork(t, cli, networkID)

	// Create container
	containerSpec := ContainerSpec{
		Name:  testPrefix + "connect-container",
		Image: "alpine:latest",
	}
	containerID, err := cli.CreateContainer(containerSpec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	// Connect
	err = cli.ConnectNetwork(networkID, containerID)
	require.NoError(t, err)
}

func TestDisconnectNetwork_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// Create network
	netSpec := NetworkSpec{
		Name:   testPrefix + "disconnect-net",
		Driver: "bridge",
	}
	networkID, err := cli.CreateNetwork(netSpec)
	require.NoError(t, err)
	defer cleanupNetwork(t, cli, networkID)

	// Create container with network
	containerSpec := ContainerSpec{
		Name:     testPrefix + "disconnect-container",
		Image:    "alpine:latest",
		Networks: []string{testPrefix + "disconnect-net"},
	}
	containerID, err := cli.CreateContainer(containerSpec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	// Disconnect
	err = cli.DisconnectNetwork(networkID, containerID, false)
	require.NoError(t, err)
}

// =============================================================================
// Volume Tests
// =============================================================================

func TestCreateVolume_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := VolumeSpec{
		Name:   testPrefix + "volume",
		Driver: "local",
		Labels: map[string]string{
			LabelManaged: "true",
		},
	}

	volumeName, err := cli.CreateVolume(spec)
	require.NoError(t, err)
	defer cleanupVolume(t, cli, volumeName)

	assert.Equal(t, testPrefix+"volume", volumeName)
}

func TestRemoveVolume_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := VolumeSpec{
		Name:   testPrefix + "volume-remove",
		Driver: "local",
	}

	volumeName, err := cli.CreateVolume(spec)
	require.NoError(t, err)

	err = cli.RemoveVolume(volumeName, false)
	require.NoError(t, err)
}

func TestRemoveVolume_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	err := cli.RemoveVolume("nonexistent-volume-name", false)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrVolumeNotFound)
}

// =============================================================================
// Image Tests
// =============================================================================

func TestPullImage_Success(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// Use a small image
	err := cli.PullImage("alpine:latest", PullOptions{})
	require.NoError(t, err)
}

func TestPullImage_NotFound(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	err := cli.PullImage("nonexistent-image-12345:latest", PullOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrImageNotFound)
}

func TestImageExists_True(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// Pull first to ensure it exists
	err := cli.PullImage("alpine:latest", PullOptions{})
	require.NoError(t, err)

	exists, err := cli.ImageExists("alpine:latest")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestImageExists_False(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	exists, err := cli.ImageExists("nonexistent-image-12345:latest")
	require.NoError(t, err)
	assert.False(t, exists)
}

// =============================================================================
// Error Tests
// =============================================================================

func TestDockerError_Error(t *testing.T) {
	// With all fields
	err := NewDockerError("CreateContainer", "container", "abc123", "failed to create", ErrContainerAlreadyExists)
	assert.Equal(t, "CreateContainer container abc123: failed to create", err.Error())

	// Without ID
	err = NewDockerError("ListContainers", "container", "", "connection failed", ErrConnectionFailed)
	assert.Equal(t, "ListContainers container: connection failed", err.Error())

	// Without entity
	err = NewDockerError("Ping", "", "", "connection refused", nil)
	assert.Equal(t, "Ping: connection refused", err.Error())
}

func TestDockerError_Unwrap(t *testing.T) {
	err := NewDockerError("CreateContainer", "container", "abc123", "already exists", ErrContainerAlreadyExists)
	assert.ErrorIs(t, err, ErrContainerAlreadyExists)
}

// =============================================================================
// Resource Limits Tests
// =============================================================================

func TestCreateContainer_WithResourceLimits(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "resources",
		Image: "alpine:latest",
		Resources: ResourceLimits{
			CPULimit:    0.5,             // Half a CPU
			MemoryLimit: 64 * 1024 * 1024, // 64MB
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

// =============================================================================
// Restart Policy Tests
// =============================================================================

func TestCreateContainer_WithRestartPolicy(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Name:  testPrefix + "restart",
		Image: "alpine:latest",
		RestartPolicy: RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 3,
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

// =============================================================================
// Volume Mount Tests
// =============================================================================

func TestCreateContainer_WithVolumeMounts(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// Create volume first
	volSpec := VolumeSpec{
		Name:   testPrefix + "mount-vol",
		Driver: "local",
	}
	volumeName, err := cli.CreateVolume(volSpec)
	require.NoError(t, err)
	defer cleanupVolume(t, cli, volumeName)

	spec := ContainerSpec{
		Name:  testPrefix + "with-mounts",
		Image: "alpine:latest",
		Volumes: []VolumeMount{
			{Source: volumeName, Target: "/data", ReadOnly: false},
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	assert.NotEmpty(t, containerID)
}

// =============================================================================
// Status Parsing Tests
// =============================================================================

func TestContainerStatus_Values(t *testing.T) {
	assert.Equal(t, ContainerStatus("created"), ContainerStatusCreated)
	assert.Equal(t, ContainerStatus("running"), ContainerStatusRunning)
	assert.Equal(t, ContainerStatus("paused"), ContainerStatusPaused)
	assert.Equal(t, ContainerStatus("restarting"), ContainerStatusRestarting)
	assert.Equal(t, ContainerStatus("removing"), ContainerStatusRemoving)
	assert.Equal(t, ContainerStatus("exited"), ContainerStatusExited)
	assert.Equal(t, ContainerStatus("dead"), ContainerStatusDead)
}

// =============================================================================
// Label Constants Tests
// =============================================================================

func TestLabelConstants(t *testing.T) {
	assert.Equal(t, "com.hoster.managed", LabelManaged)
	assert.Equal(t, "com.hoster.deployment", LabelDeployment)
	assert.Equal(t, "com.hoster.template", LabelTemplate)
	assert.Equal(t, "com.hoster.service", LabelService)
}

// =============================================================================
// Integration Test - Full Lifecycle
// =============================================================================

func TestContainerFullLifecycle(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	// 1. Create network
	netSpec := NetworkSpec{
		Name:   testPrefix + "lifecycle-net",
		Driver: "bridge",
	}
	networkID, err := cli.CreateNetwork(netSpec)
	require.NoError(t, err)
	defer cleanupNetwork(t, cli, networkID)

	// 2. Create volume
	volSpec := VolumeSpec{
		Name:   testPrefix + "lifecycle-vol",
		Driver: "local",
	}
	volumeName, err := cli.CreateVolume(volSpec)
	require.NoError(t, err)
	defer cleanupVolume(t, cli, volumeName)

	// 3. Create container
	containerSpec := ContainerSpec{
		Name:     testPrefix + "lifecycle",
		Image:    "alpine:latest",
		Command:  []string{"sleep", "30"},
		Networks: []string{testPrefix + "lifecycle-net"},
		Volumes: []VolumeMount{
			{Source: volumeName, Target: "/data"},
		},
		Labels: map[string]string{
			LabelManaged:    "true",
			LabelDeployment: "test-deployment",
		},
	}

	containerID, err := cli.CreateContainer(containerSpec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	// 4. Start container
	err = cli.StartContainer(containerID)
	require.NoError(t, err)

	// 5. Verify running
	info, err := cli.InspectContainer(containerID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStatusRunning, info.Status)

	// 6. Stop container
	timeout := 5 * time.Second
	err = cli.StopContainer(containerID, &timeout)
	require.NoError(t, err)

	// 7. Verify stopped
	info, err = cli.InspectContainer(containerID)
	require.NoError(t, err)
	assert.Equal(t, ContainerStatusExited, info.Status)

	// 8. Remove container
	err = cli.RemoveContainer(containerID, RemoveOptions{RemoveVolumes: true})
	require.NoError(t, err)

	// 9. Verify removed
	_, err = cli.InspectContainer(containerID)
	assert.ErrorIs(t, err, ErrContainerNotFound)
}

// =============================================================================
// Auto-Generated Container Name Test
// =============================================================================

func TestCreateContainer_AutoName(t *testing.T) {
	cli := skipIfNoDocker(t)
	defer cli.Close()

	spec := ContainerSpec{
		Image: "alpine:latest",
		Labels: map[string]string{
			"com.hoster.test": "auto-name",
		},
	}

	containerID, err := cli.CreateContainer(spec)
	require.NoError(t, err)
	defer cleanupContainer(t, cli, containerID)

	info, err := cli.InspectContainer(containerID)
	require.NoError(t, err)

	// Docker generates a name like adjective_noun (we strip the leading /)
	assert.NotEmpty(t, info.Name)
	// Name should contain underscore (Docker's random naming pattern)
	assert.Contains(t, info.Name, "_")
}
