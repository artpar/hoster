package docker

import (
	"context"
	"fmt"
	"sync"

	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
)

// NodeStore is the minimal store interface NodePool needs to look up nodes and SSH keys.
type NodeStore interface {
	GetNode(ctx context.Context, nodeID string) (*domain.Node, error)
	GetSSHKey(ctx context.Context, sshKeyRefID string) (*domain.SSHKey, error)
}

// NodePool manages SSH Docker clients for remote nodes.
// It provides lazy initialization and connection caching.
type NodePool struct {
	clients       map[string]*SSHDockerClient // nodeID -> client
	store         NodeStore
	encryptionKey []byte        // Key for decrypting SSH private keys
	config        SSHClientConfig
	mu            sync.RWMutex
}

// NodePoolConfig configures the node pool.
type NodePoolConfig struct {
	SSHClientConfig SSHClientConfig
}

// DefaultNodePoolConfig returns the default configuration.
func DefaultNodePoolConfig() NodePoolConfig {
	return NodePoolConfig{
		SSHClientConfig: DefaultSSHClientConfig(),
	}
}

// NewNodePool creates a new node pool.
// The encryptionKey is used to decrypt SSH private keys stored in the database.
func NewNodePool(s NodeStore, encryptionKey []byte, config NodePoolConfig) *NodePool {
	return &NodePool{
		clients:       make(map[string]*SSHDockerClient),
		store:         s,
		encryptionKey: encryptionKey,
		config:        config.SSHClientConfig,
	}
}

// GetClient returns a Docker client for the given node ID.
// If the client doesn't exist, it creates one (lazy initialization).
// The client is cached for subsequent calls.
func (p *NodePool) GetClient(ctx context.Context, nodeID string) (Client, error) {
	// Fast path: check if client exists
	p.mu.RLock()
	client, exists := p.clients[nodeID]
	p.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Slow path: create client
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := p.clients[nodeID]; exists {
		return client, nil
	}

	// Get node from store
	node, err := p.store.GetNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}

	// Validate node status
	if !node.Status.IsAvailable() {
		return nil, fmt.Errorf("node %s is not available (status: %s)", nodeID, node.Status)
	}

	// Get SSH key from store
	if node.SSHKeyID == 0 {
		return nil, fmt.Errorf("node %s has no SSH key configured", nodeID)
	}

	sshKey, err := p.store.GetSSHKey(ctx, node.SSHKeyRefID)
	if err != nil {
		return nil, fmt.Errorf("get SSH key: %w", err)
	}

	// Decrypt SSH private key
	privateKey, err := crypto.DecryptSSHKey(sshKey.PrivateKeyEncrypted, p.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt SSH key: %w", err)
	}

	// Create SSH Docker client
	client, err = NewSSHDockerClient(node, privateKey, p.config)
	if err != nil {
		return nil, fmt.Errorf("create SSH client: %w", err)
	}

	// Cache the client
	p.clients[nodeID] = client

	return client, nil
}

// GetClientForNode returns a Docker client for the given node.
// This is a convenience method when you already have the node object.
// The privateKey should be the decrypted SSH private key.
func (p *NodePool) GetClientForNode(ctx context.Context, node *domain.Node, privateKey []byte) (Client, error) {
	// Fast path: check if client exists
	p.mu.RLock()
	client, exists := p.clients[node.ReferenceID]
	p.mu.RUnlock()

	if exists {
		return client, nil
	}

	// Slow path: create client
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := p.clients[node.ReferenceID]; exists {
		return client, nil
	}

	// Create SSH Docker client
	client, err := NewSSHDockerClient(node, privateKey, p.config)
	if err != nil {
		return nil, fmt.Errorf("create SSH client: %w", err)
	}

	// Cache the client
	p.clients[node.ReferenceID] = client

	return client, nil
}

// RemoveClient removes a client from the pool and closes its connection.
// This is useful when a node is removed or needs to be reconnected.
func (p *NodePool) RemoveClient(nodeID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, exists := p.clients[nodeID]
	if !exists {
		return nil
	}

	delete(p.clients, nodeID)
	return client.Close()
}

// CloseAll closes all SSH connections in the pool.
// This should be called when shutting down the application.
func (p *NodePool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for nodeID, client := range p.clients {
		if err := client.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close client for node %s: %w", nodeID, err)
		}
		delete(p.clients, nodeID)
	}

	return firstErr
}

// ClientCount returns the number of cached clients.
func (p *NodePool) ClientCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.clients)
}

// HasClient checks if a client for the given node ID is cached.
func (p *NodePool) HasClient(nodeID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.clients[nodeID]
	return exists
}

// PingNode checks if a node is reachable and Docker is running.
// This creates a temporary client if needed and doesn't cache it.
func (p *NodePool) PingNode(ctx context.Context, nodeID string) error {
	client, err := p.GetClient(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}

	return client.Ping()
}

// RefreshClient forces recreation of a client for the given node.
// Useful when node configuration has changed.
func (p *NodePool) RefreshClient(ctx context.Context, nodeID string) (Client, error) {
	// Remove existing client
	if err := p.RemoveClient(nodeID); err != nil {
		return nil, fmt.Errorf("remove existing client: %w", err)
	}

	// Create new client
	return p.GetClient(ctx, nodeID)
}
