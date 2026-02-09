// Package scheduler provides the pure scheduling algorithm for node selection.
// This is part of the Functional Core - all functions are pure with no I/O.
package scheduler

import (
	"errors"
	"sort"

	"github.com/artpar/hoster/internal/core/domain"
)

// =============================================================================
// Scheduler Errors
// =============================================================================

var (
	// ErrNoNodesAvailable is returned when no nodes match the requirements.
	ErrNoNodesAvailable = errors.New("no nodes available for this deployment")

	// ErrNoCapableNodes is returned when no nodes have the required capabilities.
	ErrNoCapableNodes = errors.New("no nodes have the required capabilities")

	// ErrNoPlanCapabilities is returned when user's plan doesn't permit required capabilities.
	ErrNoPlanCapabilities = errors.New("plan does not allow required node capabilities")

	// ErrInsufficientCapacity is returned when no nodes have enough resources.
	ErrInsufficientCapacity = errors.New("no nodes have sufficient capacity")
)

// =============================================================================
// Scheduling Request
// =============================================================================

// ScheduleRequest contains all information needed to select a node.
type ScheduleRequest struct {
	// AvailableNodes is the list of all nodes to consider
	AvailableNodes []domain.Node

	// RequiredResources are the minimum resources needed
	RequiredResources domain.Resources

	// RequiredCapabilities are the node capabilities the template requires (e.g., ["gpu"])
	RequiredCapabilities []string

	// AllowedCapabilities are the node capabilities the user's plan permits (e.g., ["standard", "gpu"])
	AllowedCapabilities []string
}

// =============================================================================
// Scheduling Result
// =============================================================================

// ScheduleResult contains the result of the scheduling algorithm.
type ScheduleResult struct {
	// SelectedNodeID is the ID of the best node, empty if none found
	SelectedNodeID string

	// SelectedNode is a copy of the selected node, nil if none found
	SelectedNode *domain.Node

	// Score is the score of the selected node (0-100)
	Score float64

	// ConsideredCount is the number of nodes that were considered
	ConsideredCount int

	// FilteredOutReason tracks why nodes were filtered out
	FilteredOutReasons map[string]int
}

// =============================================================================
// Node Candidate (internal)
// =============================================================================

// nodeCandidate is a node with its computed score.
type nodeCandidate struct {
	node  domain.Node
	score float64
}

// =============================================================================
// Scheduling Algorithm
// =============================================================================

// Schedule selects the best node for a deployment based on the request.
// Returns the result with selected node ID, or error if no suitable node found.
//
// Algorithm:
// 1. Filter nodes to only ONLINE nodes
// 2. Filter nodes that have ALL required capabilities (if any)
// 3. Filter nodes that have AT LEAST ONE capability allowed by user's plan
// 4. Filter nodes with sufficient capacity for the required resources
// 5. Score remaining nodes by available resources (higher is better)
// 6. Return highest-scoring node
func Schedule(req ScheduleRequest) (*ScheduleResult, error) {
	result := &ScheduleResult{
		FilteredOutReasons: make(map[string]int),
	}

	if len(req.AvailableNodes) == 0 {
		return result, ErrNoNodesAvailable
	}

	var candidates []nodeCandidate

	for _, node := range req.AvailableNodes {
		result.ConsideredCount++

		// Step 1: Must be online
		if !node.IsAvailable() {
			result.FilteredOutReasons["not_online"]++
			continue
		}

		// Step 2: Must have all required capabilities (if any specified)
		if len(req.RequiredCapabilities) > 0 {
			if !node.HasAllCapabilities(req.RequiredCapabilities) {
				result.FilteredOutReasons["missing_required_capabilities"]++
				continue
			}
		}

		// Step 3: Must have at least one capability allowed by user's plan
		// If no allowed capabilities specified, skip this check (allow all)
		if len(req.AllowedCapabilities) > 0 {
			if !node.HasAnyCapability(req.AllowedCapabilities) {
				result.FilteredOutReasons["plan_capabilities_mismatch"]++
				continue
			}
		}

		// Step 4: Must have sufficient capacity
		if !node.Capacity.CanHandle(req.RequiredResources) {
			result.FilteredOutReasons["insufficient_capacity"]++
			continue
		}

		// Node passed all filters, calculate score
		score := ScoreNode(node, req.RequiredResources)
		candidates = append(candidates, nodeCandidate{
			node:  node,
			score: score,
		})
	}

	if len(candidates) == 0 {
		// Determine the most appropriate error based on filter reasons
		if result.FilteredOutReasons["plan_capabilities_mismatch"] > 0 &&
			result.FilteredOutReasons["missing_required_capabilities"] == 0 {
			return result, ErrNoPlanCapabilities
		}
		if result.FilteredOutReasons["missing_required_capabilities"] > 0 {
			return result, ErrNoCapableNodes
		}
		if result.FilteredOutReasons["insufficient_capacity"] > 0 {
			return result, ErrInsufficientCapacity
		}
		return result, ErrNoNodesAvailable
	}

	// Sort by score descending (highest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Select the best node
	best := candidates[0]
	result.SelectedNodeID = best.node.ReferenceID
	result.SelectedNode = &best.node
	result.Score = best.score

	return result, nil
}

// =============================================================================
// Scoring Algorithm
// =============================================================================

// ScoreNode calculates a score for a node based on available resources.
// Higher scores indicate better candidates (more available resources).
// Score range: 0-100
//
// Formula (weighted average):
//   - CPU: 30% weight
//   - Memory: 40% weight (most important for containers)
//   - Disk: 30% weight
func ScoreNode(node domain.Node, required domain.Resources) float64 {
	cap := node.Capacity

	// Calculate available resources after deployment
	availableCPU := cap.AvailableCPU() - required.CPUCores
	availableMemory := cap.AvailableMemory() - required.MemoryMB
	availableDisk := cap.AvailableDisk() - required.DiskMB

	// Calculate percentage of capacity that will remain
	cpuPercent := 0.0
	memoryPercent := 0.0
	diskPercent := 0.0

	if cap.CPUCores > 0 {
		cpuPercent = (availableCPU / cap.CPUCores) * 100
		if cpuPercent < 0 {
			cpuPercent = 0
		}
		if cpuPercent > 100 {
			cpuPercent = 100
		}
	}

	if cap.MemoryMB > 0 {
		memoryPercent = (float64(availableMemory) / float64(cap.MemoryMB)) * 100
		if memoryPercent < 0 {
			memoryPercent = 0
		}
		if memoryPercent > 100 {
			memoryPercent = 100
		}
	}

	if cap.DiskMB > 0 {
		diskPercent = (float64(availableDisk) / float64(cap.DiskMB)) * 100
		if diskPercent < 0 {
			diskPercent = 0
		}
		if diskPercent > 100 {
			diskPercent = 100
		}
	}

	// Weighted average: memory is most important for containers
	score := cpuPercent*0.3 + memoryPercent*0.4 + diskPercent*0.3

	return score
}

// =============================================================================
// Helper Functions
// =============================================================================

// FilterOnlineNodes returns only the online nodes from the list.
func FilterOnlineNodes(nodes []domain.Node) []domain.Node {
	result := make([]domain.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.IsAvailable() {
			result = append(result, n)
		}
	}
	return result
}

// FilterByCapabilities returns nodes that have all the required capabilities.
func FilterByCapabilities(nodes []domain.Node, required []string) []domain.Node {
	if len(required) == 0 {
		return nodes
	}

	result := make([]domain.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.HasAllCapabilities(required) {
			result = append(result, n)
		}
	}
	return result
}

// FilterByPlanCapabilities returns nodes that have at least one of the allowed capabilities.
func FilterByPlanCapabilities(nodes []domain.Node, allowed []string) []domain.Node {
	if len(allowed) == 0 {
		return nodes
	}

	result := make([]domain.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.HasAnyCapability(allowed) {
			result = append(result, n)
		}
	}
	return result
}

// FilterByCapacity returns nodes that can handle the required resources.
func FilterByCapacity(nodes []domain.Node, required domain.Resources) []domain.Node {
	result := make([]domain.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Capacity.CanHandle(required) {
			result = append(result, n)
		}
	}
	return result
}

// SortByScore sorts nodes by their score (highest first).
func SortByScore(nodes []domain.Node, required domain.Resources) []domain.Node {
	result := make([]domain.Node, len(nodes))
	copy(result, nodes)

	sort.Slice(result, func(i, j int) bool {
		scoreI := ScoreNode(result[i], required)
		scoreJ := ScoreNode(result[j], required)
		return scoreI > scoreJ
	})

	return result
}

// CapabilitiesIntersect checks if there is any overlap between two capability lists.
func CapabilitiesIntersect(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return true // No restrictions
	}

	set := make(map[string]bool, len(a))
	for _, cap := range a {
		set[cap] = true
	}

	for _, cap := range b {
		if set[cap] {
			return true
		}
	}
	return false
}

// ValidateCapabilityRequirements checks if the required capabilities can be satisfied
// by the allowed capabilities. Returns error if the template requires capabilities
// that the user's plan doesn't allow.
func ValidateCapabilityRequirements(required, allowed []string) error {
	// If no specific capabilities are required, any plan works
	if len(required) == 0 {
		return nil
	}

	// If no capabilities are restricted (empty allowed = allow all), any requirement works
	if len(allowed) == 0 {
		return nil
	}

	// Check that all required capabilities are in allowed list
	allowedSet := make(map[string]bool, len(allowed))
	for _, cap := range allowed {
		allowedSet[cap] = true
	}

	for _, req := range required {
		if !allowedSet[req] {
			return ErrNoPlanCapabilities
		}
	}

	return nil
}
