// Package workers contains background workers for Hoster.
package workers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
)

// HealthCheckerConfig configures the health checker worker.
type HealthCheckerConfig struct {
	// Interval is the time between health check cycles.
	// Default: 60 seconds.
	Interval time.Duration

	// NodeTimeout is the timeout for checking a single node.
	// Default: 10 seconds.
	NodeTimeout time.Duration

	// MaxConcurrent is the maximum number of nodes to check concurrently.
	// Default: 5.
	MaxConcurrent int
}

// DefaultHealthCheckerConfig returns the default configuration.
func DefaultHealthCheckerConfig() HealthCheckerConfig {
	return HealthCheckerConfig{
		Interval:      60 * time.Second,
		NodeTimeout:   10 * time.Second,
		MaxConcurrent: 5,
	}
}

// HealthChecker periodically checks the health of registered nodes.
// It updates node status in the database based on connectivity and Docker daemon state.
type HealthChecker struct {
	store         store.Store
	nodePool      *docker.NodePool
	encryptionKey []byte
	config        HealthCheckerConfig
	logger        *slog.Logger

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHealthChecker creates a new health checker worker.
func NewHealthChecker(
	s store.Store,
	nodePool *docker.NodePool,
	encryptionKey []byte,
	config HealthCheckerConfig,
	logger *slog.Logger,
) *HealthChecker {
	if config.Interval == 0 {
		config.Interval = 60 * time.Second
	}
	if config.NodeTimeout == 0 {
		config.NodeTimeout = 10 * time.Second
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 5
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &HealthChecker{
		store:         s,
		nodePool:      nodePool,
		encryptionKey: encryptionKey,
		config:        config,
		logger:        logger.With("component", "health_checker"),
	}
}

// Start begins the health checker background goroutine.
// It runs health checks periodically according to the configured interval.
func (h *HealthChecker) Start() {
	h.ctx, h.cancel = context.WithCancel(context.Background())

	h.wg.Add(1)
	go h.run()

	h.logger.Info("health checker started",
		"interval", h.config.Interval,
		"max_concurrent", h.config.MaxConcurrent,
	)
}

// Stop gracefully stops the health checker.
// It waits for any in-progress health checks to complete.
func (h *HealthChecker) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
	h.logger.Info("health checker stopped")
}

// run is the main loop that runs health checks periodically.
func (h *HealthChecker) run() {
	defer h.wg.Done()

	// Run immediately on start
	h.runCycle()

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.runCycle()
		}
	}
}

// runCycle executes a single health check cycle across all checkable nodes.
func (h *HealthChecker) runCycle() {
	ctx, cancel := context.WithTimeout(h.ctx, h.config.Interval)
	defer cancel()

	// Get all nodes that need checking (not in maintenance mode)
	nodes, err := h.store.ListCheckableNodes(ctx)
	if err != nil {
		h.logger.Error("failed to list checkable nodes", "error", err)
		return
	}

	if len(nodes) == 0 {
		h.logger.Debug("no nodes to check")
		return
	}

	h.logger.Debug("starting health check cycle", "node_count", len(nodes))

	// Use a semaphore to limit concurrent checks
	sem := make(chan struct{}, h.config.MaxConcurrent)
	var wg sync.WaitGroup

	for i := range nodes {
		node := &nodes[i]

		wg.Add(1)
		go func(n *domain.Node) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			h.checkNode(ctx, n)
		}(node)
	}

	wg.Wait()
	h.logger.Debug("completed health check cycle", "node_count", len(nodes))
}

// checkNode performs a health check on a single node.
func (h *HealthChecker) checkNode(ctx context.Context, node *domain.Node) {
	// Use a generous timeout to allow minion deployment on first check.
	// AutoEnsureMinion has its own 2-minute timeout for the upload;
	// the outer context just needs to not cancel before that.
	timeout := h.config.NodeTimeout
	if timeout < 3*time.Minute {
		timeout = 3 * time.Minute
	}
	nodeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logger := h.logger.With("node_id", node.ID, "node_name", node.Name)

	// Try to ping the node (includes auto-deploying minion if needed)
	err := h.pingNode(nodeCtx, node)

	now := time.Now()
	node.LastHealthCheck = &now
	node.UpdatedAt = now

	if err != nil {
		// Node is offline
		if node.Status != domain.NodeStatusOffline {
			logger.Warn("node went offline", "error", err)
		}
		node.Status = domain.NodeStatusOffline
		node.ErrorMessage = err.Error()
	} else {
		// Node is online
		if node.Status == domain.NodeStatusOffline {
			logger.Info("node came online")
		}
		node.Status = domain.NodeStatusOnline
		node.ErrorMessage = ""
	}

	// Update node in database
	if updateErr := h.store.UpdateNode(ctx, node); updateErr != nil {
		logger.Error("failed to update node", "error", updateErr)
	}
}

// pingNode checks if a node is reachable and Docker is running.
func (h *HealthChecker) pingNode(ctx context.Context, node *domain.Node) error {
	// Skip nodes without SSH key configured
	if node.SSHKeyID == "" {
		return domain.ErrSSHHostRequired
	}

	// Get the SSH key and decrypt it
	sshKey, err := h.store.GetSSHKey(ctx, node.SSHKeyID)
	if err != nil {
		return err
	}

	privateKey, err := crypto.DecryptSSHKey(sshKey.PrivateKeyEncrypted, h.encryptionKey)
	if err != nil {
		return err
	}

	// Create a temporary SSH client for the health check
	// We don't use the node pool here because we want to verify the connection
	// without caching potentially stale connections
	client, err := docker.NewSSHDockerClient(node, privateKey, docker.SSHClientConfig{
		CommandTimeout: h.config.NodeTimeout,
		ConnectTimeout: h.config.NodeTimeout,
	})
	if err != nil {
		return err
	}
	defer client.Close()

	// Auto-deploy minion if missing or outdated (use longer timeout for binary upload)
	ensureCtx, ensureCancel := context.WithTimeout(ctx, 2*time.Minute)
	defer ensureCancel()
	if err := client.AutoEnsureMinion(ensureCtx); err != nil {
		return fmt.Errorf("ensure minion: %w", err)
	}

	// Ping the Docker daemon via minion
	return client.Ping()
}

// CheckNodeNow performs an immediate health check on a specific node.
// This is useful for on-demand checks after node configuration changes.
func (h *HealthChecker) CheckNodeNow(ctx context.Context, nodeID string) error {
	node, err := h.store.GetNode(ctx, nodeID)
	if err != nil {
		return err
	}

	// Don't check nodes in maintenance mode
	if node.Status == domain.NodeStatusMaintenance {
		return nil
	}

	h.checkNode(ctx, node)
	return nil
}

// CheckAllNow runs an immediate health check cycle on all nodes.
// This is useful after configuration changes or for manual triggering.
func (h *HealthChecker) CheckAllNow(ctx context.Context) error {
	h.runCycle()
	return nil
}
