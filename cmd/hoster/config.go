package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// =============================================================================
// Config Types
// =============================================================================

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Docker   DockerConfig   `mapstructure:"docker"`
	Log      LogConfig      `mapstructure:"log"`
	Domain   DomainConfig   `mapstructure:"domain"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Billing  BillingConfig  `mapstructure:"billing"`
	Nodes    NodesConfig    `mapstructure:"nodes"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// Address returns the server address in host:port format.
func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
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

// DomainConfig holds domain generation configuration.
type DomainConfig struct {
	BaseDomain string `mapstructure:"base_domain"`
	ConfigDir  string `mapstructure:"config_dir"` // Base directory for deployment config files
}

// AuthConfig holds authentication configuration.
// Following ADR-005: APIGate Integration for Authentication and Billing
type AuthConfig struct {
	// Mode determines how authentication is handled.
	// "header" - Extract auth from APIGate headers (production)
	// "dev" - Auto-authenticate as dev-user (local development)
	// "none" - Skip auth extraction entirely (unauthenticated requests)
	Mode string `mapstructure:"mode"`

	// RequireAuth determines if authentication is required for protected endpoints.
	// When true, unauthenticated requests to protected endpoints return 401.
	RequireAuth bool `mapstructure:"require_auth"`

	// SharedSecret is an optional secret to validate X-APIGate-Secret header.
	// If empty, secret validation is skipped.
	SharedSecret string `mapstructure:"shared_secret"`
}

// BillingConfig holds billing/metering configuration.
// Following F009: Billing Integration
type BillingConfig struct {
	// Enabled determines if usage metering is enabled.
	Enabled bool `mapstructure:"enabled"`

	// APIGateURL is the base URL of the APIGate billing API.
	APIGateURL string `mapstructure:"apigate_url"`

	// APIKey is the API key for authenticating with APIGate.
	APIKey string `mapstructure:"api_key"`

	// ReportInterval is how often to batch and report usage events.
	ReportInterval time.Duration `mapstructure:"report_interval"`

	// BatchSize is the maximum number of events to report in a single batch.
	BatchSize int `mapstructure:"batch_size"`
}

// NodesConfig holds worker nodes configuration.
// Following Creator Worker Nodes Phase 7: Health Checker
type NodesConfig struct {
	// Enabled determines if remote worker nodes are enabled.
	// When false, only local Docker is used.
	Enabled bool `mapstructure:"enabled"`

	// EncryptionKey is the 32-byte key for encrypting SSH private keys.
	// Must be exactly 32 bytes for AES-256-GCM.
	// Set via HOSTER_NODES_ENCRYPTION_KEY environment variable.
	EncryptionKey string `mapstructure:"encryption_key"`

	// HealthCheckInterval is how often to check node health.
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`

	// HealthCheckTimeout is the timeout for checking a single node.
	HealthCheckTimeout time.Duration `mapstructure:"health_check_timeout"`

	// HealthCheckMaxConcurrent is the max number of concurrent health checks.
	HealthCheckMaxConcurrent int `mapstructure:"health_check_max_concurrent"`
}

// =============================================================================
// Config Loading
// =============================================================================

// LoadConfig loads configuration from file and environment.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "30s")
	v.SetDefault("database.dsn", "./data/hoster.db")
	v.SetDefault("docker.host", "")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("domain.base_domain", "apps.localhost")
	v.SetDefault("domain.config_dir", "./data/configs")
	v.SetDefault("auth.mode", "dev")           // Default to dev user for development
	v.SetDefault("auth.require_auth", false)   // Don't require auth by default
	v.SetDefault("auth.shared_secret", "")     // No secret validation by default

	// Billing defaults (F009: Billing Integration)
	v.SetDefault("billing.enabled", false)            // Disabled by default for development
	v.SetDefault("billing.apigate_url", "http://localhost:8080")
	v.SetDefault("billing.api_key", "")
	v.SetDefault("billing.report_interval", "60s")
	v.SetDefault("billing.batch_size", 100)

	// Node defaults (Creator Worker Nodes)
	v.SetDefault("nodes.enabled", false)                    // Disabled by default (local Docker only)
	v.SetDefault("nodes.encryption_key", "")                // Must be set via environment
	v.SetDefault("nodes.health_check_interval", "60s")      // Check nodes every minute
	v.SetDefault("nodes.health_check_timeout", "10s")       // 10 second timeout per node
	v.SetDefault("nodes.health_check_max_concurrent", 5)    // Max 5 concurrent checks

	// Load from file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			// Only return error if file was explicitly specified and is invalid
			if _, ok := err.(viper.ConfigParseError); ok {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
			// File not found is OK, we'll use defaults
		}
	}

	// Enable environment variable overrides
	v.SetEnvPrefix("HOSTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// =============================================================================
// Logger Setup
// =============================================================================

// SetupLogger creates a logger with the configured level and format.
func SetupLogger(cfg *Config) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Log.Format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
