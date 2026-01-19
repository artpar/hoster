package workers

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Configuration
// =============================================================================

func TestDefaultHealthCheckerConfig(t *testing.T) {
	config := DefaultHealthCheckerConfig()

	assert.Equal(t, 60*time.Second, config.Interval)
	assert.Equal(t, 10*time.Second, config.NodeTimeout)
	assert.Equal(t, 5, config.MaxConcurrent)
}

func TestNewHealthChecker_DefaultConfig(t *testing.T) {
	s := &mockStore{}
	hc := NewHealthChecker(s, nil, nil, HealthCheckerConfig{}, nil)

	assert.NotNil(t, hc)
	assert.Equal(t, 60*time.Second, hc.config.Interval)
	assert.Equal(t, 10*time.Second, hc.config.NodeTimeout)
	assert.Equal(t, 5, hc.config.MaxConcurrent)
}

func TestNewHealthChecker_CustomConfig(t *testing.T) {
	s := &mockStore{}
	config := HealthCheckerConfig{
		Interval:      30 * time.Second,
		NodeTimeout:   5 * time.Second,
		MaxConcurrent: 10,
	}
	hc := NewHealthChecker(s, nil, nil, config, slog.Default())

	assert.NotNil(t, hc)
	assert.Equal(t, 30*time.Second, hc.config.Interval)
	assert.Equal(t, 5*time.Second, hc.config.NodeTimeout)
	assert.Equal(t, 10, hc.config.MaxConcurrent)
}

// =============================================================================
// Test Lifecycle
// =============================================================================

func TestHealthChecker_StartStop(t *testing.T) {
	s := &mockStore{
		nodes: []domain.Node{}, // No nodes - quick cycle
	}

	hc := NewHealthChecker(s, nil, nil, HealthCheckerConfig{
		Interval: 100 * time.Millisecond,
	}, slog.Default())

	// Start should not block
	hc.Start()

	// Give it a moment to run
	time.Sleep(50 * time.Millisecond)

	// Stop should not block
	hc.Stop()

	// Should be able to start again
	hc.Start()
	hc.Stop()
}

func TestHealthChecker_StopWithoutStart(t *testing.T) {
	s := &mockStore{}
	hc := NewHealthChecker(s, nil, nil, HealthCheckerConfig{}, nil)

	// Stop without start should not panic
	hc.Stop()
}

// =============================================================================
// Test Run Cycle
// =============================================================================

func TestHealthChecker_RunCycle_NoNodes(t *testing.T) {
	s := &mockStore{
		nodes: []domain.Node{},
	}

	hc := NewHealthChecker(s, nil, nil, HealthCheckerConfig{
		Interval: time.Second,
	}, slog.Default())

	// Manually run a cycle
	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	defer hc.cancel()

	hc.runCycle()

	// Should complete without error
	assert.True(t, s.listCheckableNodesCalled)
}

func TestHealthChecker_RunCycle_WithNodes(t *testing.T) {
	node := createTestNode("node-1", "creator-1")
	s := &mockStore{
		nodes:   []domain.Node{node},
		sshKeys: map[string]*domain.SSHKey{},
	}

	hc := NewHealthChecker(s, nil, []byte("test-key-32-bytes-for-aes-256!!"), HealthCheckerConfig{
		Interval:    time.Second,
		NodeTimeout: 100 * time.Millisecond,
	}, slog.Default())

	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	defer hc.cancel()

	hc.runCycle()

	// Node should have been processed (though it will fail without SSH key)
	assert.True(t, s.listCheckableNodesCalled)
	// Update should have been called to mark node offline
	assert.GreaterOrEqual(t, len(s.updatedNodes), 1)
}

func TestHealthChecker_RunCycle_SkipsNodeWithoutSSHKey(t *testing.T) {
	node := createTestNode("node-1", "creator-1")
	node.SSHKeyID = "" // No SSH key

	s := &mockStore{
		nodes:   []domain.Node{node},
		sshKeys: map[string]*domain.SSHKey{},
	}

	hc := NewHealthChecker(s, nil, []byte("test-key-32-bytes-for-aes-256!!"), HealthCheckerConfig{
		Interval:    time.Second,
		NodeTimeout: 100 * time.Millisecond,
	}, slog.Default())

	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	defer hc.cancel()

	hc.runCycle()

	// Node should be marked offline because no SSH key
	require.Len(t, s.updatedNodes, 1)
	assert.Equal(t, domain.NodeStatusOffline, s.updatedNodes[0].Status)
}

// =============================================================================
// Test Check Node Now
// =============================================================================

func TestHealthChecker_CheckNodeNow_MaintenanceMode(t *testing.T) {
	node := createTestNode("node-1", "creator-1")
	node.Status = domain.NodeStatusMaintenance

	s := &mockStore{
		getNodeResult: &node,
	}

	hc := NewHealthChecker(s, nil, nil, HealthCheckerConfig{}, slog.Default())
	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	defer hc.cancel()

	err := hc.CheckNodeNow(context.Background(), "node-1")

	// Should skip maintenance nodes without error
	assert.NoError(t, err)
	assert.Empty(t, s.updatedNodes)
}

func TestHealthChecker_CheckNodeNow_NodeNotFound(t *testing.T) {
	s := &mockStore{
		getNodeErr: store.ErrNotFound,
	}

	hc := NewHealthChecker(s, nil, nil, HealthCheckerConfig{}, slog.Default())
	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	defer hc.cancel()

	err := hc.CheckNodeNow(context.Background(), "nonexistent")

	assert.Error(t, err)
}

// =============================================================================
// Test Concurrent Checks
// =============================================================================

func TestHealthChecker_RunCycle_ConcurrencyLimit(t *testing.T) {
	// Create more nodes than the concurrency limit
	nodes := make([]domain.Node, 10)
	for i := range 10 {
		nodes[i] = createTestNode("node-"+string(rune('0'+i)), "creator-1")
	}

	s := &mockStore{
		nodes:   nodes,
		sshKeys: map[string]*domain.SSHKey{},
		mu:      sync.Mutex{},
	}

	hc := NewHealthChecker(s, nil, []byte("test-key-32-bytes-for-aes-256!!"), HealthCheckerConfig{
		Interval:      time.Second,
		NodeTimeout:   100 * time.Millisecond,
		MaxConcurrent: 3, // Only 3 at a time
	}, slog.Default())

	hc.ctx, hc.cancel = context.WithCancel(context.Background())
	defer hc.cancel()

	hc.runCycle()

	// All nodes should have been checked
	assert.Equal(t, 10, len(s.updatedNodes))
}

// =============================================================================
// Mock Store
// =============================================================================

type mockStore struct {
	store.Store // Embed interface for default implementations

	nodes                    []domain.Node
	sshKeys                  map[string]*domain.SSHKey
	listCheckableNodesCalled bool
	updatedNodes             []domain.Node
	getNodeResult            *domain.Node
	getNodeErr               error
	mu                       sync.Mutex
}

func (m *mockStore) ListCheckableNodes(ctx context.Context) ([]domain.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listCheckableNodesCalled = true
	return m.nodes, nil
}

func (m *mockStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getNodeErr != nil {
		return nil, m.getNodeErr
	}
	if m.getNodeResult != nil {
		return m.getNodeResult, nil
	}
	for i := range m.nodes {
		if m.nodes[i].ID == id {
			return &m.nodes[i], nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) UpdateNode(ctx context.Context, node *domain.Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedNodes = append(m.updatedNodes, *node)
	return nil
}

func (m *mockStore) GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if key, ok := m.sshKeys[id]; ok {
		return key, nil
	}
	return nil, store.ErrNotFound
}

// =============================================================================
// Test Helpers
// =============================================================================

func createTestNode(id, creatorID string) domain.Node {
	now := time.Now()
	return domain.Node{
		ID:           id,
		Name:         "Test Node " + id,
		CreatorID:    creatorID,
		SSHHost:      "192.168.1.100",
		SSHPort:      22,
		SSHUser:      "deploy",
		SSHKeyID:     "sshkey-1",
		DockerSocket: "/var/run/docker.sock",
		Status:       domain.NodeStatusOnline,
		Capabilities: []string{"standard"},
		Capacity:     domain.NodeCapacity{CPUCores: 4, MemoryMB: 8192},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
