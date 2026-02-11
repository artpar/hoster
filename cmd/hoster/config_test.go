package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Config Loading Tests
// =============================================================================

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Clear environment
	clearEnv(t)

	cfg, err := LoadConfig("")
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.ShutdownTimeout)
	assert.Equal(t, "data/hoster.db", cfg.Database.DSN)
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
}

func TestLoadConfig_FromFile(t *testing.T) {
	clearEnv(t)

	// Create temp config file
	configContent := `
server:
  host: "127.0.0.1"
  port: 9000
  read_timeout: 60s
  write_timeout: 60s
  shutdown_timeout: 15s

database:
  dsn: "/tmp/test.db"

log:
  level: "debug"
  format: "text"
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(configContent), 0644))

	cfg, err := LoadConfig(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, 60*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 60*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.ShutdownTimeout)
	assert.Equal(t, "/tmp/test.db", cfg.Database.DSN)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
}

func TestLoadConfig_EnvironmentOverride(t *testing.T) {
	clearEnv(t)

	// Set environment variables
	t.Setenv("HOSTER_SERVER_HOST", "192.168.1.1")
	t.Setenv("HOSTER_SERVER_PORT", "3000")
	t.Setenv("HOSTER_DATABASE_DSN", "/custom/path.db")
	t.Setenv("HOSTER_LOG_LEVEL", "warn")
	t.Setenv("HOSTER_LOG_FORMAT", "text")

	cfg, err := LoadConfig("")
	require.NoError(t, err)

	assert.Equal(t, "192.168.1.1", cfg.Server.Host)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, "/custom/path.db", cfg.Database.DSN)
	assert.Equal(t, "warn", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
}

func TestLoadConfig_DataDirDerivesDSN(t *testing.T) {
	clearEnv(t)

	t.Setenv("HOSTER_DATA_DIR", "/var/lib/hoster")

	cfg, err := LoadConfig("")
	require.NoError(t, err)

	assert.Equal(t, "/var/lib/hoster/hoster.db", cfg.Database.DSN)
	assert.Equal(t, "/var/lib/hoster/configs", cfg.Domain.ConfigDir)
}

func TestLoadConfig_ExplicitDSNOverridesDataDir(t *testing.T) {
	clearEnv(t)

	t.Setenv("HOSTER_DATA_DIR", "/var/lib/hoster")
	t.Setenv("HOSTER_DATABASE_DSN", "/custom/path.db")

	cfg, err := LoadConfig("")
	require.NoError(t, err)

	assert.Equal(t, "/custom/path.db", cfg.Database.DSN)
}

func TestLoadConfig_FileNotFound_UsesDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := LoadConfig("/nonexistent/path/config.yaml")
	require.NoError(t, err) // Should not error, just use defaults

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	clearEnv(t)

	// Create invalid config file
	tmpFile := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("invalid: yaml: content: [[["), 0644))

	_, err := LoadConfig(tmpFile)
	assert.Error(t, err)
}

// =============================================================================
// Logger Setup Tests
// =============================================================================

func TestSetupLogger_JSONFormat(t *testing.T) {
	cfg := &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	logger := SetupLogger(cfg)
	assert.NotNil(t, logger)
	// Can't easily test JSON format, but at least ensure it's created
}

func TestSetupLogger_TextFormat(t *testing.T) {
	cfg := &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}

	logger := SetupLogger(cfg)
	assert.NotNil(t, logger)
}

func TestSetupLogger_InvalidLevel(t *testing.T) {
	cfg := &Config{
		Log: LogConfig{
			Level:  "invalid",
			Format: "json",
		},
	}

	// Should fall back to info level, not panic
	logger := SetupLogger(cfg)
	assert.NotNil(t, logger)
}

func TestSetupLogger_DebugLevel(t *testing.T) {
	cfg := &Config{
		Log: LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	logger := SetupLogger(cfg)
	assert.NotNil(t, logger)
}

func TestSetupLogger_WarnLevel(t *testing.T) {
	cfg := &Config{
		Log: LogConfig{
			Level:  "warn",
			Format: "json",
		},
	}

	logger := SetupLogger(cfg)
	assert.NotNil(t, logger)
}

func TestSetupLogger_ErrorLevel(t *testing.T) {
	cfg := &Config{
		Log: LogConfig{
			Level:  "error",
			Format: "json",
		},
	}

	logger := SetupLogger(cfg)
	assert.NotNil(t, logger)
}

// =============================================================================
// Config Validation Tests
// =============================================================================

func TestConfig_Address(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	assert.Equal(t, "localhost:8080", cfg.Server.Address())
}

// =============================================================================
// Test Helpers
// =============================================================================

func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"HOSTER_SERVER_HOST",
		"HOSTER_SERVER_PORT",
		"HOSTER_DATABASE_DSN",
		"HOSTER_DATA_DIR",
		"HOSTER_LOG_LEVEL",
		"HOSTER_LOG_FORMAT",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
