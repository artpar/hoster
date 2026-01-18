# Node

## Overview

A Node represents a VPS server registered by a creator for running deployments. Nodes are connected via SSH to access the remote Docker daemon. Creators register their own infrastructure and assign capability tags to indicate what workloads the node can handle.

## Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | UUID | Yes (auto) | Unique identifier, generated on creation |
| `name` | string | Yes | Human-readable name (3-100 chars) |
| `creator_id` | UUID | Yes | Who owns this node |
| `ssh_host` | string | Yes | SSH hostname or IP address |
| `ssh_port` | int | Yes | SSH port (default 22) |
| `ssh_user` | string | Yes | SSH username |
| `ssh_key_id` | UUID | No | Reference to stored SSH key (encrypted) |
| `docker_socket` | string | No | Remote Docker socket path (default /var/run/docker.sock) |
| `status` | NodeStatus | Yes | Current operational status |
| `capabilities` | []string | Yes | Node capability tags (e.g., ["standard", "gpu", "ssd"]) |
| `capacity` | NodeCapacity | Yes | Resource capacity and usage |
| `location` | string | No | Geographic location/region for display |
| `last_health_check` | timestamp | No | When last health check ran |
| `error_message` | string | No | Last error message if offline |
| `created_at` | timestamp | Yes (auto) | When created |
| `updated_at` | timestamp | Yes (auto) | When last modified |

### NodeStatus Enum

| Value | Description |
|-------|-------------|
| `online` | Node is reachable and Docker is running |
| `offline` | Node is not reachable or Docker is not responding |
| `maintenance` | Node is temporarily unavailable (user-set) |

### NodeCapacity Type

| Field | Type | Description |
|-------|------|-------------|
| `cpu_cores` | float64 | Total CPU cores available |
| `memory_mb` | int64 | Total RAM in MB |
| `disk_mb` | int64 | Total disk space in MB |
| `cpu_used` | float64 | Currently used CPU cores |
| `memory_used_mb` | int64 | Currently used RAM in MB |
| `disk_used_mb` | int64 | Currently used disk in MB |

### SSHKey Type (stored separately)

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique identifier |
| `creator_id` | UUID | Owner of this key |
| `name` | string | Key name for identification |
| `private_key_encrypted` | bytes | AES-256-GCM encrypted private key |
| `fingerprint` | string | SHA256 fingerprint of public key |
| `created_at` | timestamp | When created |

## Standard Capabilities

The following capability tags are predefined. Creators can use custom tags as well.

| Capability | Description |
|------------|-------------|
| `standard` | General-purpose compute node |
| `gpu` | Node has GPU available |
| `high-memory` | Node has extra RAM (32GB+) |
| `high-cpu` | Node has extra CPU cores (8+) |
| `ssd` | Node has SSD storage |
| `nvme` | Node has NVMe storage |

## Invariants

1. **Name is required**: Must be 3-100 characters
2. **Name is unique per creator**: No two nodes with same name for same creator
3. **SSH host is valid**: Must be valid hostname or IP address
4. **SSH port is valid**: Must be 1-65535
5. **SSH user is required**: Non-empty string
6. **Capabilities non-empty**: Must have at least one capability (default: ["standard"])
7. **Creator must exist**: Cannot create node for non-existent creator
8. **Only one status at a time**: Status is a single enum value

## Behaviors

### Health Check
- Connect via SSH and run `docker info`
- Update `status`, `last_health_check`, and capacity metrics
- Run periodically (every 60 seconds) and on-demand
- On failure, set `status = offline` and record error message

### Capacity Calculation
- Query `docker system df` for disk usage
- Query container stats for CPU/memory usage
- Calculate available capacity: total - used

### Node Selection (Scheduler)
When scheduling a deployment, the scheduler:
1. Get template's `required_capabilities`
2. Get user's plan `allowed_capabilities`
3. Filter nodes by: `status = online`, capabilities match, sufficient capacity
4. Score by: available resources / total resources
5. Return highest-scoring node

```
Score = (available_cpu / total_cpu) * 0.3 +
        (available_memory / total_memory) * 0.4 +
        (available_disk / total_disk) * 0.3
```

### Capacity Helpers
```go
func (c NodeCapacity) AvailableCPU() float64 {
    return c.CPUCores - c.CPUUsed
}

func (c NodeCapacity) AvailableMemory() int64 {
    return c.MemoryMB - c.MemoryUsedMB
}

func (c NodeCapacity) AvailableDisk() int64 {
    return c.DiskMB - c.DiskUsedMB
}

func (c NodeCapacity) CanHandle(required ResourceRequirements) bool {
    return c.AvailableCPU() >= required.CPUCores &&
           c.AvailableMemory() >= required.MemoryMB &&
           c.AvailableDisk() >= required.DiskMB
}
```

## Validation Rules

### Name Validation
```go
func ValidateNodeName(name string) error
// - Non-empty
// - 3-100 characters
// Returns: ErrNameRequired, ErrNameTooShort, ErrNameTooLong
```

### SSH Host Validation
```go
func ValidateSSHHost(host string) error
// - Non-empty
// - Valid hostname or IP address format
// Returns: ErrSSHHostRequired, ErrSSHHostInvalid
```

### SSH Port Validation
```go
func ValidateSSHPort(port int) error
// - Range: 1-65535
// Returns: ErrSSHPortInvalid
```

### SSH User Validation
```go
func ValidateSSHUser(user string) error
// - Non-empty
// Returns: ErrSSHUserRequired
```

### Capabilities Validation
```go
func ValidateCapabilities(caps []string) error
// - Non-empty (at least one capability)
// - Each capability is non-empty string
// Returns: ErrCapabilitiesRequired, ErrCapabilityEmpty
```

### Connection Test
```go
func TestNodeConnection(node Node, sshKey SSHKey) error
// - Connect via SSH
// - Run: docker info
// - Parse Docker version and resources
// Returns: ErrSSHConnectionFailed, ErrDockerNotFound, ErrDockerNotRunning
```

## Not Supported

1. **Agent-based connection**: Only SSH tunnel supported
   - *Reason*: Simpler setup for creators, no agent installation needed
   - *Future*: May add lightweight agent for better security

2. **Shared nodes between creators**: Nodes belong to single creator
   - *Reason*: Clear ownership and billing
   - *Future*: May add node pool sharing

3. **Auto-discovery of nodes**: Nodes must be manually registered
   - *Reason*: Security - explicit registration required

4. **Kubernetes/Swarm nodes**: Only plain Docker daemon supported
   - *Reason*: ADR-001 - Docker Direct architecture

5. **Password-based SSH**: Only key-based authentication
   - *Reason*: Security best practice

## JSON:API Resource

### Resource Type
```
nodes
```

### Resource Structure
```json
{
  "data": {
    "type": "nodes",
    "id": "node_abc123",
    "attributes": {
      "name": "Production Server 1",
      "ssh_host": "192.168.1.100",
      "ssh_port": 22,
      "ssh_user": "deploy",
      "docker_socket": "/var/run/docker.sock",
      "status": "online",
      "capabilities": ["standard", "ssd"],
      "capacity": {
        "cpu_cores": 8,
        "memory_mb": 16384,
        "disk_mb": 102400,
        "cpu_used": 2.5,
        "memory_used_mb": 8192,
        "disk_used_mb": 51200
      },
      "location": "us-east-1",
      "last_health_check": "2024-01-15T12:00:00Z",
      "error_message": null,
      "created_at": "2024-01-10T09:00:00Z",
      "updated_at": "2024-01-15T12:00:00Z"
    },
    "relationships": {
      "creator": {
        "data": { "type": "users", "id": "user_xyz789" }
      },
      "deployments": {
        "links": { "related": "/api/v1/nodes/node_abc123/deployments" }
      }
    }
  }
}
```

### API Actions

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/nodes` | List creator's nodes |
| POST | `/api/v1/nodes` | Register new node |
| GET | `/api/v1/nodes/:id` | Get node details |
| PATCH | `/api/v1/nodes/:id` | Update node |
| DELETE | `/api/v1/nodes/:id` | Remove node |
| POST | `/api/v1/nodes/:id/test` | Test SSH connection |
| POST | `/api/v1/nodes/:id/health` | Run health check |
| POST | `/api/v1/nodes/:id/maintenance` | Toggle maintenance mode |

## Security Considerations

1. **SSH Key Storage**: Keys encrypted with AES-256-GCM using platform secret
2. **Key Rotation**: Support key updates without downtime
3. **Access Control**: Only node creator can view/modify node
4. **SSH Key Never Exposed**: API never returns private key material
5. **Connection Isolation**: Each node gets separate SSH connection
6. **Host Key Verification**: Store and verify node host keys on first connect

## Recommended Node Setup

Creators should configure their VPS nodes as follows:

```bash
# 1. Create deploy user
sudo useradd -m -s /bin/bash deploy
sudo usermod -aG docker deploy

# 2. Set up SSH key authentication
sudo mkdir -p /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh
echo "PUBLIC_KEY_HERE" | sudo tee /home/deploy/.ssh/authorized_keys
sudo chmod 600 /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh

# 3. Disable password authentication (recommended)
# Edit /etc/ssh/sshd_config: PasswordAuthentication no
sudo systemctl restart sshd

# 4. Optional: Restrict SSH to Hoster platform IP
sudo ufw allow from HOSTER_IP to any port 22
```

## Tests

Test files following STC methodology:

- `internal/core/domain/node_test.go` - Node validation tests
- `internal/core/scheduler/scheduler_test.go` - Node selection tests
- `internal/shell/docker/ssh_client_test.go` - SSH Docker client tests
- `internal/shell/store/sqlite_node_test.go` - Node store tests
- `internal/shell/api/resources/node_test.go` - API resource tests
- `tests/e2e/node_test.go` - E2E node lifecycle tests
