# F005: Entry Point Specification

## Overview

CLI entry point that configures and starts the Hoster server. Handles configuration loading, dependency initialization, and graceful shutdown.

## Dependencies

- F002: SQLite Store (for persistence)
- F003: Docker Client (for container operations)
- F004: HTTP API (for request handling)

## Package

`cmd/hoster/`

## Files

```
cmd/hoster/
├── main.go          # Entry point
├── config.go        # Config loading
├── server.go        # Server struct
└── main_test.go     # ~15 tests
```

---

## Configuration

### Config File (hoster.yaml)

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 30s

database:
  dsn: "./data/hoster.db"

docker:
  host: ""  # Empty uses default from environment

log:
  level: "info"
  format: "json"  # json or text
```

### Environment Variables

Environment variables override config file values:

| Variable | Config Path | Default |
|----------|-------------|---------|
| `HOSTER_SERVER_HOST` | server.host | 0.0.0.0 |
| `HOSTER_SERVER_PORT` | server.port | 8080 |
| `HOSTER_DATABASE_DSN` | database.dsn | ./data/hoster.db |
| `HOSTER_DOCKER_HOST` | docker.host | (from DOCKER_HOST) |
| `HOSTER_LOG_LEVEL` | log.level | info |
| `HOSTER_LOG_FORMAT` | log.format | json |

### Config Precedence

1. Environment variables (highest priority)
2. Config file (hoster.yaml)
3. Default values (lowest priority)

---

## Config Struct

```go
// Config holds all application configuration.
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Docker   DockerConfig   `mapstructure:"docker"`
    Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
    Host            string        `mapstructure:"host"`
    Port            int           `mapstructure:"port"`
    ReadTimeout     time.Duration `mapstructure:"read_timeout"`
    WriteTimeout    time.Duration `mapstructure:"write_timeout"`
    ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
    DSN string `mapstructure:"dsn"`
}

// DockerConfig holds Docker client configuration.
type DockerConfig struct {
    Host string `mapstructure:"host"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
}
```

---

## Server Struct

```go
// Server represents the Hoster application server.
type Server struct {
    config     *Config
    httpServer *http.Server
    store      store.Store
    docker     docker.Client
    logger     *slog.Logger
}

// NewServer creates a new server with the given config.
func NewServer(cfg *Config) (*Server, error)

// Start starts the server and blocks until shutdown.
func (s *Server) Start(ctx context.Context) error

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error
```

---

## Startup Sequence

1. **Parse flags** - Handle -config, -version, -help
2. **Load config** - Load from file and environment
3. **Initialize logger** - Set up slog with configured level/format
4. **Connect database** - Create SQLite store, run migrations
5. **Connect Docker** - Create Docker client, verify connection
6. **Create HTTP handler** - Wire up dependencies
7. **Start HTTP server** - Listen on configured address
8. **Wait for shutdown** - Wait for SIGINT/SIGTERM
9. **Graceful shutdown** - Stop server, close connections

---

## Shutdown Sequence

1. Receive shutdown signal (SIGINT or SIGTERM)
2. Log shutdown initiated
3. Stop accepting new HTTP connections
4. Wait for active requests to complete (up to timeout)
5. Close Docker client
6. Close database connection
7. Log shutdown complete
8. Exit with code 0

---

## CLI Flags

```
Usage: hoster [flags]

Flags:
  -config string    Path to config file (default "hoster.yaml")
  -version          Print version and exit
  -help             Print help and exit
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (normal shutdown) |
| 1 | Configuration error |
| 2 | Database connection failed |
| 3 | Docker connection failed |
| 4 | HTTP server failed to start |

---

## Test Categories (~15 tests)

### Config Loading (5 tests)
1. LoadConfig_DefaultValues
2. LoadConfig_FromFile
3. LoadConfig_EnvironmentOverride
4. LoadConfig_FileNotFound_UsesDefaults
5. LoadConfig_InvalidFile

### Server Lifecycle (5 tests)
1. NewServer_Success
2. NewServer_DatabaseFailed
3. NewServer_DockerFailed
4. Server_GracefulShutdown
5. Server_ShutdownTimeout

### Logger Setup (3 tests)
1. SetupLogger_JSONFormat
2. SetupLogger_TextFormat
3. SetupLogger_InvalidLevel

### Integration (2 tests)
1. Server_FullLifecycle
2. Server_SignalHandling

---

## Error Handling

### Startup Errors

Errors during startup are logged and cause immediate exit:
- Config file parse error → exit 1
- Database connection failed → exit 2
- Docker connection failed → exit 3
- HTTP bind failed → exit 4

### Runtime Errors

Runtime errors are logged but don't cause shutdown:
- Request handling errors → 5xx responses
- Docker operation failures → logged, returned to client

---

## Implementation Notes

1. Use `github.com/spf13/viper` for config loading
2. Use `log/slog` for structured logging
3. Use `context.Context` for cancellation
4. Signal handling via `os/signal`
5. No global state - all dependencies injected
6. All I/O in shell layer

---

## Verification

### Manual Testing

```bash
# Start with defaults
./hoster

# Start with custom config
./hoster -config /path/to/config.yaml

# Override via environment
HOSTER_SERVER_PORT=9000 ./hoster

# Test shutdown (press Ctrl+C or send SIGTERM)
```

### Endpoint Verification

```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready
```
