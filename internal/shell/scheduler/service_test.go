package scheduler

import (
	"context"
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// testStore creates a test SQLite store
func testStore(t *testing.T) store.Store {
	s, err := store.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

// testNode creates a test node
func testNode(id string, creatorID int, name string, status domain.NodeStatus, caps []string) domain.Node {
	return domain.Node{
		ReferenceID:  id,
		Name:         name,
		CreatorID:    creatorID,
		SSHHost:      "192.168.1.100",
		SSHPort:      22,
		SSHUser:      "deploy",
		SSHKeyID:     1,
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
func testTemplate(id string, creatorID int, caps []string, resources domain.Resources) *domain.Template {
	return &domain.Template{
		ReferenceID:          id,
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

	service := NewService(s, nil, nil)

	assert.NotNil(t, service)
	assert.False(t, service.SupportsRemoteNodes())
}

func TestScheduleDeployment_NoNodePool_ReturnsError(t *testing.T) {
	s := testStore(t)
	service := NewService(s, nil, nil)

	template := testTemplate("tmpl-1", 1, nil, domain.Resources{})
	req := ScheduleDeploymentRequest{
		Template:  template,
		CreatorID: 1,
	}

	result, err := service.ScheduleDeployment(context.Background(), req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNoNodesConfigured)
	assert.Nil(t, result)
}

func TestScheduleDeployment_NoNodesForCreator_ReturnsError(t *testing.T) {
	// Create store with nodes for a different creator
	s := testStore(t)
	ctx := context.Background()

	// Create users first (required for foreign key constraints)
	_, err := s.ResolveUser(ctx, "creator-1", "", "", "free")
	require.NoError(t, err)
	otherCreatorID, err := s.ResolveUser(ctx, "other-creator", "", "", "free")
	require.NoError(t, err)

	// Create SSH key first (required for foreign key constraint)
	sshKey := &domain.SSHKey{
		ReferenceID:         "key-1",
		Name:                "Test Key",
		CreatorID:           otherCreatorID,
		PrivateKeyEncrypted: []byte("encrypted-key"),
		Fingerprint:         "SHA256:test",
	}
	err = s.CreateSSHKey(ctx, sshKey)
	require.NoError(t, err)

	node := testNode("node-1", otherCreatorID, "Node 1", domain.NodeStatusOnline, []string{"standard"})
	node.SSHKeyID = sshKey.ID
	err = s.CreateNode(ctx, &node)
	require.NoError(t, err)

	// NodePool required but no nodes for creator 1
	nodePool := docker.NewNodePool(s, []byte("test-key-32-bytes-long-here!!"), docker.DefaultNodePoolConfig())
	service := NewService(s, nodePool, nil)

	template := testTemplate("tmpl-1", 1, nil, domain.Resources{})
	req := ScheduleDeploymentRequest{
		Template:  template,
		CreatorID: 1,
	}

	result, err := service.ScheduleDeployment(ctx, req)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNoNodesForCreator)
	assert.Nil(t, result)
}

func TestGetClientForNode_EmptyNodeID_ReturnsError(t *testing.T) {
	s := testStore(t)
	service := NewService(s, nil, nil)

	_, err := service.GetClientForNode(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all deployments use remote nodes")
}

func TestGetClientForNode_LocalNodeID_ReturnsError(t *testing.T) {
	s := testStore(t)
	service := NewService(s, nil, nil)

	_, err := service.GetClientForNode(context.Background(), "local")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all deployments use remote nodes")
}

func TestGetClientForNode_NoNodePool(t *testing.T) {
	s := testStore(t)
	service := NewService(s, nil, nil)

	_, err := service.GetClientForNode(context.Background(), "node-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node pool not configured")
}

func TestFilterNodesByCreator(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []domain.Node
		creatorID int
		expected  int
	}{
		{
			name:      "empty nodes",
			nodes:     []domain.Node{},
			creatorID: 1,
			expected:  0,
		},
		{
			name: "zero creator ID returns all",
			nodes: []domain.Node{
				testNode("n1", 1, "Node 1", domain.NodeStatusOnline, []string{"standard"}),
				testNode("n2", 2, "Node 2", domain.NodeStatusOnline, []string{"standard"}),
			},
			creatorID: 0,
			expected:  2,
		},
		{
			name: "filters by creator",
			nodes: []domain.Node{
				testNode("n1", 1, "Node 1", domain.NodeStatusOnline, []string{"standard"}),
				testNode("n2", 1, "Node 2", domain.NodeStatusOnline, []string{"standard"}),
				testNode("n3", 2, "Node 3", domain.NodeStatusOnline, []string{"standard"}),
			},
			creatorID: 1,
			expected:  2,
		},
		{
			name: "no matching creator",
			nodes: []domain.Node{
				testNode("n1", 1, "Node 1", domain.NodeStatusOnline, []string{"standard"}),
			},
			creatorID: 2,
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

	// Without node pool
	service1 := NewService(s, nil, nil)
	assert.False(t, service1.SupportsRemoteNodes())

	// With node pool (even if empty)
	nodePool := docker.NewNodePool(s, []byte("test-key-32-bytes-long-here!!"), docker.DefaultNodePoolConfig())
	service2 := NewService(s, nodePool, nil)
	assert.True(t, service2.SupportsRemoteNodes())
}
