# F003: Docker Client

## Purpose

Manage Docker container lifecycle for deployments using the Docker SDK directly.

## Package Location

```
internal/shell/docker/
├── client.go        # Client interface + DockerClient implementation
├── types.go         # ContainerSpec, NetworkSpec, VolumeSpec
├── errors.go        # Error types
└── client_test.go   # Integration tests with real Docker
```

## Interface

```go
// Client defines the Docker client interface.
type Client interface {
    // Container operations
    CreateContainer(ctx context.Context, spec ContainerSpec) (containerID string, error)
    StartContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error
    RemoveContainer(ctx context.Context, containerID string, opts RemoveOptions) error
    InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error)
    ListContainers(ctx context.Context, opts ListOptions) ([]ContainerInfo, error)
    ContainerLogs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error)

    // Network operations
    CreateNetwork(ctx context.Context, spec NetworkSpec) (networkID string, error)
    RemoveNetwork(ctx context.Context, networkID string) error
    ConnectNetwork(ctx context.Context, networkID, containerID string) error
    DisconnectNetwork(ctx context.Context, networkID, containerID string, force bool) error

    // Volume operations
    CreateVolume(ctx context.Context, spec VolumeSpec) (volumeName string, error)
    RemoveVolume(ctx context.Context, volumeName string, force bool) error

    // Image operations
    PullImage(ctx context.Context, image string, opts PullOptions) error
    ImageExists(ctx context.Context, image string) (bool, error)

    // Health operations
    Ping(ctx context.Context) error
    Close() error
}
```

## Types

### ContainerSpec

```go
type ContainerSpec struct {
    Name        string
    Image       string
    Command     []string
    Entrypoint  []string
    Env         map[string]string
    Labels      map[string]string
    Ports       []PortBinding
    Volumes     []VolumeMount
    Networks    []string
    WorkingDir  string
    User        string
    RestartPolicy RestartPolicy
    Resources   ResourceLimits
    HealthCheck *HealthCheck
}

type PortBinding struct {
    ContainerPort int
    HostPort      int    // 0 for auto-assign
    Protocol      string // "tcp" or "udp"
    HostIP        string // "" for 0.0.0.0
}

type VolumeMount struct {
    Source   string // Volume name or host path
    Target   string // Container path
    ReadOnly bool
}

type RestartPolicy struct {
    Name              string // "no", "always", "on-failure", "unless-stopped"
    MaximumRetryCount int
}

type ResourceLimits struct {
    CPULimit    float64 // CPU cores
    MemoryLimit int64   // Bytes
}

type HealthCheck struct {
    Test        []string
    Interval    time.Duration
    Timeout     time.Duration
    Retries     int
    StartPeriod time.Duration
}
```

### ContainerInfo

```go
type ContainerInfo struct {
    ID          string
    Name        string
    Image       string
    Status      ContainerStatus
    State       string // "running", "exited", "created", etc.
    Health      string // "healthy", "unhealthy", "starting", ""
    CreatedAt   time.Time
    StartedAt   *time.Time
    FinishedAt  *time.Time
    Ports       []PortBinding
    Labels      map[string]string
    ExitCode    int
}

type ContainerStatus string

const (
    ContainerStatusCreated    ContainerStatus = "created"
    ContainerStatusRunning    ContainerStatus = "running"
    ContainerStatusPaused     ContainerStatus = "paused"
    ContainerStatusRestarting ContainerStatus = "restarting"
    ContainerStatusRemoving   ContainerStatus = "removing"
    ContainerStatusExited     ContainerStatus = "exited"
    ContainerStatusDead       ContainerStatus = "dead"
)
```

### NetworkSpec

```go
type NetworkSpec struct {
    Name   string
    Driver string // "bridge", "overlay", etc.
    Labels map[string]string
}
```

### VolumeSpec

```go
type VolumeSpec struct {
    Name   string
    Driver string
    Labels map[string]string
}
```

### Options

```go
type RemoveOptions struct {
    Force         bool
    RemoveVolumes bool
}

type ListOptions struct {
    All     bool              // Include stopped containers
    Filters map[string]string // e.g., {"label": "com.hoster.deployment=xyz"}
}

type LogOptions struct {
    Follow     bool
    Tail       string // "all" or number
    Since      time.Time
    Until      time.Time
    Timestamps bool
}

type PullOptions struct {
    Platform string // e.g., "linux/amd64"
}
```

## Error Types

```go
var (
    ErrContainerNotFound      = errors.New("container not found")
    ErrContainerAlreadyExists = errors.New("container already exists")
    ErrContainerNotRunning    = errors.New("container is not running")
    ErrContainerAlreadyRunning = errors.New("container is already running")

    ErrNetworkNotFound     = errors.New("network not found")
    ErrNetworkAlreadyExists = errors.New("network already exists")

    ErrVolumeNotFound = errors.New("volume not found")
    ErrVolumeInUse    = errors.New("volume is in use")

    ErrImageNotFound = errors.New("image not found")
    ErrImagePullFailed = errors.New("image pull failed")

    ErrPortAlreadyAllocated = errors.New("port is already allocated")
    ErrConnectionFailed     = errors.New("docker connection failed")
    ErrTimeout              = errors.New("operation timed out")
)
```

## Constructor

```go
// NewDockerClient creates a new Docker client.
// host can be:
//   - "" or "unix:///var/run/docker.sock" for local Docker
//   - "tcp://localhost:2375" for TCP connection
func NewDockerClient(host string) (*DockerClient, error)
```

## Behavior Specifications

### Container Lifecycle

1. **CreateContainer**
   - Creates container without starting it
   - Pulls image if not present locally (unless PullNever specified)
   - Generates container name if not provided
   - Returns container ID
   - Fails if container with same name exists

2. **StartContainer**
   - Starts a created/stopped container
   - No-op if already running (or error - TBD)
   - Returns error if container doesn't exist

3. **StopContainer**
   - Sends SIGTERM, then SIGKILL after timeout
   - Default timeout: 10 seconds
   - No-op if already stopped
   - Returns error if container doesn't exist

4. **RemoveContainer**
   - Removes container
   - With Force: also stops if running
   - With RemoveVolumes: removes anonymous volumes
   - Returns error if container doesn't exist (unless Force)

5. **InspectContainer**
   - Returns detailed container info
   - Returns error if container doesn't exist

6. **ListContainers**
   - Lists containers matching filters
   - By default, only running containers
   - With All: includes stopped containers

7. **ContainerLogs**
   - Returns log stream (io.ReadCloser)
   - With Follow: streams logs (caller must close)
   - With Tail: returns last N lines

### Network Operations

1. **CreateNetwork**
   - Creates Docker network
   - Default driver: "bridge"
   - Returns network ID

2. **RemoveNetwork**
   - Removes network
   - Fails if containers attached

3. **ConnectNetwork**
   - Connects container to network
   - Container can be running

4. **DisconnectNetwork**
   - Disconnects container from network
   - With Force: can disconnect running container

### Volume Operations

1. **CreateVolume**
   - Creates named volume
   - Default driver: "local"
   - Returns volume name

2. **RemoveVolume**
   - Removes volume
   - Fails if in use (unless Force)

### Image Operations

1. **PullImage**
   - Pulls image from registry
   - Parses image:tag correctly
   - Streams progress (internal)

2. **ImageExists**
   - Checks if image exists locally
   - Handles image:tag format

### Health Operations

1. **Ping**
   - Verifies Docker daemon is reachable
   - Quick health check

2. **Close**
   - Closes client connection
   - Safe to call multiple times

## Label Conventions

All Hoster-managed resources use labels for identification:

```
com.hoster.managed=true
com.hoster.deployment={deployment-id}
com.hoster.template={template-id}
com.hoster.service={service-name}
```

## Test Categories (~40 tests)

### Container Tests
- CreateContainer with minimal spec
- CreateContainer with full spec (ports, volumes, env)
- CreateContainer fails for existing name
- CreateContainer pulls image if missing
- StartContainer success
- StartContainer fails for missing container
- StopContainer success
- StopContainer with timeout
- StopContainer already stopped
- RemoveContainer success
- RemoveContainer force running
- RemoveContainer with volumes
- InspectContainer success
- InspectContainer not found
- ListContainers with filters
- ListContainers all
- ContainerLogs basic
- ContainerLogs with tail

### Network Tests
- CreateNetwork success
- CreateNetwork duplicate
- RemoveNetwork success
- RemoveNetwork with containers attached
- ConnectNetwork success
- DisconnectNetwork success

### Volume Tests
- CreateVolume success
- CreateVolume duplicate
- RemoveVolume success
- RemoveVolume in use
- RemoveVolume force

### Image Tests
- PullImage success
- PullImage not found
- ImageExists true
- ImageExists false

### Error Handling Tests
- Connection refused
- Context cancelled
- Port already allocated
- Resource constraints

### Health Tests
- Ping success
- Ping connection refused
- Close success

## Implementation Notes

### Using Docker SDK

```go
import (
    "github.com/docker/docker/client"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/api/types/volume"
)

func NewDockerClient(host string) (*DockerClient, error) {
    opts := []client.Opt{
        client.FromEnv,
        client.WithAPIVersionNegotiation(),
    }
    if host != "" {
        opts = append(opts, client.WithHost(host))
    }

    cli, err := client.NewClientWithOpts(opts...)
    if err != nil {
        return nil, err
    }

    return &DockerClient{client: cli}, nil
}
```

### Test Setup

Tests require a running Docker daemon. Use build tags or environment check:

```go
func skipIfNoDocker(t *testing.T) {
    cli, err := NewDockerClient("")
    if err != nil {
        t.Skip("Docker not available:", err)
    }
    if err := cli.Ping(context.Background()); err != nil {
        t.Skip("Docker not reachable:", err)
    }
    cli.Close()
}
```

### Cleanup

Tests must clean up created resources:

```go
func cleanup(t *testing.T, cli Client, containerID string) {
    t.Helper()
    ctx := context.Background()
    cli.StopContainer(ctx, containerID, nil)
    cli.RemoveContainer(ctx, containerID, RemoveOptions{Force: true, RemoveVolumes: true})
}
```

## NOT Supported (By Design)

- Docker Swarm operations
- Docker Compose file handling (use F001)
- Container exec (not needed for MVP)
- Container attach (not needed for MVP)
- Image build (templates use pre-built images)
- Multi-platform builds
- Registry authentication (public images only for MVP)
