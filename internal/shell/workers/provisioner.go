package workers

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/provider"
	"github.com/artpar/hoster/internal/shell/store"
	"golang.org/x/crypto/ssh"
)

// ProvisionerConfig configures the provisioner worker.
type ProvisionerConfig struct {
	Interval      time.Duration
	MaxConcurrent int
}

// DefaultProvisionerConfig returns default configuration.
func DefaultProvisionerConfig() ProvisionerConfig {
	return ProvisionerConfig{
		Interval:      5 * time.Second,
		MaxConcurrent: 3,
	}
}

// Provisioner polls for pending cloud provisions and executes them.
type Provisioner struct {
	store         store.Store
	encryptionKey []byte
	config        ProvisionerConfig
	logger        *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewProvisioner creates a new provisioner worker.
func NewProvisioner(s store.Store, encryptionKey []byte, config ProvisionerConfig, logger *slog.Logger) *Provisioner {
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 3
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Provisioner{
		store:         s,
		encryptionKey: encryptionKey,
		config:        config,
		logger:        logger.With("component", "provisioner"),
	}
}

// Start begins the provisioner background goroutine.
func (p *Provisioner) Start() {
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.wg.Add(1)
	go p.run()
	p.logger.Info("provisioner started", "interval", p.config.Interval, "max_concurrent", p.config.MaxConcurrent)
}

// Stop gracefully stops the provisioner.
func (p *Provisioner) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	p.logger.Info("provisioner stopped")
}

func (p *Provisioner) run() {
	defer p.wg.Done()

	// Run immediately on start
	p.runCycle()

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.runCycle()
		}
	}
}

func (p *Provisioner) runCycle() {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	provisions, err := p.store.ListActiveProvisions(ctx)
	if err != nil {
		p.logger.Error("failed to list active provisions", "error", err)
		return
	}

	if len(provisions) == 0 {
		return
	}

	p.logger.Debug("processing active provisions", "count", len(provisions))

	sem := make(chan struct{}, p.config.MaxConcurrent)
	var wg sync.WaitGroup

	for i := range provisions {
		prov := &provisions[i]
		wg.Add(1)
		go func(pr *domain.CloudProvision) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			p.processProvision(ctx, pr)
		}(prov)
	}

	wg.Wait()
}

func (p *Provisioner) processProvision(ctx context.Context, prov *domain.CloudProvision) {
	logger := p.logger.With("provision_id", prov.ReferenceID, "provider", prov.Provider, "status", prov.Status)

	switch prov.Status {
	case domain.ProvisionStatusPending:
		p.stepCreateInstance(ctx, prov, logger)
	case domain.ProvisionStatusCreating:
		// Instance already being created, check if IP is available
		// (In our implementation, CreateInstance blocks until IP is available,
		// so this state is mainly for resumed provisions)
		p.stepConfigureInstance(ctx, prov, logger)
	case domain.ProvisionStatusConfiguring:
		p.stepFinalize(ctx, prov, logger)
	case domain.ProvisionStatusDestroying:
		p.stepDestroyInstance(ctx, prov, logger)
	}
}

func (p *Provisioner) stepCreateInstance(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	// Bug 3: If a previous attempt already created the cloud instance, skip to configuring
	if prov.ProviderInstanceID != "" {
		logger.Info("reusing existing cloud instance from previous attempt", "instance_id", prov.ProviderInstanceID)
		if err := prov.Transition(domain.ProvisionStatusConfiguring); err != nil {
			p.failProvision(ctx, prov, "failed to transition to configuring: "+err.Error(), logger)
			return
		}
		prov.SetStep("Waiting for Docker to be ready")
		if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
			logger.Error("failed to persist transition to configuring", "error", err)
		}
		return
	}

	logger.Info("starting instance creation")

	// Transition to creating
	if err := prov.Transition(domain.ProvisionStatusCreating); err != nil {
		p.failProvision(ctx, prov, "failed to transition to creating: "+err.Error(), logger)
		return
	}

	var pubKey []byte

	// Resolve SSH key: reuse existing (by ref ID or by creator+name) or generate new
	keyName := fmt.Sprintf("cloud-%s", prov.InstanceName)
	var sshKey *domain.SSHKey

	if prov.SSHKeyID != "" {
		// Previous attempt already linked an SSH key — load it by reference ID
		logger.Info("reusing existing SSH key from previous attempt", "ssh_key_id", prov.SSHKeyID)
		var err error
		sshKey, err = p.store.GetSSHKey(ctx, prov.SSHKeyID)
		if err != nil {
			p.failProvision(ctx, prov, "failed to retrieve existing SSH key: "+err.Error(), logger)
			return
		}
	} else {
		// No SSH key linked — check if an orphaned key from a prior failed attempt exists in DB
		existingKey, err := p.store.GetSSHKeyByCreatorAndName(ctx, prov.CreatorID, keyName)
		if err == nil && existingKey != nil {
			logger.Info("reusing orphaned SSH key from previous failed attempt", "ssh_key_id", existingKey.ReferenceID)
			sshKey = existingKey
			prov.SSHKeyID = existingKey.ReferenceID
			prov.SetStep("Creating cloud instance")
			p.store.UpdateCloudProvision(ctx, prov)
		} else {
			// No existing key at all — generate and store a new one
			prov.SetStep("Generating SSH key pair")
			p.store.UpdateCloudProvision(ctx, prov)

			var privKeyPEM []byte
			pubKey, privKeyPEM, err = generateSSHKeyPair()
			if err != nil {
				p.failProvision(ctx, prov, "failed to generate SSH key: "+err.Error(), logger)
				return
			}

			encryptedKey, err := crypto.EncryptSSHKey(privKeyPEM, p.encryptionKey)
			if err != nil {
				p.failProvision(ctx, prov, "failed to encrypt SSH key: "+err.Error(), logger)
				return
			}

			fingerprint, err := crypto.GetSSHPublicKeyFingerprint(privKeyPEM)
			if err != nil {
				fingerprint = "unknown"
			}

			newKey := &domain.SSHKey{
				ReferenceID:         domain.GenerateSSHKeyID(),
				CreatorID:           prov.CreatorID,
				Name:                keyName,
				PrivateKeyEncrypted: encryptedKey,
				Fingerprint:         fingerprint,
				CreatedAt:           time.Now(),
			}

			if err := p.store.CreateSSHKey(ctx, newKey); err != nil {
				p.failProvision(ctx, prov, "failed to store SSH key: "+err.Error(), logger)
				return
			}

			prov.SSHKeyID = newKey.ReferenceID
			prov.SetStep("Creating cloud instance")
			p.store.UpdateCloudProvision(ctx, prov)
		}
	}

	// Derive public key from stored SSH key (for reuse paths where pubKey wasn't just generated)
	if pubKey == nil {
		privKeyPEM, err := crypto.DecryptSSHKey(sshKey.PrivateKeyEncrypted, p.encryptionKey)
		if err != nil {
			p.failProvision(ctx, prov, "failed to decrypt SSH key: "+err.Error(), logger)
			return
		}
		signer, err := ssh.ParsePrivateKey(privKeyPEM)
		if err != nil {
			p.failProvision(ctx, prov, "failed to parse SSH key: "+err.Error(), logger)
			return
		}
		pubKey = ssh.MarshalAuthorizedKey(signer.PublicKey())
	}

	// Get credentials and create provider client
	cred, err := p.store.GetCloudCredential(ctx, prov.CredentialRefID)
	if err != nil {
		p.failProvision(ctx, prov, "failed to get credentials: "+err.Error(), logger)
		return
	}

	decryptedCreds, err := crypto.DecryptSSHKey(cred.CredentialsEncrypted, p.encryptionKey)
	if err != nil {
		p.failProvision(ctx, prov, "failed to decrypt credentials: "+err.Error(), logger)
		return
	}

	prov.SetStep("Launching instance with provider")
	p.store.UpdateCloudProvision(ctx, prov)

	cloudProvider, err := provider.NewProvider(string(prov.Provider), decryptedCreds, p.logger)
	if err != nil {
		p.failProvision(ctx, prov, "failed to create provider client: "+err.Error(), logger)
		return
	}

	// Create instance
	result, err := cloudProvider.CreateInstance(ctx, provider.ProvisionRequest{
		InstanceName: prov.InstanceName,
		Region:       prov.Region,
		Size:         prov.Size,
		SSHPublicKey: string(pubKey),
	})
	if err != nil {
		p.failProvision(ctx, prov, "failed to create instance: "+err.Error(), logger)
		return
	}

	// Bug 1: Persist ProviderInstanceID + PublicIP immediately BEFORE state transition.
	// If this save fails, the cloud instance is orphaned — log critical and fail.
	prov.ProviderInstanceID = result.ProviderInstanceID
	prov.PublicIP = result.PublicIP
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("CRITICAL: cloud instance created but failed to persist instance ID — instance may be orphaned",
			"instance_id", result.ProviderInstanceID, "ip", result.PublicIP, "error", err)
		p.failProvision(ctx, prov, "failed to persist instance ID after creation: "+err.Error(), logger)
		return
	}
	logger.Info("instance created", "instance_id", result.ProviderInstanceID, "ip", result.PublicIP)

	// Transition to configuring
	if err := prov.Transition(domain.ProvisionStatusConfiguring); err != nil {
		p.failProvision(ctx, prov, "failed to transition to configuring: "+err.Error(), logger)
		return
	}
	prov.SetStep("Waiting for Docker to be ready")
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("failed to persist transition to configuring", "error", err)
	}
}

func (p *Provisioner) stepConfigureInstance(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	if prov.PublicIP == "" {
		p.failProvision(ctx, prov, "no public IP available for configuration", logger)
		return
	}

	prov.SetStep("Configuring Docker on instance")
	p.store.UpdateCloudProvision(ctx, prov)

	// Docker should already be installed via cloud-init/user data
	// Status is already "configuring" from stepCreateInstance — just update step and let next cycle finalize
	prov.SetStep("Creating node record")
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("failed to persist configure step", "error", err)
	}
}

func (p *Provisioner) stepFinalize(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	if prov.NodeID != "" { // NodeID is string reference ID
		// Already have a node, mark as ready
		if err := prov.Transition(domain.ProvisionStatusReady); err != nil {
			p.failProvision(ctx, prov, "failed to transition to ready: "+err.Error(), logger)
			return
		}
		prov.SetStep("Complete")
		if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
			logger.Error("failed to persist ready status", "error", err)
		}
		logger.Info("provision complete", "node_id", prov.NodeID)
		return
	}

	prov.SetStep("Registering node")
	p.store.UpdateCloudProvision(ctx, prov)

	// Bug 4: Check if a node with this creator+name already exists (retry after partial finalize)
	existingNode, err := p.store.GetNodeByCreatorAndName(ctx, prov.CreatorID, prov.InstanceName)
	if err == nil && existingNode != nil {
		logger.Info("reusing existing node from previous attempt", "node_id", existingNode.ReferenceID)
		prov.NodeID = existingNode.ReferenceID
	} else {
		// Create node in the database
		node, err := domain.NewNode(
			prov.CreatorID,
			prov.InstanceName,
			prov.PublicIP,
			"root", // Cloud instances use root by default
			22,
			domain.DefaultCapabilities(),
		)
		if err != nil {
			p.failProvision(ctx, prov, "failed to create node: "+err.Error(), logger)
			return
		}

		node.SSHKeyRefID = prov.SSHKeyID
		node.ProviderType = string(prov.Provider)
		node.ProvisionID = prov.ReferenceID

		if err := p.store.CreateNode(ctx, node); err != nil {
			p.failProvision(ctx, prov, "failed to store node: "+err.Error(), logger)
			return
		}
		prov.NodeID = node.ReferenceID
	}

	// Bug 1: Persist NodeID immediately BEFORE transitioning to ready
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("CRITICAL: node created but failed to persist node ID on provision", "node_id", prov.NodeID, "error", err)
		p.failProvision(ctx, prov, "failed to persist node ID: "+err.Error(), logger)
		return
	}

	if err := prov.Transition(domain.ProvisionStatusReady); err != nil {
		p.failProvision(ctx, prov, "failed to transition to ready: "+err.Error(), logger)
		return
	}
	prov.SetStep("Complete")
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("failed to persist ready status", "error", err)
	}

	logger.Info("provision complete", "node_id", prov.NodeID, "ip", prov.PublicIP)
}

func (p *Provisioner) stepDestroyInstance(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	logger.Info("starting instance destruction")

	// If instance was never created, clean up SSH key and skip to destroyed
	if prov.ProviderInstanceID == "" {
		logger.Info("no provider instance to destroy, marking as destroyed")
		if prov.SSHKeyID != "" {
			if err := p.store.DeleteSSHKey(ctx, prov.SSHKeyID); err != nil {
				logger.Warn("failed to delete SSH key during cleanup", "ssh_key_id", prov.SSHKeyID, "error", err)
			} else {
				logger.Info("deleted SSH key during cleanup", "ssh_key_id", prov.SSHKeyID)
			}
		}
		if err := prov.Transition(domain.ProvisionStatusDestroyed); err != nil {
			p.failProvision(ctx, prov, "failed to transition to destroyed: "+err.Error(), logger)
			return
		}
		prov.SetStep("Destroyed (no instance to clean up)")
		if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
			logger.Error("failed to persist destroyed status", "error", err)
		}
		return
	}

	// Get credentials to create provider client
	cred, err := p.store.GetCloudCredential(ctx, prov.CredentialRefID)
	if err != nil {
		p.failProvision(ctx, prov, "failed to get credentials for destroy: "+err.Error(), logger)
		return
	}

	decryptedCreds, err := crypto.DecryptSSHKey(cred.CredentialsEncrypted, p.encryptionKey)
	if err != nil {
		p.failProvision(ctx, prov, "failed to decrypt credentials for destroy: "+err.Error(), logger)
		return
	}

	cloudProvider, err := provider.NewProvider(string(prov.Provider), decryptedCreds, p.logger)
	if err != nil {
		p.failProvision(ctx, prov, "failed to create provider client for destroy: "+err.Error(), logger)
		return
	}

	// Destroy the instance (Bug 6, 8: pass DestroyRequest with region + instance name for cleanup)
	prov.SetStep("Destroying cloud instance")
	p.store.UpdateCloudProvision(ctx, prov)

	if err := cloudProvider.DestroyInstance(ctx, provider.DestroyRequest{
		ProviderInstanceID: prov.ProviderInstanceID,
		InstanceName:       prov.InstanceName,
		Region:             prov.Region,
	}); err != nil {
		p.failProvision(ctx, prov, "failed to destroy instance: "+err.Error(), logger)
		return
	}

	// Bug 5: Mark deployments on the node as deleted before removing the node
	if prov.NodeID != "" {
		deployments, err := p.store.ListDeploymentsByNode(ctx, prov.NodeID)
		if err != nil {
			logger.Warn("failed to list deployments on node", "node_id", prov.NodeID, "error", err)
		} else {
			now := time.Now()
			for i := range deployments {
				d := &deployments[i]
				d.Status = domain.StatusDeleted
				d.ErrorMessage = "Node destroyed via cloud provision"
				d.StoppedAt = &now
				if err := p.store.UpdateDeployment(ctx, d); err != nil {
					logger.Warn("failed to mark deployment as deleted", "deployment_id", d.ReferenceID, "error", err)
				} else {
					logger.Info("marked deployment as deleted", "deployment_id", d.ReferenceID)
				}
			}
		}
	}

	// Delete associated node if one was created
	if prov.NodeID != "" {
		if err := p.store.DeleteNode(ctx, prov.NodeID); err != nil {
			logger.Warn("failed to delete associated node", "node_id", prov.NodeID, "error", err)
		} else {
			logger.Info("deleted associated node", "node_id", prov.NodeID)
		}
	}

	// Delete associated SSH key (after node, since node references key via FK)
	if prov.SSHKeyID != "" {
		if err := p.store.DeleteSSHKey(ctx, prov.SSHKeyID); err != nil {
			logger.Warn("failed to delete associated SSH key", "ssh_key_id", prov.SSHKeyID, "error", err)
		} else {
			logger.Info("deleted associated SSH key", "ssh_key_id", prov.SSHKeyID)
		}
	}

	if err := prov.Transition(domain.ProvisionStatusDestroyed); err != nil {
		p.failProvision(ctx, prov, "failed to transition to destroyed: "+err.Error(), logger)
		return
	}
	prov.SetStep("Instance destroyed")
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("failed to persist destroyed status", "error", err)
	}

	logger.Info("instance destroyed", "instance_id", prov.ProviderInstanceID)
}

func (p *Provisioner) failProvision(ctx context.Context, prov *domain.CloudProvision, errMsg string, logger *slog.Logger) {
	logger.Error("provision failed", "error", errMsg)
	if err := prov.TransitionToFailed(errMsg); err != nil {
		// Transition invalid (e.g., already terminal) — set error message directly
		logger.Error("failed to transition to failed state", "transition_error", err, "current_status", prov.Status)
		prov.ErrorMessage = errMsg
	}
	if err := p.store.UpdateCloudProvision(ctx, prov); err != nil {
		logger.Error("failed to persist failed status", "error", err)
	}
}

// generateSSHKeyPair generates an Ed25519 SSH key pair.
// Returns the public key (authorized_keys format) and private key (PEM format).
func generateSSHKeyPair() (publicKey []byte, privateKeyPEM []byte, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Marshal public key to authorized_keys format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SSH public key: %w", err)
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)

	// Marshal private key to PEM
	privKeyBytes, err := ssh.MarshalPrivateKey(privKey, "hoster-provisioned")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(privKeyBytes)

	return pubKeyBytes, privPEM, nil
}
