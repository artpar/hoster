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
	}
}

func (p *Provisioner) stepCreateInstance(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	logger.Info("starting instance creation")

	// Transition to creating
	prov.Transition(domain.ProvisionStatusCreating)
	prov.SetStep("Generating SSH key pair")
	p.store.UpdateCloudProvision(ctx, prov)

	// Generate SSH key pair
	pubKey, privKeyPEM, err := generateSSHKeyPair()
	if err != nil {
		p.failProvision(ctx, prov, "failed to generate SSH key: "+err.Error(), logger)
		return
	}

	// Encrypt and store the SSH key
	encryptedKey, err := crypto.EncryptSSHKey(privKeyPEM, p.encryptionKey)
	if err != nil {
		p.failProvision(ctx, prov, "failed to encrypt SSH key: "+err.Error(), logger)
		return
	}

	fingerprint, err := crypto.GetSSHPublicKeyFingerprint(privKeyPEM)
	if err != nil {
		fingerprint = "unknown"
	}

	sshKey := &domain.SSHKey{
		ReferenceID:         domain.GenerateSSHKeyID(),
		CreatorID:           prov.CreatorID,
		Name:                fmt.Sprintf("cloud-%s", prov.InstanceName),
		PrivateKeyEncrypted: encryptedKey,
		Fingerprint:         fingerprint,
		CreatedAt:           time.Now(),
	}

	if err := p.store.CreateSSHKey(ctx, sshKey); err != nil {
		p.failProvision(ctx, prov, "failed to store SSH key: "+err.Error(), logger)
		return
	}

	prov.SSHKeyID = sshKey.ReferenceID
	prov.SetStep("Creating cloud instance")
	p.store.UpdateCloudProvision(ctx, prov)

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

	prov.ProviderInstanceID = result.ProviderInstanceID
	prov.PublicIP = result.PublicIP
	logger.Info("instance created", "instance_id", result.ProviderInstanceID, "ip", result.PublicIP)

	// Transition to configuring
	prov.Transition(domain.ProvisionStatusConfiguring)
	prov.SetStep("Waiting for Docker to be ready")
	p.store.UpdateCloudProvision(ctx, prov)
}

func (p *Provisioner) stepConfigureInstance(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	if prov.PublicIP == "" {
		p.failProvision(ctx, prov, "no public IP available for configuration", logger)
		return
	}

	prov.SetStep("Configuring Docker on instance")
	p.store.UpdateCloudProvision(ctx, prov)

	// Docker should already be installed via cloud-init/user data
	// Transition to configuring and let the next cycle finalize
	prov.Transition(domain.ProvisionStatusConfiguring)
	prov.SetStep("Creating node record")
	p.store.UpdateCloudProvision(ctx, prov)
}

func (p *Provisioner) stepFinalize(ctx context.Context, prov *domain.CloudProvision, logger *slog.Logger) {
	if prov.NodeID != "" { // NodeID is string reference ID
		// Already have a node, mark as ready
		prov.Transition(domain.ProvisionStatusReady)
		prov.SetStep("Complete")
		p.store.UpdateCloudProvision(ctx, prov)
		logger.Info("provision complete", "node_id", prov.NodeID)
		return
	}

	prov.SetStep("Registering node")
	p.store.UpdateCloudProvision(ctx, prov)

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
	prov.Transition(domain.ProvisionStatusReady)
	prov.SetStep("Complete")
	p.store.UpdateCloudProvision(ctx, prov)

	logger.Info("provision complete", "node_id", node.ReferenceID, "ip", prov.PublicIP)
}

func (p *Provisioner) failProvision(ctx context.Context, prov *domain.CloudProvision, errMsg string, logger *slog.Logger) {
	logger.Error("provision failed", "error", errMsg)
	prov.TransitionToFailed(errMsg)
	p.store.UpdateCloudProvision(ctx, prov)
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
