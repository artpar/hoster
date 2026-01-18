# Monitoring

## Overview

Monitoring types represent the health, logs, statistics, and events for deployments. These are read-only resources derived from Docker container state and stored events.

## Types

### DeploymentHealth

Aggregated health status for a deployment and its containers.

| Field | Type | Description |
|-------|------|-------------|
| `status` | HealthStatus | Overall deployment health |
| `containers` | []ContainerHealth | Health of each container |
| `checked_at` | timestamp | When health was checked |

### HealthStatus (Enum)

| Value | Description |
|-------|-------------|
| `healthy` | All containers running and healthy |
| `unhealthy` | One or more containers down or unhealthy |
| `degraded` | Running but with issues (high restarts, etc.) |
| `unknown` | Cannot determine health (no containers) |

### ContainerHealth

Health information for a single container.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Service name from compose spec |
| `status` | string | Container status (running, stopped, paused, restarting) |
| `health` | HealthStatus | Container health status |
| `started_at` | timestamp | When container started (null if not running) |
| `restarts` | int | Number of restarts since creation |

### ContainerStats

Point-in-time resource usage statistics for a container.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Service name from compose spec |
| `cpu_percent` | float64 | CPU usage percentage |
| `memory_usage_bytes` | int64 | Current memory usage in bytes |
| `memory_limit_bytes` | int64 | Memory limit in bytes |
| `memory_percent` | float64 | Memory usage as percentage of limit |
| `network_rx_bytes` | int64 | Network bytes received |
| `network_tx_bytes` | int64 | Network bytes transmitted |
| `block_read_bytes` | int64 | Disk bytes read |
| `block_write_bytes` | int64 | Disk bytes written |
| `pids` | int | Number of processes |

### ContainerLog

A single log line from a container.

| Field | Type | Description |
|-------|------|-------------|
| `container` | string | Service name from compose spec |
| `timestamp` | timestamp | When log was generated |
| `stream` | string | Output stream (`stdout` or `stderr`) |
| `message` | string | Log message content |

### ContainerEvent

A lifecycle event for a container.

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Unique event identifier |
| `type` | EventType | Type of event |
| `container` | string | Service name from compose spec |
| `message` | string | Human-readable event description |
| `timestamp` | timestamp | When event occurred |

### EventType (Enum)

| Value | Description |
|-------|-------------|
| `container_created` | Container was created |
| `container_started` | Container started running |
| `container_stopped` | Container stopped (graceful) |
| `container_restarted` | Container was restarted |
| `container_died` | Container died unexpectedly |
| `container_oom` | Container killed due to out-of-memory |
| `health_unhealthy` | Health check started failing |
| `health_healthy` | Health check started passing |

## Behaviors

### Health Aggregation

Overall deployment health is determined by aggregating container health:

```go
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
```

### Container Health Determination

Container health is determined from Docker state:

```go
func DetermineContainerHealth(status string, healthCheck *string, restarts int) HealthStatus {
    // Not running = unhealthy
    if status != "running" {
        return HealthStatusUnhealthy
    }

    // Health check failing = unhealthy
    if healthCheck != nil && *healthCheck == "unhealthy" {
        return HealthStatusUnhealthy
    }

    // High restart count = degraded
    if restarts > 3 {
        return HealthStatusDegraded
    }

    return HealthStatusHealthy
}
```

### Log Retrieval

Logs are retrieved from Docker with options:
- `tail`: Number of lines to return (default 100)
- `since`: Only logs after this timestamp
- `container`: Filter to specific service name

### Event Recording

Events are recorded by the orchestrator during deployment lifecycle:
- Container creation → `container_created`
- Container start → `container_started`
- Container stop → `container_stopped`
- Container restart → `container_restarted`
- Container crash → `container_died`
- OOM kill → `container_oom`
- Health check transitions → `health_healthy` / `health_unhealthy`

## JSON:API Resource Definitions

Monitoring data is exposed as sub-resources of deployments.

### Health Endpoint

```
GET /api/v1/deployments/:id/monitoring/health
```

```json
{
  "data": {
    "type": "deployment-health",
    "id": "dep_abc123",
    "attributes": {
      "status": "healthy",
      "containers": [
        {
          "name": "wordpress",
          "status": "running",
          "health": "healthy",
          "started_at": "2024-01-15T10:30:00Z",
          "restarts": 0
        }
      ],
      "checked_at": "2024-01-15T12:00:00Z"
    }
  }
}
```

### Stats Endpoint

```
GET /api/v1/deployments/:id/monitoring/stats
```

```json
{
  "data": {
    "type": "deployment-stats",
    "id": "dep_abc123",
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
        }
      ],
      "collected_at": "2024-01-15T12:00:00Z"
    }
  }
}
```

### Logs Endpoint

```
GET /api/v1/deployments/:id/monitoring/logs?tail=100&container=wordpress
```

```json
{
  "data": {
    "type": "deployment-logs",
    "id": "dep_abc123",
    "attributes": {
      "logs": [
        {
          "container": "wordpress",
          "timestamp": "2024-01-15T12:00:01Z",
          "stream": "stdout",
          "message": "Apache/2.4.54 configured"
        }
      ]
    }
  },
  "meta": {
    "container_filter": "wordpress",
    "tail": 100
  }
}
```

### Events Endpoint

```
GET /api/v1/deployments/:id/monitoring/events?limit=50&type=container_started
```

```json
{
  "data": {
    "type": "deployment-events",
    "id": "dep_abc123",
    "attributes": {
      "events": [
        {
          "id": "evt_1",
          "type": "container_started",
          "container": "wordpress",
          "message": "Container started successfully",
          "timestamp": "2024-01-15T10:30:00Z"
        }
      ]
    }
  },
  "meta": {
    "limit": 50,
    "total": 1,
    "type_filter": "container_started"
  }
}
```

## Not Supported

1. **Historical metrics**: No time-series storage
   - *Reason*: Would require InfluxDB/Prometheus
   - *Future*: May add metrics export to external systems

2. **Real-time log streaming**: Polling only
   - *Reason*: WebSocket adds complexity
   - *Future*: May add SSE or WebSocket streaming

3. **Alerting**: No notification system
   - *Reason*: Prototype simplicity
   - *Future*: May add webhook alerts

4. **Distributed tracing**: No request tracing
   - *Reason*: Prototype simplicity
   - *Future*: May integrate with Jaeger/Zipkin

5. **Custom health checks**: Docker health only
   - *Reason*: Prototype simplicity
   - *Future*: May add HTTP health probes

## Tests

- `internal/core/monitoring/health_test.go` - Health aggregation tests
- `internal/shell/docker/stats_test.go` - Docker stats integration tests
- `internal/shell/api/monitoring_handlers_test.go` - API handler tests
