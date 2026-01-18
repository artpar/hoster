package docker

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Config File Tests
// =============================================================================

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "nginx.conf", "nginx.conf"},
		{"with spaces", "my config.conf", "my_config.conf"},
		{"with slashes", "/etc/nginx/nginx.conf", "etc_nginx_nginx.conf"},
		{"with special chars", "file:name<>test", "file_name__test"},
		{"leading underscore", "/leading", "leading"},
		{"multiple special", "a:b/c\\d*e?f", "a_b_c_d_e_f"},
		{"empty after sanitize", "///", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteConfigFiles_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	o := &Orchestrator{
		configDir: tmpDir,
	}

	mounts, err := o.writeConfigFiles("depl-123", nil)
	require.NoError(t, err)
	assert.Empty(t, mounts)
}

func TestWriteConfigFiles_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	o := &Orchestrator{
		configDir: tmpDir,
		logger:    nil, // Will cause panic if used, but writeConfigFiles doesn't require logging without files
	}

	// Setup logger to avoid nil panic
	o.logger = setupTestLogger()

	configFiles := []domain.ConfigFile{
		{
			Name:    "nginx.conf",
			Path:    "/etc/nginx/nginx.conf",
			Content: "server { listen 80; }",
			Mode:    "0644",
		},
	}

	mounts, err := o.writeConfigFiles("depl-123", configFiles)
	require.NoError(t, err)

	// Should have one mount
	assert.Len(t, mounts, 1)
	assert.Contains(t, mounts, "/etc/nginx/nginx.conf")

	// Check the file was written
	hostPath := mounts["/etc/nginx/nginx.conf"]
	content, err := os.ReadFile(hostPath)
	require.NoError(t, err)
	assert.Equal(t, "server { listen 80; }", string(content))

	// Check directory structure
	expectedDir := filepath.Join(tmpDir, "depl-123")
	_, err = os.Stat(expectedDir)
	assert.NoError(t, err)
}

func TestWriteConfigFiles_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	o := &Orchestrator{
		configDir: tmpDir,
		logger:    setupTestLogger(),
	}

	configFiles := []domain.ConfigFile{
		{
			Name:    "nginx.conf",
			Path:    "/etc/nginx/nginx.conf",
			Content: "server { listen 80; }",
			Mode:    "0644",
		},
		{
			Name:    "app.conf",
			Path:    "/etc/nginx/conf.d/app.conf",
			Content: "location / { proxy_pass http://app; }",
			Mode:    "0644",
		},
	}

	mounts, err := o.writeConfigFiles("depl-456", configFiles)
	require.NoError(t, err)

	// Should have two mounts
	assert.Len(t, mounts, 2)
	assert.Contains(t, mounts, "/etc/nginx/nginx.conf")
	assert.Contains(t, mounts, "/etc/nginx/conf.d/app.conf")

	// Verify both files exist and have correct content
	for _, cf := range configFiles {
		hostPath := mounts[cf.Path]
		content, err := os.ReadFile(hostPath)
		require.NoError(t, err)
		assert.Equal(t, cf.Content, string(content))
	}
}

func TestWriteConfigFiles_CustomMode(t *testing.T) {
	tmpDir := t.TempDir()
	o := &Orchestrator{
		configDir: tmpDir,
		logger:    setupTestLogger(),
	}

	configFiles := []domain.ConfigFile{
		{
			Name:    "script.sh",
			Path:    "/app/script.sh",
			Content: "#!/bin/bash\necho hello",
			Mode:    "0755", // Executable
		},
	}

	mounts, err := o.writeConfigFiles("depl-789", configFiles)
	require.NoError(t, err)

	hostPath := mounts["/app/script.sh"]
	info, err := os.Stat(hostPath)
	require.NoError(t, err)

	// Check mode (may need to mask with 0777 on some systems)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0755), mode)
}

func TestCleanupConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()
	o := &Orchestrator{
		configDir: tmpDir,
		logger:    setupTestLogger(),
	}

	// Create a config file first
	configFiles := []domain.ConfigFile{
		{
			Name:    "test.conf",
			Path:    "/test.conf",
			Content: "test content",
		},
	}

	_, err := o.writeConfigFiles("depl-cleanup", configFiles)
	require.NoError(t, err)

	// Verify directory exists
	deploymentDir := filepath.Join(tmpDir, "depl-cleanup")
	_, err = os.Stat(deploymentDir)
	require.NoError(t, err)

	// Cleanup
	err = o.CleanupConfigFiles("depl-cleanup")
	require.NoError(t, err)

	// Verify directory is gone
	_, err = os.Stat(deploymentDir)
	assert.True(t, os.IsNotExist(err))
}

func TestCleanupConfigFiles_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	o := &Orchestrator{
		configDir: tmpDir,
		logger:    setupTestLogger(),
	}

	// Should not error when directory doesn't exist
	err := o.CleanupConfigFiles("nonexistent-deployment")
	assert.NoError(t, err)
}

// setupTestLogger creates a logger for tests that discards output
func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
