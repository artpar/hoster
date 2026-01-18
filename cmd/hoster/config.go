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
