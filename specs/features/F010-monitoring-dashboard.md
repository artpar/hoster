# F010: Monitoring Dashboard

## Overview

Provide health, logs, stats, and events endpoints for deployment monitoring. Enables customers to monitor their deployments through the frontend UI.

## User Stories

### US-1: As a customer, I want to see if my deployment is healthy

**Acceptance Criteria:**
- Health endpoint returns aggregated health status
- Shows status of each container in deployment
- Indicates if any container is unhealthy or down

### US-2: As a customer, I want to view container logs

**Acceptance Criteria:**
- Logs endpoint returns recent log lines
- Can filter by service/container name
- Supports tail (last N lines) parameter
- Returns logs from all containers by default

### US-3: As a customer, I want to see resource usage stats

**Acceptance Criteria:**
- Stats endpoint returns CPU, memory, network stats
- Stats are per-container within deployment
- Stats are point-in-time (not historical)

### US-4: As a customer, I want to see deployment events

**Acceptance Criteria:**
- Events endpoint returns lifecycle events
- Events include container starts, stops, restarts, errors
- Events are ordered by timestamp (newest first)

## Technical Specification

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/deployments/:id/monitoring/health` | Deployment health status |
| GET | `/api/v1/deployments/:id/monitoring/logs` | Container logs |
| GET | `/api/v1/deployments/:id/monitoring/stats` | Resource statistics |
| GET | `/api/v1/deployments/:id/monitoring/events` | Lifecycle events |

### Health Response

```json
{
  "data": {
    "type": "deployment-health",
    "id": "dep_123",
    "attributes": {
      "status": "healthy",
      "containers": [
        {
          "name": "wordpress",
          "status": "running",
          "health": "healthy",
          "started_at": "2024-01-15T10:30:00Z",
          "restarts": 0
        },
        {
          "name": "mysql",
          "status": "running",
          "health": "healthy",
          "started_at": "2024-01-15T10:29:55Z",
          "restarts": 0
        }
      ],
      "checked_at": "2024-01-15T12:00:00Z"
    }
  }
}
```

### Logs Response

```json
{
  "data": {
    "type": "deployment-logs",
    "id": "dep_123",
    "attributes": {
      "logs": [
        {
          "container": "wordpress",
          "timestamp": "2024-01-15T12:00:01Z",
          "stream": "stdout",
          "message": "Apache/2.4.54 (Debian) PHP/8.1.14 configured"
        },
        {
          "container": "mysql",
          "timestamp": "2024-01-15T12:00:00Z",
          "stream": "stdout",
          "message": "ready for connections"
        }
      ]
    }
  },
  "meta": {
    "container_filter": null,
    "tail": 100,
    "since": null
  }
}
```

Query parameters:
- `tail` (int, default 100): Number of lines to return
- `container` (string, optional): Filter to specific container
- `since` (RFC3339, optional): Return logs since timestamp

### Stats Response

```json
{
  "data": {
    "type": "deployment-stats",
    "id": "dep_123",
    "attributes": {
      "containers": [
        {
          "name": "wordpress",
          "cpu_percent": 2.5,
          "memory_usage_bytes": 134217728,
          "memory_limit_bytes": 536870912,
          "memory_percent": 25.0,
          "network_rx_bytes": 1048576,
          "network_tx_bytes": 524288,
          "block_read_bytes": 10485760,
          "block_write_bytes": 5242880,
          "pids": 12
        },
        {
          "name": "mysql",
          "cpu_percent": 5.1,
          "memory_usage_bytes": 268435456,
          "memory_limit_bytes": 1073741824,
          "memory_percent": 25.0,
          "network_rx_bytes": 2097152,
          "network_tx_bytes": 1048576,
          "block_read_bytes": 52428800,
          "block_write_bytes": 26214400,
          "pids": 28
        }
      ],
      "collected_at": "2024-01-15T12:00:00Z"
    }
  }
}
```

### Events Response

```json
{
  "data": {
    "type": "deployment-events",
    "id": "dep_123",
    "attributes": {
      "events": [
        {
          "id": "evt_1",
          "type": "container_started",
          "container": "wordpress",
          "message": "Container started successfully",
          "timestamp": "2024-01-15T10:30:00Z"
        },
        {
          "id": "evt_2",
          "type": "container_started",
          "container": "mysql",
          "message": "Container started successfully",
          "timestamp": "2024-01-15T10:29:55Z"
        }
      ]
    }
  },
  "meta": {
    "limit": 50,
    "total": 2
  }
}
```

Query parameters:
- `limit` (int, default 50): Max events to return
- `type` (string, optional): Filter by event type

### Domain Types (Pure Core)

```go
// internal/core/domain/monitoring.go

type DeploymentHealth struct {
    Status     HealthStatus      `json:"status"`
    Containers []ContainerHealth `json:"containers"`
    CheckedAt  time.Time         `json:"checked_at"`
}

type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnknown   HealthStatus = "unknown"
)

type ContainerHealth struct {
    Name      string       `json:"name"`
    Status    string       `json:"status"`    // running, stopped, paused, restarting
    Health    HealthStatus `json:"health"`
    StartedAt *time.Time   `json:"started_at,omitempty"`
    Restarts  int          `json:"restarts"`
}

type ContainerStats struct {
    Name             string  `json:"name"`
    CPUPercent       float64 `json:"cpu_percent"`
    MemoryUsageBytes int64   `json:"memory_usage_bytes"`
    MemoryLimitBytes int64   `json:"memory_limit_bytes"`
    MemoryPercent    float64 `json:"memory_percent"`
    NetworkRxBytes   int64   `json:"network_rx_bytes"`
    NetworkTxBytes   int64   `json:"network_tx_bytes"`
    BlockReadBytes   int64   `json:"block_read_bytes"`
    BlockWriteBytes  int64   `json:"block_write_bytes"`
    PIDs             int     `json:"pids"`
}

type ContainerLog struct {
    Container string    `json:"container"`
    Timestamp time.Time `json:"timestamp"`
    Stream    string    `json:"stream"` // stdout, stderr
    Message   string    `json:"message"`
}

type ContainerEvent struct {
    ID        string    `json:"id"`
    Type      EventType `json:"type"`
    Container string    `json:"container"`
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}

type EventType string

const (
    EventContainerCreated   EventType = "container_created"
    EventContainerStarted   EventType = "container_started"
    EventContainerStopped   EventType = "container_stopped"
    EventContainerRestarted EventType = "container_restarted"
    EventContainerDied      EventType = "container_died"
    EventContainerOOM       EventType = "container_oom"
    EventHealthUnhealthy    EventType = "health_unhealthy"
    EventHealthHealthy      EventType = "health_healthy"
)
```

### Health Aggregation (Pure Core)

```go
// internal/core/monitoring/health.go

// AggregateHealth determines overall deployment health from container states
func AggregateHealth(containers []ContainerHealth) HealthStatus {
    if len(containers) == 0 {
        return HealthStatusUnknown
    }

    unhealthy := 0
    degraded := 0

    for _, c := range containers {
        switch c.Health {
        case HealthStatusUnhealthy:
            unhealthy++
        case HealthStatusDegraded:
            degraded++
        }
    }

    if unhealthy == len(containers) {
        return HealthStatusUnhealthy
    }
    if unhealthy > 0 || degraded > 0 {
        return HealthStatusDegraded
    }
    return HealthStatusHealthy
}

// DetermineContainerHealth determines health from container state
func DetermineContainerHealth(status string, healthCheck *string, restarts int) HealthStatus {
    if status != "running" {
        return HealthStatusUnhealthy
    }
    if healthCheck != nil && *healthCheck == "unhealthy" {
        return HealthStatusUnhealthy
    }
    if restarts > 3 {
        return HealthStatusDegraded
    }
    return HealthStatusHealthy
}
```

### Docker Stats Interface (Shell)

```go
// internal/shell/docker/client.go (interface extension)

type Client interface {
    // ... existing methods

    // ContainerStats returns current resource stats for a container
    ContainerStats(ctx context.Context, containerID string) (domain.ContainerStats, error)

    // ContainerLogs returns logs from a container
    ContainerLogs(ctx context.Context, containerID string, opts LogOptions) ([]domain.ContainerLog, error)

    // ContainerInspect returns detailed container info
    ContainerInspect(ctx context.Context, containerID string) (ContainerInfo, error)
}

type LogOptions struct {
    Tail      int
    Since     time.Time
    Container string // Filter to specific container
}
```

### Event Storage (Shell)

```go
// internal/shell/store/store.go (interface extension)

type Store interface {
    // ... existing methods

    // Container events
    CreateContainerEvent(ctx context.Context, deploymentID string, event domain.ContainerEvent) error
    GetContainerEvents(ctx context.Context, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error)
}
```

### Database Migration

```sql
-- internal/shell/store/migrations/005_container_events.up.sql

CREATE TABLE IF NOT EXISTS container_events (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    type TEXT NOT NULL,
    container TEXT NOT NULL,
    message TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE INDEX idx_container_events_deployment_time
    ON container_events(deployment_id, timestamp DESC);

CREATE INDEX idx_container_events_type
    ON container_events(deployment_id, type);
```

## Test Cases

### Unit Tests (internal/core/monitoring/)

```go
// health_test.go
func TestAggregateHealth_AllHealthy(t *testing.T)
func TestAggregateHealth_OneUnhealthy(t *testing.T)
func TestAggregateHealth_AllUnhealthy(t *testing.T)
func TestAggregateHealth_MixedStatus(t *testing.T)
func TestAggregateHealth_EmptyContainers(t *testing.T)
func TestDetermineContainerHealth_Running(t *testing.T)
func TestDetermineContainerHealth_Stopped(t *testing.T)
func TestDetermineContainerHealth_HighRestarts(t *testing.T)
func TestDetermineContainerHealth_UnhealthyCheck(t *testing.T)
```

### Integration Tests (internal/shell/)

```go
// docker/stats_test.go
func TestClient_ContainerStats(t *testing.T)
func TestClient_ContainerLogs(t *testing.T)
func TestClient_ContainerLogs_WithTail(t *testing.T)
func TestClient_ContainerLogs_WithSince(t *testing.T)

// api/monitoring_test.go
func TestHealthEndpoint_HealthyDeployment(t *testing.T)
func TestHealthEndpoint_UnhealthyContainer(t *testing.T)
func TestHealthEndpoint_UnauthorizedUser(t *testing.T)
func TestLogsEndpoint_AllContainers(t *testing.T)
func TestLogsEndpoint_FilterByContainer(t *testing.T)
func TestStatsEndpoint_ReturnsStats(t *testing.T)
func TestEventsEndpoint_ReturnsEvents(t *testing.T)
func TestEventsEndpoint_FilterByType(t *testing.T)
```

## Files to Create

- `internal/core/domain/monitoring.go` - Monitoring types
- `internal/core/monitoring/health.go` - Health aggregation (pure)
- `internal/core/monitoring/health_test.go` - Tests
- `internal/shell/docker/stats.go` - Docker stats/logs implementation
- `internal/shell/docker/stats_test.go` - Tests
- `internal/shell/api/monitoring_handlers.go` - API handlers
- `internal/shell/api/monitoring_handlers_test.go` - Tests
- `internal/shell/store/migrations/005_container_events.up.sql`
- `internal/shell/store/migrations/005_container_events.down.sql`

## Files to Modify

- `internal/shell/docker/client.go` - Add stats/logs interface
- `internal/shell/store/store.go` - Add events interface
- `internal/shell/store/sqlite.go` - Implement events storage
- `internal/shell/api/handler.go` - Register monitoring routes
- Event recording in orchestrator (on container state changes)

## NOT Supported

- Historical metrics (time-series database required)
- Real-time log streaming (WebSocket/SSE required)
- Alerting / notifications
- Custom health checks (beyond Docker health)
- Distributed tracing
- APM integration
- Log aggregation / search
- Metrics retention / cleanup
- Dashboard customization

## Dependencies

- F008: Authentication Integration (authorization for endpoints)
- ADR-003: JSON:API with api2go (response format)
- Existing Docker client implementation

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Docker API latency | Set reasonable timeouts, cache briefly |
| Large log output | Limit tail size, paginate |
| Stats polling overhead | Don't poll automatically, on-demand only |
| Event storage growth | Retention policy (delete old events) |
