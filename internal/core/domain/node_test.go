package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Node Status Tests
// =============================================================================

func TestNodeStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   bool
	}{
		{"online is valid", NodeStatusOnline, true},
		{"offline is valid", NodeStatusOffline, true},
		{"maintenance is valid", NodeStatusMaintenance, true},
		{"empty is invalid", NodeStatus(""), false},
		{"random is invalid", NodeStatus("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsValid())
		})
	}
}

func TestNodeStatus_IsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   bool
	}{
		{"online is available", NodeStatusOnline, true},
		{"offline is not available", NodeStatusOffline, false},
		{"maintenance is not available", NodeStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsAvailable())
		})
	}
}

// =============================================================================
// Node Capacity Tests
// =============================================================================

func TestNodeCapacity_AvailableCPU(t *testing.T) {
	tests := []struct {
		name     string
		capacity NodeCapacity
		want     float64
	}{
		{"full capacity", NodeCapacity{CPUCores: 8, CPUUsed: 0}, 8},
		{"partial usage", NodeCapacity{CPUCores: 8, CPUUsed: 3}, 5},
		{"full usage", NodeCapacity{CPUCores: 8, CPUUsed: 8}, 0},
		{"over usage returns zero", NodeCapacity{CPUCores: 8, CPUUsed: 10}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.capacity.AvailableCPU())
		})
	}
}

func TestNodeCapacity_AvailableMemory(t *testing.T) {
	tests := []struct {
		name     string
		capacity NodeCapacity
		want     int64
	}{
		{"full capacity", NodeCapacity{MemoryMB: 16384, MemoryUsedMB: 0}, 16384},
		{"partial usage", NodeCapacity{MemoryMB: 16384, MemoryUsedMB: 8192}, 8192},
		{"full usage", NodeCapacity{MemoryMB: 16384, MemoryUsedMB: 16384}, 0},
		{"over usage returns zero", NodeCapacity{MemoryMB: 16384, MemoryUsedMB: 20000}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.capacity.AvailableMemory())
		})
	}
}

func TestNodeCapacity_AvailableDisk(t *testing.T) {
	tests := []struct {
		name     string
		capacity NodeCapacity
		want     int64
	}{
		{"full capacity", NodeCapacity{DiskMB: 102400, DiskUsedMB: 0}, 102400},
		{"partial usage", NodeCapacity{DiskMB: 102400, DiskUsedMB: 51200}, 51200},
		{"full usage", NodeCapacity{DiskMB: 102400, DiskUsedMB: 102400}, 0},
		{"over usage returns zero", NodeCapacity{DiskMB: 102400, DiskUsedMB: 110000}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.capacity.AvailableDisk())
		})
	}
}

func TestNodeCapacity_CanHandle(t *testing.T) {
	capacity := NodeCapacity{
		CPUCores:     8,
		MemoryMB:     16384,
		DiskMB:       102400,
		CPUUsed:      2,
		MemoryUsedMB: 8192,
		DiskUsedMB:   51200,
	}
	// Available: 6 CPU, 8192 MB RAM, 51200 MB disk

	tests := []struct {
		name     string
		required Resources
		want     bool
	}{
		{
			"can handle small workload",
			Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 1024},
			true,
		},
		{
			"can handle exact available",
			Resources{CPUCores: 6, MemoryMB: 8192, DiskMB: 51200},
			true,
		},
		{
			"cannot handle CPU overload",
			Resources{CPUCores: 7, MemoryMB: 1024, DiskMB: 1024},
			false,
		},
		{
			"cannot handle memory overload",
			Resources{CPUCores: 1, MemoryMB: 10000, DiskMB: 1024},
			false,
		},
		{
			"cannot handle disk overload",
			Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 60000},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, capacity.CanHandle(tt.required))
		})
	}
}

func TestNodeCapacity_UsagePercent(t *testing.T) {
	tests := []struct {
		name     string
		capacity NodeCapacity
		wantMin  float64
		wantMax  float64
	}{
		{
			"zero capacity returns zero",
			NodeCapacity{},
			0, 0,
		},
		{
			"zero usage",
			NodeCapacity{CPUCores: 8, MemoryMB: 16384, DiskMB: 102400},
			0, 0,
		},
		{
			"50% usage",
			NodeCapacity{
				CPUCores: 8, MemoryMB: 16384, DiskMB: 102400,
				CPUUsed: 4, MemoryUsedMB: 8192, DiskUsedMB: 51200,
			},
			49, 51, // ~50%
		},
		{
			"100% usage",
			NodeCapacity{
				CPUCores: 8, MemoryMB: 16384, DiskMB: 102400,
				CPUUsed: 8, MemoryUsedMB: 16384, DiskUsedMB: 102400,
			},
			99, 101, // ~100%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.capacity.UsagePercent()
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax)
		})
	}
}

// =============================================================================
// Node Validation Tests
// =============================================================================

func TestValidateNodeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid name", "Production Server 1", nil},
		{"valid short name", "abc", nil},
		{"empty is invalid", "", ErrNodeNameRequired},
		{"too short", "ab", ErrNodeNameTooShort},
		{"100 chars is valid", string(make([]byte, 100)), nil},
		{"101 chars is too long", string(make([]byte, 101)), ErrNodeNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNodeName(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSHHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid IP", "192.168.1.100", nil},
		{"valid IPv6", "::1", nil},
		{"valid hostname", "server.example.com", nil},
		{"valid simple hostname", "localhost", nil},
		{"valid hostname with dashes", "my-server-01", nil},
		{"empty is invalid", "", ErrSSHHostRequired},
		{"invalid hostname with underscore", "my_server", ErrSSHHostInvalid},
		{"invalid starts with dash", "-server", ErrSSHHostInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHHost(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSHPort(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr error
	}{
		{"default port 22", 22, nil},
		{"port 1", 1, nil},
		{"port 65535", 65535, nil},
		{"port 0 invalid", 0, ErrSSHPortInvalid},
		{"negative port invalid", -1, ErrSSHPortInvalid},
		{"port 65536 invalid", 65536, ErrSSHPortInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHPort(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSSHUser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid user", "deploy", nil},
		{"valid root", "root", nil},
		{"empty is invalid", "", ErrSSHUserRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSSHUser(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantErr error
	}{
		{"single capability", []string{"standard"}, nil},
		{"multiple capabilities", []string{"standard", "gpu", "ssd"}, nil},
		{"empty slice is invalid", []string{}, ErrCapabilitiesRequired},
		{"nil slice is invalid", nil, ErrCapabilitiesRequired},
		{"empty string capability invalid", []string{""}, ErrCapabilityEmpty},
		{"mixed with empty invalid", []string{"standard", ""}, ErrCapabilityEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCapabilities(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// NewNode Tests
// =============================================================================

func TestNewNode(t *testing.T) {
	t.Run("valid node creation", func(t *testing.T) {
		node, err := NewNode(
			1,
			"Production Server",
			"192.168.1.100",
			"deploy",
			22,
			[]string{"standard", "ssd"},
		)

		require.NoError(t, err)
		assert.NotEmpty(t, node.ReferenceID)
		assert.True(t, len(node.ReferenceID) > 5)
		assert.Equal(t, "Production Server", node.Name)
		assert.Equal(t, 1, node.CreatorID)
		assert.Equal(t, "192.168.1.100", node.SSHHost)
		assert.Equal(t, 22, node.SSHPort)
		assert.Equal(t, "deploy", node.SSHUser)
		assert.Equal(t, "/var/run/docker.sock", node.DockerSocket)
		assert.Equal(t, NodeStatusOffline, node.Status)
		assert.Equal(t, []string{"standard", "ssd"}, node.Capabilities)
		assert.NotZero(t, node.CreatedAt)
		assert.NotZero(t, node.UpdatedAt)
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := NewNode(1, "", "192.168.1.100", "deploy", 22, []string{"standard"})
		assert.ErrorIs(t, err, ErrNodeNameRequired)
	})

	t.Run("invalid host", func(t *testing.T) {
		_, err := NewNode(1, "Server", "", "deploy", 22, []string{"standard"})
		assert.ErrorIs(t, err, ErrSSHHostRequired)
	})

	t.Run("invalid port", func(t *testing.T) {
		_, err := NewNode(1, "Server", "192.168.1.100", "deploy", 0, []string{"standard"})
		assert.ErrorIs(t, err, ErrSSHPortInvalid)
	})

	t.Run("invalid user", func(t *testing.T) {
		_, err := NewNode(1, "Server", "192.168.1.100", "", 22, []string{"standard"})
		assert.ErrorIs(t, err, ErrSSHUserRequired)
	})

	t.Run("no capabilities", func(t *testing.T) {
		_, err := NewNode(1, "Server", "192.168.1.100", "deploy", 22, []string{})
		assert.ErrorIs(t, err, ErrCapabilitiesRequired)
	})

	t.Run("no creator ID", func(t *testing.T) {
		_, err := NewNode(0, "Server", "192.168.1.100", "deploy", 22, []string{"standard"})
		assert.Error(t, err)
	})
}

// =============================================================================
// Node Methods Tests
// =============================================================================

func TestNode_HasCapability(t *testing.T) {
	node := &Node{
		Capabilities: []string{"standard", "gpu", "ssd"},
	}

	assert.True(t, node.HasCapability("standard"))
	assert.True(t, node.HasCapability("gpu"))
	assert.True(t, node.HasCapability("ssd"))
	assert.False(t, node.HasCapability("nvme"))
	assert.False(t, node.HasCapability(""))
}

func TestNode_HasAllCapabilities(t *testing.T) {
	node := &Node{
		Capabilities: []string{"standard", "gpu", "ssd"},
	}

	assert.True(t, node.HasAllCapabilities([]string{}))
	assert.True(t, node.HasAllCapabilities([]string{"standard"}))
	assert.True(t, node.HasAllCapabilities([]string{"standard", "gpu"}))
	assert.True(t, node.HasAllCapabilities([]string{"standard", "gpu", "ssd"}))
	assert.False(t, node.HasAllCapabilities([]string{"nvme"}))
	assert.False(t, node.HasAllCapabilities([]string{"standard", "nvme"}))
}

func TestNode_HasAnyCapability(t *testing.T) {
	node := &Node{
		Capabilities: []string{"standard", "gpu"},
	}

	assert.True(t, node.HasAnyCapability([]string{}))
	assert.True(t, node.HasAnyCapability([]string{"standard"}))
	assert.True(t, node.HasAnyCapability([]string{"gpu", "nvme"}))
	assert.False(t, node.HasAnyCapability([]string{"nvme", "high-memory"}))
}

func TestNode_IsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   bool
	}{
		{"online", NodeStatusOnline, true},
		{"offline", NodeStatusOffline, false},
		{"maintenance", NodeStatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &Node{Status: tt.status}
			assert.Equal(t, tt.want, node.IsAvailable())
		})
	}
}

// =============================================================================
// Standard Capabilities Tests
// =============================================================================

func TestIsStandardCapability(t *testing.T) {
	assert.True(t, IsStandardCapability("standard"))
	assert.True(t, IsStandardCapability("gpu"))
	assert.True(t, IsStandardCapability("high-memory"))
	assert.True(t, IsStandardCapability("high-cpu"))
	assert.True(t, IsStandardCapability("ssd"))
	assert.True(t, IsStandardCapability("nvme"))
	assert.False(t, IsStandardCapability("custom"))
	assert.False(t, IsStandardCapability(""))
}

func TestDefaultCapabilities(t *testing.T) {
	caps := DefaultCapabilities()
	assert.Equal(t, []string{"standard"}, caps)
}

// =============================================================================
// ID Generation Tests
// =============================================================================

func TestGenerateNodeID(t *testing.T) {
	id1 := GenerateNodeID()
	id2 := GenerateNodeID()

	assert.True(t, len(id1) > 5)
	assert.True(t, id1[:5] == "node_")
	assert.NotEqual(t, id1, id2) // IDs should be unique
}

func TestGenerateSSHKeyID(t *testing.T) {
	id1 := GenerateSSHKeyID()
	id2 := GenerateSSHKeyID()

	assert.True(t, len(id1) > 7)
	assert.True(t, id1[:7] == "sshkey_")
	assert.NotEqual(t, id1, id2) // IDs should be unique
}
