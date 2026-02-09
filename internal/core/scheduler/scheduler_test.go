package scheduler

import (
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

func makeNode(id, name string, status domain.NodeStatus, caps []string, cpu float64, mem, disk int64) domain.Node {
	return domain.Node{
		ReferenceID:  id,
		Name:         name,
		CreatorID:    1,
		Status:       status,
		Capabilities: caps,
		Capacity: domain.NodeCapacity{
			CPUCores:     cpu,
			MemoryMB:     mem,
			DiskMB:       disk,
			CPUUsed:      0,
			MemoryUsedMB: 0,
			DiskUsedMB:   0,
		},
	}
}

func makeNodeWithUsage(id string, caps []string, cpuTotal, cpuUsed float64, memTotal, memUsed, diskTotal, diskUsed int64) domain.Node {
	return domain.Node{
		ReferenceID:  id,
		Name:         id,
		CreatorID:    1,
		Status:       domain.NodeStatusOnline,
		Capabilities: caps,
		Capacity: domain.NodeCapacity{
			CPUCores:     cpuTotal,
			MemoryMB:     memTotal,
			DiskMB:       diskTotal,
			CPUUsed:      cpuUsed,
			MemoryUsedMB: memUsed,
			DiskUsedMB:   diskUsed,
		},
	}
}

// =============================================================================
// Schedule Tests
// =============================================================================

func TestSchedule_BasicSelection(t *testing.T) {
	nodes := []domain.Node{
		makeNode("node_1", "Node 1", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("node_2", "Node 2", domain.NodeStatusOnline, []string{"standard"}, 8, 16384, 102400),
	}

	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.NotEmpty(t, result.SelectedNodeID)
	// Should prefer the larger node (higher score)
	assert.Equal(t, "node_2", result.SelectedNodeID)
}

func TestSchedule_NoNodes(t *testing.T) {
	req := ScheduleRequest{
		AvailableNodes:    []domain.Node{},
		RequiredResources: domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
	}

	result, err := Schedule(req)
	assert.ErrorIs(t, err, ErrNoNodesAvailable)
	assert.Empty(t, result.SelectedNodeID)
}

func TestSchedule_AllNodesOffline(t *testing.T) {
	nodes := []domain.Node{
		makeNode("node_1", "Node 1", domain.NodeStatusOffline, []string{"standard"}, 4, 8192, 51200),
		makeNode("node_2", "Node 2", domain.NodeStatusMaintenance, []string{"standard"}, 8, 16384, 102400),
	}

	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
	}

	result, err := Schedule(req)
	assert.ErrorIs(t, err, ErrNoNodesAvailable)
	assert.Equal(t, 2, result.FilteredOutReasons["not_online"])
}

func TestSchedule_RequiredCapabilities(t *testing.T) {
	nodes := []domain.Node{
		makeNode("node_1", "Standard Node", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("node_2", "GPU Node", domain.NodeStatusOnline, []string{"standard", "gpu"}, 8, 16384, 102400),
	}

	req := ScheduleRequest{
		AvailableNodes:       nodes,
		RequiredResources:    domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
		RequiredCapabilities: []string{"gpu"},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "node_2", result.SelectedNodeID)
}

func TestSchedule_NoNodesWithRequiredCapabilities(t *testing.T) {
	nodes := []domain.Node{
		makeNode("node_1", "Standard Node", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("node_2", "SSD Node", domain.NodeStatusOnline, []string{"standard", "ssd"}, 8, 16384, 102400),
	}

	req := ScheduleRequest{
		AvailableNodes:       nodes,
		RequiredResources:    domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
		RequiredCapabilities: []string{"gpu"}, // Neither node has GPU
	}

	result, err := Schedule(req)
	assert.ErrorIs(t, err, ErrNoCapableNodes)
	assert.Equal(t, 2, result.FilteredOutReasons["missing_required_capabilities"])
}

func TestSchedule_PlanCapabilities(t *testing.T) {
	nodes := []domain.Node{
		makeNode("node_1", "Standard Node", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("node_2", "GPU Node", domain.NodeStatusOnline, []string{"gpu"}, 8, 16384, 102400),
		makeNode("node_3", "High-Memory Node", domain.NodeStatusOnline, []string{"high-memory"}, 8, 32768, 102400),
	}

	// User's plan only allows "standard" capability
	req := ScheduleRequest{
		AvailableNodes:      nodes,
		RequiredResources:   domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
		AllowedCapabilities: []string{"standard"},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "node_1", result.SelectedNodeID)
	assert.Equal(t, 2, result.FilteredOutReasons["plan_capabilities_mismatch"])
}

func TestSchedule_PlanDoesNotAllowRequiredCapabilities(t *testing.T) {
	nodes := []domain.Node{
		makeNode("node_1", "Standard Node", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("node_2", "GPU Node", domain.NodeStatusOnline, []string{"gpu"}, 8, 16384, 102400),
	}

	// Template requires GPU, but user's plan only allows standard
	req := ScheduleRequest{
		AvailableNodes:       nodes,
		RequiredResources:    domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
		RequiredCapabilities: []string{"gpu"},
		AllowedCapabilities:  []string{"standard"},
	}

	result, err := Schedule(req)
	// GPU node matches required caps, but not allowed by plan
	// Standard node is allowed by plan, but doesn't have GPU
	assert.Error(t, err)
	assert.Empty(t, result.SelectedNodeID)
}

func TestSchedule_InsufficientCapacity(t *testing.T) {
	nodes := []domain.Node{
		makeNodeWithUsage("node_1", []string{"standard"}, 4, 3.5, 8192, 7000, 51200, 50000),
		makeNodeWithUsage("node_2", []string{"standard"}, 8, 7.5, 16384, 15000, 102400, 100000),
	}

	// Require more resources than available on any node
	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 2, MemoryMB: 4096, DiskMB: 10000},
	}

	result, err := Schedule(req)
	assert.ErrorIs(t, err, ErrInsufficientCapacity)
	assert.Equal(t, 2, result.FilteredOutReasons["insufficient_capacity"])
}

func TestSchedule_SelectsLeastLoadedNode(t *testing.T) {
	nodes := []domain.Node{
		makeNodeWithUsage("node_busy", []string{"standard"}, 8, 6, 16384, 12000, 102400, 80000),
		makeNodeWithUsage("node_idle", []string{"standard"}, 8, 0, 16384, 0, 102400, 0),
	}

	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "node_idle", result.SelectedNodeID)
	assert.Greater(t, result.Score, 50.0) // Idle node should have high score
}

func TestSchedule_MixedConditions(t *testing.T) {
	nodes := []domain.Node{
		makeNode("offline_standard", "Offline Standard", domain.NodeStatusOffline, []string{"standard"}, 8, 16384, 102400),
		makeNode("online_wrong_caps", "Online Wrong Caps", domain.NodeStatusOnline, []string{"other"}, 8, 16384, 102400),
		makeNodeWithUsage("online_full", []string{"standard"}, 8, 8, 16384, 16384, 102400, 102400),
		makeNode("online_correct", "Online Correct", domain.NodeStatusOnline, []string{"standard"}, 8, 16384, 102400),
	}

	req := ScheduleRequest{
		AvailableNodes:      nodes,
		RequiredResources:   domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
		AllowedCapabilities: []string{"standard"},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "online_correct", result.SelectedNodeID)
	assert.Equal(t, 4, result.ConsideredCount)
}

// =============================================================================
// ScoreNode Tests
// =============================================================================

func TestScoreNode_FullCapacity(t *testing.T) {
	node := makeNode("node_1", "Test", domain.NodeStatusOnline, []string{"standard"}, 8, 16384, 102400)

	required := domain.Resources{CPUCores: 0, MemoryMB: 0, DiskMB: 0}
	score := ScoreNode(node, required)

	// Full capacity available, should score 100
	assert.Equal(t, 100.0, score)
}

func TestScoreNode_HalfUsed(t *testing.T) {
	node := makeNodeWithUsage("node_1", []string{"standard"}, 8, 4, 16384, 8192, 102400, 51200)

	required := domain.Resources{CPUCores: 0, MemoryMB: 0, DiskMB: 0}
	score := ScoreNode(node, required)

	// 50% used, so 50% available - score should be 50
	assert.InDelta(t, 50.0, score, 0.1)
}

func TestScoreNode_WithRequired(t *testing.T) {
	node := makeNode("node_1", "Test", domain.NodeStatusOnline, []string{"standard"}, 8, 16384, 102400)

	// Require half the resources
	required := domain.Resources{CPUCores: 4, MemoryMB: 8192, DiskMB: 51200}
	score := ScoreNode(node, required)

	// After deployment, 50% will be used, so score ~50
	assert.InDelta(t, 50.0, score, 0.1)
}

func TestScoreNode_ZeroCapacity(t *testing.T) {
	node := domain.Node{
		ReferenceID: "node_1",
		Status:   domain.NodeStatusOnline,
		Capacity: domain.NodeCapacity{}, // All zeros
	}

	required := domain.Resources{CPUCores: 0, MemoryMB: 0, DiskMB: 0}
	score := ScoreNode(node, required)

	// Zero capacity means 0% available
	assert.Equal(t, 0.0, score)
}

// =============================================================================
// Filter Function Tests
// =============================================================================

func TestFilterOnlineNodes(t *testing.T) {
	nodes := []domain.Node{
		makeNode("online_1", "Online 1", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("offline", "Offline", domain.NodeStatusOffline, []string{"standard"}, 4, 8192, 51200),
		makeNode("online_2", "Online 2", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("maintenance", "Maintenance", domain.NodeStatusMaintenance, []string{"standard"}, 4, 8192, 51200),
	}

	result := FilterOnlineNodes(nodes)
	assert.Len(t, result, 2)
	assert.Equal(t, "online_1", result[0].ReferenceID)
	assert.Equal(t, "online_2", result[1].ReferenceID)
}

func TestFilterByCapabilities(t *testing.T) {
	nodes := []domain.Node{
		makeNode("standard_only", "Standard", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("gpu_and_standard", "GPU", domain.NodeStatusOnline, []string{"standard", "gpu"}, 8, 16384, 102400),
		makeNode("gpu_only", "GPU Only", domain.NodeStatusOnline, []string{"gpu"}, 8, 16384, 102400),
	}

	// Require GPU capability
	result := FilterByCapabilities(nodes, []string{"gpu"})
	assert.Len(t, result, 2)

	// Require both GPU and standard
	result = FilterByCapabilities(nodes, []string{"gpu", "standard"})
	assert.Len(t, result, 1)
	assert.Equal(t, "gpu_and_standard", result[0].ReferenceID)

	// Empty required should return all
	result = FilterByCapabilities(nodes, []string{})
	assert.Len(t, result, 3)
}

func TestFilterByPlanCapabilities(t *testing.T) {
	nodes := []domain.Node{
		makeNode("standard_only", "Standard", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("gpu_only", "GPU", domain.NodeStatusOnline, []string{"gpu"}, 8, 16384, 102400),
		makeNode("high_memory", "High Mem", domain.NodeStatusOnline, []string{"high-memory"}, 8, 32768, 102400),
	}

	// Plan allows only standard
	result := FilterByPlanCapabilities(nodes, []string{"standard"})
	assert.Len(t, result, 1)
	assert.Equal(t, "standard_only", result[0].ReferenceID)

	// Plan allows standard and gpu
	result = FilterByPlanCapabilities(nodes, []string{"standard", "gpu"})
	assert.Len(t, result, 2)

	// Empty allowed should return all (no restrictions)
	result = FilterByPlanCapabilities(nodes, []string{})
	assert.Len(t, result, 3)
}

func TestFilterByCapacity(t *testing.T) {
	nodes := []domain.Node{
		makeNode("small", "Small", domain.NodeStatusOnline, []string{"standard"}, 2, 4096, 20000),
		makeNode("medium", "Medium", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
		makeNode("large", "Large", domain.NodeStatusOnline, []string{"standard"}, 8, 16384, 102400),
	}

	// Require medium resources
	required := domain.Resources{CPUCores: 3, MemoryMB: 6000, DiskMB: 40000}
	result := FilterByCapacity(nodes, required)
	assert.Len(t, result, 2) // Medium and large

	// Require large resources
	required = domain.Resources{CPUCores: 6, MemoryMB: 12000, DiskMB: 80000}
	result = FilterByCapacity(nodes, required)
	assert.Len(t, result, 1) // Only large
	assert.Equal(t, "large", result[0].ReferenceID)
}

func TestSortByScore(t *testing.T) {
	nodes := []domain.Node{
		makeNodeWithUsage("busy", []string{"standard"}, 8, 6, 16384, 12000, 102400, 80000),
		makeNodeWithUsage("idle", []string{"standard"}, 8, 0, 16384, 0, 102400, 0),
		makeNodeWithUsage("half", []string{"standard"}, 8, 4, 16384, 8192, 102400, 51200),
	}

	required := domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000}
	result := SortByScore(nodes, required)

	assert.Len(t, result, 3)
	assert.Equal(t, "idle", result[0].ReferenceID) // Highest score (most available)
	assert.Equal(t, "half", result[1].ReferenceID)
	assert.Equal(t, "busy", result[2].ReferenceID) // Lowest score (least available)
}

// =============================================================================
// Capability Helper Tests
// =============================================================================

func TestCapabilitiesIntersect(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"both empty", []string{}, []string{}, true},
		{"a empty", []string{}, []string{"standard"}, true},
		{"b empty", []string{"standard"}, []string{}, true},
		{"single overlap", []string{"standard"}, []string{"standard"}, true},
		{"partial overlap", []string{"standard", "gpu"}, []string{"gpu", "high-memory"}, true},
		{"no overlap", []string{"standard"}, []string{"gpu"}, false},
		{"multi no overlap", []string{"standard", "ssd"}, []string{"gpu", "high-memory"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CapabilitiesIntersect(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateCapabilityRequirements(t *testing.T) {
	tests := []struct {
		name     string
		required []string
		allowed  []string
		wantErr  error
	}{
		{"no requirements", []string{}, []string{"standard"}, nil},
		{"no restrictions", []string{"gpu"}, []string{}, nil},
		{"both empty", []string{}, []string{}, nil},
		{"requirement satisfied", []string{"gpu"}, []string{"standard", "gpu"}, nil},
		{"multi requirements satisfied", []string{"gpu", "ssd"}, []string{"standard", "gpu", "ssd"}, nil},
		{"requirement not allowed", []string{"gpu"}, []string{"standard"}, ErrNoPlanCapabilities},
		{"partial requirement not allowed", []string{"gpu", "ssd"}, []string{"standard", "gpu"}, ErrNoPlanCapabilities},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCapabilityRequirements(tt.required, tt.allowed)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestSchedule_SingleNode(t *testing.T) {
	nodes := []domain.Node{
		makeNode("only_node", "Only Node", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
	}

	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "only_node", result.SelectedNodeID)
	assert.Equal(t, 1, result.ConsideredCount)
}

func TestSchedule_ExactCapacityMatch(t *testing.T) {
	nodes := []domain.Node{
		makeNode("exact", "Exact", domain.NodeStatusOnline, []string{"standard"}, 4, 8192, 51200),
	}

	// Require exactly what's available
	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 4, MemoryMB: 8192, DiskMB: 51200},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "exact", result.SelectedNodeID)
	assert.Equal(t, 0.0, result.Score) // No headroom left
}

func TestSchedule_NoCapabilitiesRequired(t *testing.T) {
	// Node without standard capability
	nodes := []domain.Node{
		makeNode("special", "Special", domain.NodeStatusOnline, []string{"custom-cap"}, 4, 8192, 51200),
	}

	// No capabilities required, no plan restrictions
	req := ScheduleRequest{
		AvailableNodes:    nodes,
		RequiredResources: domain.Resources{CPUCores: 1, MemoryMB: 1024, DiskMB: 5000},
	}

	result, err := Schedule(req)
	require.NoError(t, err)
	assert.Equal(t, "special", result.SelectedNodeID)
}
