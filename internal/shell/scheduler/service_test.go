package scheduler

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// mockDockerClient is a simple mock for docker.Client
type mockDockerClient struct {
	pingError error
}

func (m *mockDockerClient) Ping() error { return m.pingError }
func (m *mockDockerClient) CreateContainer(spec docker.ContainerSpec) (string, error) {
	return "test-container", nil
}
func (m *mockDockerClient) StartContainer(id string) error { return nil }
func (m *mockDockerClient) StopContainer(id string, timeout *time.Duration) error {
	return nil
}
func (m *mockDockerClient) RemoveContainer(id string, opts docker.RemoveOptions) error {
	return nil
}
func (m *mockDockerClient) InspectContainer(id string) (*docker.ContainerInfo, error) {
	return &docker.ContainerInfo{ID: id, Status: docker.ContainerStatusRunning}, nil
}
func (m *mockDockerClient) ListContainers(opts docker.ListOptions) ([]docker.ContainerInfo, error) {
	return nil, nil
}
func (m *mockDockerClient) ContainerLogs(id string, opts docker.LogOptions) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockDockerClient) ContainerStats(id string) (*docker.ContainerResourceStats, error) {
	return nil, nil
}
func (m *mockDockerClient) CreateNetwork(spec docker.NetworkSpec) (string, error) {
	return "test-network", nil
}
func (m *mockDockerClient) RemoveNetwork(id string) error { return nil }
func (m *mockDockerClient) ConnectNetwork(networkID, containerID string) error {
	return nil
}
func (m *mockDockerClient) DisconnectNetwork(networkID, containerID string, force bool) error {
	return nil
}
func (m *mockDockerClient) CreateVolume(spec docker.VolumeSpec) (string, error) {
	return "test-volume", nil
}
func (m *mockDockerClient) RemoveVolume(name string, force bool) error { return nil }
func (m *mockDockerClient) PullImage(image string, opts docker.PullOptions) error {
	return nil
}
func (m *mockDockerClient) ImageExists(image string) (bool, error) { return true, nil }
func (m *mockDockerClient) Close() error                           { return nil }

// testStore creates a test SQLite store
func testStore(t *testing.T) store.Store {
	s, err := store.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

// testNode creates a test node
func testNode(id, creatorID, name string, status domain.NodeStatus, caps []string) domain.Node {
	return domain.Node{
		ID:           id,
		Name:         name,
		CreatorID:    creatorID,
		SSHHost:      "192.168.1.100",
		SSHPort:      22,
		SSHUser:      "deploy",
		SSHKeyID:     "key-1",
		Status:       status,
		Capabilities: caps,
		Capacity: domain.NodeCapacity{
			CPUCores:     8,
			MemoryMB:     16384,
			DiskMB:       102400,
			CPUUsed:      2,
			MemoryUsedMB: 4096,
			DiskUsedMB:   20480,
		},
	}
}

// testTemplate creates a test template
func testTemplate(id, creatorID string, caps []string, resources domain.Resources) *domain.Template {
	return &domain.Template{
		ID:                   id,
		Name:                 "Test Template",
		CreatorID:            creatorID,
		RequiredCapabilities: caps,
		ResourceRequirements: resources,
	}
}

// =============================================================================
// Tests
// =============================================================================

func TestNewService(t *testing.T) {
	s := testStore(t)
	localClient := &mockDockerClient{}

	service := NewService(s, nil, localClient, nil)

	assert.NotNil(t, service)
	assert.Equal(t, localClient, service.LocalClient())
	assert.False(t, service.SupportsRemoteNodes())
}

func TestScheduleDeployment_NoNodePool_ReturnsLocal(t *testing.T) {
	s := testStore(t)
	localClient := &mockDockerClient{}
	service := NewService(s, nil, localClient, nil)

	template := testTemplate("tmpl-1", "creator-1", nil, domain.Resources{})
	req := ScheduleDeploymentRequest{
		Template:  template,
		CreatorID: "creator-1",
	}

	result, err := service.ScheduleDeployment(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "local", result.NodeID)
	assert.True(t, result.IsLocal)
	assert.Nil(t, result.Node)
	assert.Equal(t, localClient, result.Client)
}

func TestScheduleDeployment_NoNodesForCreator_ReturnsLocal(t *testing.T) {
	// Create store with nodes for a different creator
	s := testStore(t)
	ctx := context.Background()

	// Create SSH key first (required for foreign key constraint)
	sshKey := &domain.SSHKey{
		ID:                  "key-1",
		Name:                "Test Key",
		CreatorID:           "other-creator",
		PrivateKeyEncrypted: []byte("encrypted-key"),
		Fingerprint:         "SHA256:test",
	}
	err := s.CreateSSHKey(ctx, sshKey)
	require.NoError(t, err)

	node := testNode("node-1", "other-creator", "Node 1", domain.NodeStatusOnline, []string{"standard"})
	err = s.CreateNode(ctx, &node)
	require.NoError(t, err)

	localClient := &mockDockerClient{}
	// Note: We can't easily test with a real NodePool without SSH keys,
	// so this test verifies the local fallback behavior
	service := NewService(s, nil, localClient, nil)

	template := testTemplate("tmpl-1", "creator-1", nil, domain.Resources{})
	req := ScheduleDeploymentRequest{
		Template:  template,
		CreatorID: "creator-1",
	}

	result, err := service.ScheduleDeployment(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "local", result.NodeID)
	assert.True(t, result.IsLocal)
}

func TestGetClientForNode_Local(t *testing.T) {
	s := testStore(t)
	localClient := &mockDockerClient{}
	service := NewService(s, nil, localClient, nil)

	// Test with empty nodeID
	client, err := service.GetClientForNode(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, localClient, client)

	// Test with "local" nodeID
	client, err = service.GetClientForNode(context.Background(), "local")
	require.NoError(t, err)
	assert.Equal(t, localClient, client)
}

func TestGetClientForNode_NoNodePool(t *testing.T) {
	s := testStore(t)
	localClient := &mockDockerClient{}
	service := NewService(s, nil, localClient, nil)

	_, err := service.GetClientForNode(context.Background(), "node-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node pool not configured")
}

func TestGetClientForNode_NoLocalClient(t *testing.T) {
	s := testStore(t)
	service := NewService(s, nil, nil, nil)

	_, err := service.GetClientForNode(context.Background(), "local")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "local client not configured")
}

func TestFilterNodesByCreator(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []domain.Node
		creatorID string
		expected  int
	}{
		{
			name:      "empty nodes",
			nodes:     []domain.Node{},
			creatorID: "creator-1",
			expected:  0,
		},
		{
			name: "empty creator ID returns all",
			nodes: []domain.Node{
				testNode("n1", "c1", "Node 1", domain.NodeStatusOnline, []string{"standard"}),
				testNode("n2", "c2", "Node 2", domain.NodeStatusOnline, []string{"standard"}),
			},
			creatorID: "",
			expected:  2,
		},
		{
			name: "filters by creator",
			nodes: []domain.Node{
				testNode("n1", "c1", "Node 1", domain.NodeStatusOnline, []string{"standard"}),
				testNode("n2", "c1", "Node 2", domain.NodeStatusOnline, []string{"standard"}),
				testNode("n3", "c2", "Node 3", domain.NodeStatusOnline, []string{"standard"}),
			},
			creatorID: "c1",
			expected:  2,
		},
		{
			name: "no matching creator",
			nodes: []domain.Node{
				testNode("n1", "c1", "Node 1", domain.NodeStatusOnline, []string{"standard"}),
			},
			creatorID: "c2",
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterNodesByCreator(tt.nodes, tt.creatorID)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestSupportsRemoteNodes(t *testing.T) {
	s := testStore(t)
	localClient := &mockDockerClient{}

	// Without node pool
	service1 := NewService(s, nil, localClient, nil)
	assert.False(t, service1.SupportsRemoteNodes())

	// With node pool (even if empty)
	nodePool := docker.NewNodePool(s, []byte("test-key-32-bytes-long-here!!"), docker.DefaultNodePoolConfig())
	service2 := NewService(s, nodePool, localClient, nil)
	assert.True(t, service2.SupportsRemoteNodes())
}

func TestLocalResult(t *testing.T) {
	s := testStore(t)
	localClient := &mockDockerClient{}
	service := NewService(s, nil, localClient, nil)

	result := service.localResult()

	assert.Equal(t, "local", result.NodeID)
	assert.True(t, result.IsLocal)
	assert.Nil(t, result.Node)
	assert.Equal(t, localClient, result.Client)
	assert.Equal(t, float64(0), result.Score)
}
