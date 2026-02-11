package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/provider"
)

// =============================================================================
// Health Checker
// =============================================================================

// HealthChecker periodically checks node health via SSH.
type HealthChecker struct {
	store         *Store
	nodePool      *docker.NodePool
	encryptionKey []byte
	interval      time.Duration
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewHealthChecker creates a health checker that uses the engine store directly.
func NewHealthChecker(store *Store, nodePool *docker.NodePool, encryptionKey []byte, interval time.Duration, logger *slog.Logger) *HealthChecker {
	if interval == 0 {
		interval = 60 * time.Second
	}
	return &HealthChecker{
		store:         store,
		nodePool:      nodePool,
		encryptionKey: encryptionKey,
		interval:      interval,
		logger:        logger.With("component", "health_checker"),
	}
}

func (h *HealthChecker) Start() {
	h.ctx, h.cancel = context.WithCancel(context.Background())
	h.wg.Add(1)
	go h.run()
	h.logger.Info("health checker started", "interval", h.interval)
}

func (h *HealthChecker) Stop() {
	if h.cancel != nil {
		h.cancel()
	}
	h.wg.Wait()
}

func (h *HealthChecker) run() {
	defer h.wg.Done()
	h.checkAll()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.checkAll()
		}
	}
}

func (h *HealthChecker) checkAll() {
	nodes, err := h.store.List(h.ctx, "nodes", []Filter{}, Page{Limit: 1000})
	if err != nil {
		h.logger.Error("failed to list nodes", "error", err)
		return
	}

	for _, node := range nodes {
		refID, _ := node["reference_id"].(string)
		status, _ := node["status"].(string)
		if status == "maintenance" {
			continue
		}

		if h.nodePool == nil {
			continue
		}

		err := h.nodePool.PingNode(h.ctx, refID)
		now := time.Now().UTC().Format(time.RFC3339)

		if err != nil {
			h.logger.Debug("node health check failed", "node", refID, "error", err)
			h.store.Update(h.ctx, "nodes", refID, map[string]any{
				"status":            "offline",
				"last_health_check": now,
				"error_message":     err.Error(),
			})
		} else {
			h.store.Update(h.ctx, "nodes", refID, map[string]any{
				"status":            "online",
				"last_health_check": now,
				"error_message":     "",
			})
		}
	}
}

// CheckNode triggers an immediate health check for a single node.
func (h *HealthChecker) CheckNode(ctx context.Context, nodeRefID string) {
	if h.nodePool == nil {
		return
	}
	err := h.nodePool.PingNode(ctx, nodeRefID)
	now := time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		h.store.Update(ctx, "nodes", nodeRefID, map[string]any{
			"status":            "offline",
			"last_health_check": now,
			"error_message":     err.Error(),
		})
	} else {
		h.store.Update(ctx, "nodes", nodeRefID, map[string]any{
			"status":            "online",
			"last_health_check": now,
			"error_message":     "",
		})
	}
}

// =============================================================================
// Provisioner
// =============================================================================

// Provisioner polls for active cloud provisions and processes them.
type Provisioner struct {
	store         *Store
	encryptionKey []byte
	interval      time.Duration
	logger        *slog.Logger
	healthChecker *HealthChecker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewProvisioner(store *Store, encryptionKey []byte, interval time.Duration, logger *slog.Logger) *Provisioner {
	if interval == 0 {
		interval = 5 * time.Second
	}
	return &Provisioner{
		store:         store,
		encryptionKey: encryptionKey,
		interval:      interval,
		logger:        logger.With("component", "provisioner"),
	}
}

func (p *Provisioner) SetHealthChecker(hc *HealthChecker) {
	p.healthChecker = hc
}

func (p *Provisioner) Start() {
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.wg.Add(1)
	go p.run()
	p.logger.Info("provisioner started", "interval", p.interval)
}

func (p *Provisioner) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *Provisioner) run() {
	defer p.wg.Done()
	p.runCycle()

	ticker := time.NewTicker(p.interval)
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
	// Query for active provisions
	rows, err := p.store.RawQuery(p.ctx,
		`SELECT cp.*, cc.credentials_encrypted, cc.provider as cred_provider
		 FROM cloud_provisions cp
		 JOIN cloud_credentials cc ON cc.id = cp.credential_id
		 WHERE cp.status IN ('pending', 'creating', 'configuring', 'destroying')
		 ORDER BY cp.id ASC LIMIT 10`)
	if err != nil {
		p.logger.Error("failed to list active provisions", "error", err)
		return
	}

	for _, row := range rows {
		refID := strVal(row["reference_id"])
		status := strVal(row["status"])

		switch status {
		case "pending":
			p.stepCreate(p.ctx, row)
		case "creating":
			p.stepConfigure(p.ctx, row)
		case "configuring":
			p.stepFinalize(p.ctx, row)
		case "destroying":
			p.stepDestroy(p.ctx, row)
		default:
			p.logger.Warn("unknown provision status", "ref_id", refID, "status", status)
		}
	}
}

func (p *Provisioner) stepCreate(ctx context.Context, row map[string]any) {
	refID := strVal(row["reference_id"])
	providerType := strVal(row["provider"])

	// Decrypt credentials
	credEncrypted := row["credentials_encrypted"]
	var credBytes []byte
	switch v := credEncrypted.(type) {
	case []byte:
		credBytes = v
	case string:
		credBytes = []byte(v)
	}

	decrypted, err := crypto.Decrypt(credBytes, p.encryptionKey)
	if err != nil {
		p.failProvision(ctx, refID, "decrypt credentials: "+err.Error())
		return
	}

	prov, err := provider.NewProvider(providerType, decrypted, p.logger)
	if err != nil {
		p.failProvision(ctx, refID, "create provider: "+err.Error())
		return
	}

	instanceName := strVal(row["instance_name"])
	region := strVal(row["region"])
	size := strVal(row["size"])

	// Create instance
	result, err := prov.CreateInstance(ctx, provider.ProvisionRequest{
		InstanceName: instanceName,
		Region:       region,
		Size:         size,
	})
	if err != nil {
		p.failProvision(ctx, refID, "create instance: "+err.Error())
		return
	}

	// Update provision with instance details
	p.store.Update(ctx, "cloud_provisions", refID, map[string]any{
		"provider_instance_id": result.ProviderInstanceID,
		"public_ip":            result.PublicIP,
		"current_step":         "instance_created",
	})

	// Transition to creating
	p.store.Transition(ctx, "cloud_provisions", refID, "creating")
	p.logger.Info("instance created", "provision", refID, "instance_id", result.ProviderInstanceID)
}

func (p *Provisioner) stepConfigure(ctx context.Context, row map[string]any) {
	refID := strVal(row["reference_id"])

	// Transition to configuring
	p.store.Update(ctx, "cloud_provisions", refID, map[string]any{
		"current_step": "configuring_instance",
	})
	p.store.Transition(ctx, "cloud_provisions", refID, "configuring")
	p.logger.Info("instance configuring", "provision", refID)
}

func (p *Provisioner) stepFinalize(ctx context.Context, row map[string]any) {
	refID := strVal(row["reference_id"])

	// Transition to ready
	now := time.Now().UTC().Format(time.RFC3339)
	p.store.Update(ctx, "cloud_provisions", refID, map[string]any{
		"current_step": "ready",
		"completed_at": now,
	})
	p.store.Transition(ctx, "cloud_provisions", refID, "ready")

	// Trigger health check on the new node if it was linked
	nodeID := strVal(row["node_id"])
	if nodeID != "" && p.healthChecker != nil {
		p.healthChecker.CheckNode(ctx, nodeID)
	}

	p.logger.Info("provision ready", "provision", refID)
}

func (p *Provisioner) stepDestroy(ctx context.Context, row map[string]any) {
	refID := strVal(row["reference_id"])
	providerType := strVal(row["provider"])
	instanceID := strVal(row["provider_instance_id"])

	if instanceID == "" {
		// No instance to destroy — just mark as destroyed
		p.store.Transition(ctx, "cloud_provisions", refID, "destroyed")
		return
	}

	// Decrypt credentials
	credEncrypted := row["credentials_encrypted"]
	var credBytes []byte
	switch v := credEncrypted.(type) {
	case []byte:
		credBytes = v
	case string:
		credBytes = []byte(v)
	}

	decrypted, err := crypto.Decrypt(credBytes, p.encryptionKey)
	if err != nil {
		p.failProvision(ctx, refID, "decrypt credentials: "+err.Error())
		return
	}

	prov, err := provider.NewProvider(providerType, decrypted, p.logger)
	if err != nil {
		p.failProvision(ctx, refID, "create provider: "+err.Error())
		return
	}

	destroyReq := provider.DestroyRequest{
		ProviderInstanceID: instanceID,
		InstanceName:       strVal(row["instance_name"]),
		Region:             strVal(row["region"]),
	}
	if err := prov.DestroyInstance(ctx, destroyReq); err != nil {
		p.logger.Warn("destroy instance failed, treating as success", "provision", refID, "error", err)
	}

	p.store.Transition(ctx, "cloud_provisions", refID, "destroyed")
	p.logger.Info("instance destroyed", "provision", refID, "instance_id", instanceID)
}

func (p *Provisioner) failProvision(ctx context.Context, refID, reason string) {
	p.store.Update(ctx, "cloud_provisions", refID, map[string]any{
		"error_message": reason,
	})
	p.store.Transition(ctx, "cloud_provisions", refID, "failed")
	p.logger.Error("provision failed", "provision", refID, "error", reason)
}

// =============================================================================
// DNS Verifier
// =============================================================================

// DNSVerifier periodically checks custom domain DNS records.
type DNSVerifier struct {
	store      *Store
	baseDomain string
	interval   time.Duration
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewDNSVerifier(store *Store, baseDomain string, interval time.Duration, logger *slog.Logger) *DNSVerifier {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	return &DNSVerifier{
		store:      store,
		baseDomain: baseDomain,
		interval:   interval,
		logger:     logger.With("component", "dns_verifier"),
	}
}

func (v *DNSVerifier) Start() {
	v.ctx, v.cancel = context.WithCancel(context.Background())
	v.wg.Add(1)
	go v.run()
	v.logger.Info("dns verifier started", "interval", v.interval)
}

func (v *DNSVerifier) Stop() {
	if v.cancel != nil {
		v.cancel()
	}
	v.wg.Wait()
}

func (v *DNSVerifier) run() {
	defer v.wg.Done()

	ticker := time.NewTicker(v.interval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			v.checkDomains()
		}
	}
}

func (v *DNSVerifier) checkDomains() {
	// Find deployments with custom domains that need verification
	deployments, err := v.store.List(v.ctx, "deployments", []Filter{
		{Field: "status", Value: "running"},
	}, Page{Limit: 1000})
	if err != nil {
		v.logger.Error("failed to list deployments", "error", err)
		return
	}

	for _, depl := range deployments {
		_ = depl // DNS verification logic would go here
		// For now, auto domains work without verification
	}
}
