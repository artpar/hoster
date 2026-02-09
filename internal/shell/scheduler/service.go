// Package scheduler provides the scheduling service for node selection with I/O.
// This is part of the Imperative Shell - it handles I/O and calls the pure scheduler.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/artpar/hoster/internal/core/domain"
	corescheduler "github.com/artpar/hoster/internal/core/scheduler"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
)

// =============================================================================
// Service Errors
// =============================================================================

var (
	// ErrNoNodesConfigured is returned when no nodes exist in the system.
	ErrNoNodesConfigured = errors.New("no nodes configured")

	// ErrLocalNodeRequired is returned when template requires local execution.
	ErrLocalNodeRequired = errors.New("template requires local node execution")
)

// =============================================================================
// Scheduling Service
// =============================================================================

// Service provides scheduling functionality with I/O operations.
// It loads nodes from the store and uses the pure scheduler algorithm.
type Service struct {
	store       store.Store
	nodePool    *docker.NodePool
	localClient docker.Client
	logger      *slog.Logger
}

// NewService creates a new scheduling service.
// localClient is used for deployments when no remote nodes are available.
// nodePool may be nil if only local deployments are supported.
func NewService(s store.Store, nodePool *docker.NodePool, localClient docker.Client, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		store:       s,
		nodePool:    nodePool,
		localClient: localClient,
		logger:      logger,
	}
}

// =============================================================================
// Schedule Request/Result
// =============================================================================

// ScheduleDeploymentRequest contains the input for scheduling a deployment.
type ScheduleDeploymentRequest struct {
	// Template is the template being deployed (for required capabilities and resources)
	Template *domain.Template

	// CreatorID is the ID of the template creator (to filter nodes by ownership)
	CreatorID int

	// AllowedCapabilities are the capabilities allowed by the user's plan
	// If empty, all capabilities are allowed
	AllowedCapabilities []string

	// PreferredNodeID optionally specifies a preferred node (for restarts)
	PreferredNodeID string
}

// ScheduleDeploymentResult contains the result of scheduling.
type ScheduleDeploymentResult struct {
	// NodeID is the selected node ID ("local" for local Docker)
	NodeID string

	// Node is the selected node (nil for local)
	Node *domain.Node

	// Client is the Docker client for the selected node
	Client docker.Client

	// IsLocal indicates if this is a local deployment
	IsLocal bool

	// Score is the scheduler score (0 for local)
	Score float64
}

// =============================================================================
// Schedule Deployment
// =============================================================================

// ScheduleDeployment selects the best node for a deployment.
// If no remote nodes are available/suitable, falls back to local.
//
// The algorithm:
// 1. Get all online nodes for the template's creator
// 2. If preferred node is specified and available, use it
// 3. Otherwise, run the scheduler algorithm
// 4. If no suitable node found, fall back to local
func (s *Service) ScheduleDeployment(ctx context.Context, req ScheduleDeploymentRequest) (*ScheduleDeploymentResult, error) {
	s.logger.Debug("scheduling deployment",
		"template_id", req.Template.ReferenceID,
		"creator_id", req.CreatorID,
		"required_capabilities", req.Template.RequiredCapabilities,
		"preferred_node", req.PreferredNodeID,
	)

	// If no node pool configured, use local
	if s.nodePool == nil {
		s.logger.Debug("no node pool configured, using local")
		return s.localResult(), nil
	}

	// Get online nodes for the creator
	nodes, err := s.store.ListOnlineNodes(ctx)
	if err != nil {
		s.logger.Warn("failed to list online nodes, falling back to local", "error", err)
		return s.localResult(), nil
	}

	// Filter nodes by creator
	creatorNodes := filterNodesByCreator(nodes, req.CreatorID)
	if len(creatorNodes) == 0 {
		s.logger.Debug("no nodes for creator, using local", "creator_id", req.CreatorID)
		return s.localResult(), nil
	}

	// If preferred node specified and available, try to use it
	if req.PreferredNodeID != "" && req.PreferredNodeID != "local" {
		for _, node := range creatorNodes {
			if node.ReferenceID == req.PreferredNodeID && node.IsAvailable() {
				client, err := s.nodePool.GetClient(ctx, node.ReferenceID)
				if err != nil {
					s.logger.Warn("preferred node unavailable", "node_id", node.ReferenceID, "error", err)
					break // Try scheduler instead
				}
				nodeCopy := node
				return &ScheduleDeploymentResult{
					NodeID:  node.ReferenceID,
					Node:    &nodeCopy,
					Client:  client,
					IsLocal: false,
					Score:   100, // Preferred node gets max score
				}, nil
			}
		}
	}

	// Build scheduler request
	schedReq := corescheduler.ScheduleRequest{
		AvailableNodes:       creatorNodes,
		RequiredResources:    req.Template.ResourceRequirements,
		RequiredCapabilities: req.Template.RequiredCapabilities,
		AllowedCapabilities:  req.AllowedCapabilities,
	}

	// Run pure scheduler
	result, err := corescheduler.Schedule(schedReq)
	if err != nil {
		s.logger.Debug("scheduler returned no suitable node, using local",
			"error", err,
			"considered", result.ConsideredCount,
			"filtered_reasons", result.FilteredOutReasons,
		)
		return s.localResult(), nil
	}

	// Get client for selected node
	client, err := s.nodePool.GetClient(ctx, result.SelectedNodeID)
	if err != nil {
		s.logger.Warn("failed to get client for selected node, falling back to local",
			"node_id", result.SelectedNodeID,
			"error", err,
		)
		return s.localResult(), nil
	}

	s.logger.Info("scheduled deployment to node",
		"node_id", result.SelectedNodeID,
		"node_name", result.SelectedNode.Name,
		"score", result.Score,
	)

	return &ScheduleDeploymentResult{
		NodeID:  result.SelectedNodeID,
		Node:    result.SelectedNode,
		Client:  client,
		IsLocal: false,
		Score:   result.Score,
	}, nil
}

// =============================================================================
// Get Client for Node
// =============================================================================

// GetClientForNode returns the Docker client for a specific node ID.
// Returns the local client if nodeID is "local".
func (s *Service) GetClientForNode(ctx context.Context, nodeID string) (docker.Client, error) {
	if nodeID == "" || nodeID == "local" {
		if s.localClient == nil {
			return nil, errors.New("local client not configured")
		}
		return s.localClient, nil
	}

	if s.nodePool == nil {
		return nil, fmt.Errorf("node pool not configured, cannot get client for node %s", nodeID)
	}

	return s.nodePool.GetClient(ctx, nodeID)
}

// =============================================================================
// Local Fallback
// =============================================================================

// localResult returns a result for local deployment.
func (s *Service) localResult() *ScheduleDeploymentResult {
	return &ScheduleDeploymentResult{
		NodeID:  "local",
		Node:    nil,
		Client:  s.localClient,
		IsLocal: true,
		Score:   0,
	}
}

// SupportsRemoteNodes returns true if the service has a node pool configured.
func (s *Service) SupportsRemoteNodes() bool {
	return s.nodePool != nil
}

// LocalClient returns the local Docker client.
func (s *Service) LocalClient() docker.Client {
	return s.localClient
}

// =============================================================================
// Helpers
// =============================================================================

// filterNodesByCreator filters nodes to only those owned by the creator.
func filterNodesByCreator(nodes []domain.Node, creatorID int) []domain.Node {
	if creatorID == 0 {
		return nodes
	}

	result := make([]domain.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.CreatorID == creatorID {
			result = append(result, n)
		}
	}
	return result
}
