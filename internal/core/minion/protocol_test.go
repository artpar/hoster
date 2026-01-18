package minion

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Response Tests
// =============================================================================

func TestNewSuccessResponse_WithData(t *testing.T) {
	data := CreateResult{ID: "container123"}

	resp, err := NewSuccessResponse(data)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)

	// Verify data can be unmarshaled
	var result CreateResult
	err = resp.UnmarshalData(&result)
	require.NoError(t, err)
	assert.Equal(t, "container123", result.ID)
}

func TestNewSuccessResponse_WithNilData(t *testing.T) {
	resp, err := NewSuccessResponse(nil)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)
	assert.Nil(t, resp.Data)
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse("start-container", ErrCodeNotFound, "container not found: abc123")

	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "start-container", resp.Error.Command)
	assert.Equal(t, ErrCodeNotFound, resp.Error.Code)
	assert.Equal(t, "container not found: abc123", resp.Error.Message)
}

func TestParseResponse_Success(t *testing.T) {
	jsonData := `{"success":true,"data":{"id":"abc123"}}`

	resp, err := ParseResponse([]byte(jsonData))
	require.NoError(t, err)

	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)

	var result CreateResult
	err = resp.UnmarshalData(&result)
	require.NoError(t, err)
	assert.Equal(t, "abc123", result.ID)
}

func TestParseResponse_Error(t *testing.T) {
	jsonData := `{"success":false,"error":{"command":"create-container","code":"already_exists","message":"container exists"}}`

	resp, err := ParseResponse([]byte(jsonData))
	require.NoError(t, err)

	assert.False(t, resp.Success)
	require.NotNil(t, resp.Error)
	assert.Equal(t, "create-container", resp.Error.Command)
	assert.Equal(t, ErrCodeAlreadyExists, resp.Error.Code)
	assert.Equal(t, "container exists", resp.Error.Message)
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	_, err := ParseResponse([]byte("not json"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestResponse_JSON_RoundTrip(t *testing.T) {
	original := &Response{
		Success: true,
		Data:    json.RawMessage(`{"id":"test123"}`),
	}

	// Marshal
	bytes, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var parsed Response
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, original.Success, parsed.Success)
	assert.Equal(t, string(original.Data), string(parsed.Data))
}

// =============================================================================
// VersionInfo Tests
// =============================================================================

func TestVersionInfo_JSON(t *testing.T) {
	info := VersionInfo{
		Version:   "1.0.0",
		BuildTime: "2024-01-15T10:00:00Z",
		GoVersion: "go1.21",
	}

	bytes, err := json.Marshal(info)
	require.NoError(t, err)

	var parsed VersionInfo
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, info.Version, parsed.Version)
	assert.Equal(t, info.BuildTime, parsed.BuildTime)
	assert.Equal(t, info.GoVersion, parsed.GoVersion)
}

// =============================================================================
// PingInfo Tests
// =============================================================================

func TestPingInfo_JSON(t *testing.T) {
	info := PingInfo{
		DockerVersion: "24.0.7",
		APIVersion:    "1.43",
		OS:            "linux",
		Arch:          "amd64",
	}

	bytes, err := json.Marshal(info)
	require.NoError(t, err)

	var parsed PingInfo
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "24.0.7", parsed.DockerVersion)
	assert.Equal(t, "1.43", parsed.APIVersion)
	assert.Equal(t, "linux", parsed.OS)
	assert.Equal(t, "amd64", parsed.Arch)
}

// =============================================================================
// ContainerSpec Tests
// =============================================================================

func TestContainerSpec_JSON_Full(t *testing.T) {
	spec := ContainerSpec{
		Name:       "test-container",
		Image:      "nginx:alpine",
		Command:    []string{"nginx", "-g", "daemon off;"},
		Entrypoint: []string{"/docker-entrypoint.sh"},
		Env: map[string]string{
			"PORT": "8080",
			"ENV":  "production",
		},
		Labels: map[string]string{
			"com.hoster.deployment": "deploy123",
		},
		Ports: []PortBinding{
			{ContainerPort: 80, HostPort: 8080, Protocol: "tcp"},
		},
		Volumes: []VolumeMount{
			{Source: "data-volume", Target: "/data", ReadOnly: false},
		},
		Networks:   []string{"my-network"},
		WorkingDir: "/app",
		User:       "nginx",
		RestartPolicy: RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 3,
		},
		Resources: ResourceLimits{
			CPULimit:    2.0,
			MemoryLimit: 536870912, // 512MB
		},
		HealthCheck: &HealthCheck{
			Test:        []string{"CMD", "curl", "-f", "http://localhost/"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			Retries:     3,
			StartPeriod: 5 * time.Second,
		},
	}

	bytes, err := json.Marshal(spec)
	require.NoError(t, err)

	var parsed ContainerSpec
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-container", parsed.Name)
	assert.Equal(t, "nginx:alpine", parsed.Image)
	assert.Equal(t, []string{"nginx", "-g", "daemon off;"}, parsed.Command)
	assert.Equal(t, "8080", parsed.Env["PORT"])
	assert.Len(t, parsed.Ports, 1)
	assert.Equal(t, 80, parsed.Ports[0].ContainerPort)
	assert.Len(t, parsed.Volumes, 1)
	assert.Equal(t, "/data", parsed.Volumes[0].Target)
	assert.Equal(t, "on-failure", parsed.RestartPolicy.Name)
	assert.Equal(t, 2.0, parsed.Resources.CPULimit)
	require.NotNil(t, parsed.HealthCheck)
	assert.Equal(t, 30*time.Second, parsed.HealthCheck.Interval)
}

func TestContainerSpec_JSON_Minimal(t *testing.T) {
	spec := ContainerSpec{
		Name:  "minimal",
		Image: "busybox",
	}

	bytes, err := json.Marshal(spec)
	require.NoError(t, err)

	// Should not include omitempty fields
	jsonStr := string(bytes)
	assert.NotContains(t, jsonStr, "command")
	assert.NotContains(t, jsonStr, "entrypoint")

	var parsed ContainerSpec
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "minimal", parsed.Name)
	assert.Equal(t, "busybox", parsed.Image)
	assert.Nil(t, parsed.Command)
	assert.Nil(t, parsed.Env)
}

// =============================================================================
// ContainerInfo Tests
// =============================================================================

func TestContainerInfo_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	startedAt := now.Add(-1 * time.Hour)

	info := ContainerInfo{
		ID:        "abc123def456",
		Name:      "test-container",
		Image:     "nginx:alpine",
		Status:    "running",
		State:     "running",
		Health:    "healthy",
		CreatedAt: now,
		StartedAt: &startedAt,
		Ports: []PortBinding{
			{ContainerPort: 80, HostPort: 8080},
		},
		Labels: map[string]string{
			"com.hoster.deployment": "deploy123",
		},
	}

	bytes, err := json.Marshal(info)
	require.NoError(t, err)

	var parsed ContainerInfo
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "abc123def456", parsed.ID)
	assert.Equal(t, "test-container", parsed.Name)
	assert.Equal(t, "running", parsed.Status)
	assert.Equal(t, "healthy", parsed.Health)
	assert.Equal(t, now, parsed.CreatedAt)
	require.NotNil(t, parsed.StartedAt)
	assert.Equal(t, startedAt, *parsed.StartedAt)
	assert.Len(t, parsed.Ports, 1)
}

// =============================================================================
// ContainerResourceStats Tests
// =============================================================================

func TestContainerResourceStats_JSON(t *testing.T) {
	stats := ContainerResourceStats{
		CPUPercent:       25.5,
		MemoryUsageBytes: 268435456, // 256MB
		MemoryLimitBytes: 536870912, // 512MB
		MemoryPercent:    50.0,
		NetworkRxBytes:   1024000,
		NetworkTxBytes:   512000,
		BlockReadBytes:   2048000,
		BlockWriteBytes:  1024000,
		PIDs:             10,
	}

	bytes, err := json.Marshal(stats)
	require.NoError(t, err)

	var parsed ContainerResourceStats
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, 25.5, parsed.CPUPercent)
	assert.Equal(t, int64(268435456), parsed.MemoryUsageBytes)
	assert.Equal(t, 50.0, parsed.MemoryPercent)
	assert.Equal(t, 10, parsed.PIDs)
}

// =============================================================================
// NetworkSpec Tests
// =============================================================================

func TestNetworkSpec_JSON(t *testing.T) {
	spec := NetworkSpec{
		Name:   "my-network",
		Driver: "bridge",
		Labels: map[string]string{
			"com.hoster.deployment": "deploy123",
		},
	}

	bytes, err := json.Marshal(spec)
	require.NoError(t, err)

	var parsed NetworkSpec
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "my-network", parsed.Name)
	assert.Equal(t, "bridge", parsed.Driver)
	assert.Equal(t, "deploy123", parsed.Labels["com.hoster.deployment"])
}

// =============================================================================
// VolumeSpec Tests
// =============================================================================

func TestVolumeSpec_JSON(t *testing.T) {
	spec := VolumeSpec{
		Name:   "data-volume",
		Driver: "local",
		Labels: map[string]string{
			"purpose": "storage",
		},
	}

	bytes, err := json.Marshal(spec)
	require.NoError(t, err)

	var parsed VolumeSpec
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "data-volume", parsed.Name)
	assert.Equal(t, "local", parsed.Driver)
}

// =============================================================================
// Options Tests
// =============================================================================

func TestListOptions_JSON(t *testing.T) {
	opts := ListOptions{
		All: true,
		Filters: map[string]string{
			"label": "com.hoster.deployment=deploy123",
		},
	}

	bytes, err := json.Marshal(opts)
	require.NoError(t, err)

	var parsed ListOptions
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.True(t, parsed.All)
	assert.Equal(t, "com.hoster.deployment=deploy123", parsed.Filters["label"])
}

func TestRemoveOptions_JSON(t *testing.T) {
	opts := RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}

	bytes, err := json.Marshal(opts)
	require.NoError(t, err)

	var parsed RemoveOptions
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.True(t, parsed.Force)
	assert.True(t, parsed.RemoveVolumes)
}

func TestLogOptions_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	opts := LogOptions{
		Follow:     false,
		Tail:       "100",
		Since:      now.Add(-1 * time.Hour),
		Timestamps: true,
	}

	bytes, err := json.Marshal(opts)
	require.NoError(t, err)

	var parsed LogOptions
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.False(t, parsed.Follow)
	assert.Equal(t, "100", parsed.Tail)
	assert.True(t, parsed.Timestamps)
}

func TestPullOptions_JSON(t *testing.T) {
	opts := PullOptions{
		Platform: "linux/amd64",
	}

	bytes, err := json.Marshal(opts)
	require.NoError(t, err)

	var parsed PullOptions
	err = json.Unmarshal(bytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "linux/amd64", parsed.Platform)
}

// =============================================================================
// Error Codes Tests
// =============================================================================

func TestErrorCodes_Values(t *testing.T) {
	// Verify error codes are distinct strings
	codes := []string{
		ErrCodeNotFound,
		ErrCodeAlreadyExists,
		ErrCodeNotRunning,
		ErrCodeAlreadyRunning,
		ErrCodeInUse,
		ErrCodePortConflict,
		ErrCodeConnectionFailed,
		ErrCodeTimeout,
		ErrCodePullFailed,
		ErrCodeInvalidInput,
		ErrCodeInternal,
	}

	// Check uniqueness
	seen := make(map[string]bool)
	for _, code := range codes {
		assert.False(t, seen[code], "duplicate error code: %s", code)
		seen[code] = true
	}
}
