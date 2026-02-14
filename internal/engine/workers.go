package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
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
		`SELECT cp.*, cc.credentials, cc.provider as cred_provider
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
	credEncrypted := row["credentials"]
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

	// Resolve SSH public key from the linked ssh_key
	sshKeyRefID := strVal(row["ssh_key_id"])
	var sshPublicKey string
	if sshKeyRefID != "" {
		sshKey, err := p.store.Get(ctx, "ssh_keys", sshKeyRefID)
		if err != nil {
			p.failProvision(ctx, refID, "lookup SSH key: "+err.Error())
			return
		}
		sshPublicKey = strVal(sshKey["public_key"])
	}
	if sshPublicKey == "" {
		p.failProvision(ctx, refID, "SSH public key is empty — cannot provision without an SSH key")
		return
	}

	// Create instance
	result, err := prov.CreateInstance(ctx, provider.ProvisionRequest{
		InstanceName: instanceName,
		Region:       region,
		Size:         size,
		SSHPublicKey: sshPublicKey,
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
	publicIP := strVal(row["public_ip"])

	// Wait for SSH port to be reachable before creating the node.
	// New cloud instances take 30-90s for SSH to accept connections after the
	// provider API reports the instance as active. Use a short dial timeout
	// so we don't block the provisioner cycle — we'll retry next cycle (5s).
	// Fail after 5 minutes to avoid stuck provisions.
	if created, ok := row["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			if time.Since(t) > 5*time.Minute {
				p.failProvision(ctx, refID, "SSH not reachable after 5 minutes on "+publicIP+":22")
				return
			}
		}
	}

	conn, err := net.DialTimeout("tcp", publicIP+":22", 3*time.Second)
	if err != nil {
		p.logger.Debug("SSH not yet reachable, will retry next cycle", "provision", refID, "ip", publicIP)
		p.store.Update(ctx, "cloud_provisions", refID, map[string]any{
			"current_step": "waiting_for_ssh",
		})
		return // Stay in configuring, retry on next 5s cycle
	}
	conn.Close()
	p.logger.Info("SSH reachable", "provision", refID, "ip", publicIP)

	// Resolve ssh_key_id (SoftRefField = reference_id) → integer PK for node's RefField
	sshKeyRefID := strVal(row["ssh_key_id"])
	var sshKeyIntID int
	if sshKeyRefID != "" {
		sshKey, err := p.store.Get(ctx, "ssh_keys", sshKeyRefID)
		if err != nil {
			p.failProvision(ctx, refID, "lookup SSH key for node: "+err.Error())
			return
		}
		if id, ok := toInt64(sshKey["id"]); ok {
			sshKeyIntID = int(id)
		}
	}

	creatorID, _ := toInt64(row["creator_id"])
	instanceName := strVal(row["instance_name"])
	providerType := strVal(row["provider"])

	// Create node entry from the completed provision
	nodeRow, err := p.store.Create(ctx, "nodes", map[string]any{
		"name":          instanceName,
		"ssh_host":      publicIP,
		"ssh_port":      22,
		"ssh_user":      "root",
		"ssh_key_id":    sshKeyIntID,
		"creator_id":    int(creatorID),
		"provider_type": providerType,
		"provision_id":  refID,
		"status":        "online",
		"docker_socket": "/var/run/docker.sock",
	})
	if err != nil {
		p.failProvision(ctx, refID, "create node: "+err.Error())
		return
	}

	nodeRefID := strVal(nodeRow["reference_id"])

	// Transition provision to ready
	now := time.Now().UTC().Format(time.RFC3339)
	p.store.Update(ctx, "cloud_provisions", refID, map[string]any{
		"current_step": "ready",
		"completed_at": now,
		"node_id":      nodeRefID,
	})
	p.store.Transition(ctx, "cloud_provisions", refID, "ready")

	p.logger.Info("provision ready", "provision", refID, "node", nodeRefID)
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
	credEncrypted := row["credentials"]
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

// =============================================================================
// Invoice Generator
// =============================================================================

// InvoiceGenerator periodically creates/updates invoices for users with running deployments.
type InvoiceGenerator struct {
	store    *Store
	interval time.Duration
	logger   *slog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewInvoiceGenerator(store *Store, interval time.Duration, logger *slog.Logger) *InvoiceGenerator {
	if interval == 0 {
		interval = 24 * time.Hour
	}
	return &InvoiceGenerator{
		store:    store,
		interval: interval,
		logger:   logger.With("component", "invoice_generator"),
	}
}

func (ig *InvoiceGenerator) Start() {
	ig.ctx, ig.cancel = context.WithCancel(context.Background())
	ig.wg.Add(1)
	go ig.run()
	ig.logger.Info("invoice generator started", "interval", ig.interval)
}

func (ig *InvoiceGenerator) Stop() {
	if ig.cancel != nil {
		ig.cancel()
	}
	ig.wg.Wait()
}

func (ig *InvoiceGenerator) run() {
	defer ig.wg.Done()
	ig.generateAll()

	ticker := time.NewTicker(ig.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ig.ctx.Done():
			return
		case <-ticker.C:
			ig.generateAll()
		}
	}
}

// timeToYearMonth extracts "YYYY-MM" from a DB value that may be string or time.Time.
func timeToYearMonth(v any) string {
	switch t := v.(type) {
	case time.Time:
		return t.UTC().Format("2006-01")
	case string:
		if len(t) >= 7 {
			return t[:7]
		}
	case []byte:
		if len(t) >= 7 {
			return string(t[:7])
		}
	}
	return ""
}

func (ig *InvoiceGenerator) generateAll() {
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)
	periodStartStr := periodStart.Format(time.RFC3339)
	currentYM := periodStart.Format("2006-01") // "2026-02"

	// Get all running deployments
	deployments, err := ig.store.List(ig.ctx, "deployments", []Filter{
		{Field: "status", Value: "running"},
	}, Page{Limit: 1000})
	if err != nil {
		ig.logger.Error("failed to list deployments", "error", err)
		return
	}

	if len(deployments) == 0 {
		return
	}

	// Group running deployments by owner (customer_id)
	type lineItem struct {
		DeploymentID   string `json:"deployment_id"`
		DeploymentName string `json:"deployment_name"`
		TemplateName   string `json:"template_name"`
		MonthlyCents   int    `json:"monthly_cents"`
		Description    string `json:"description"`
	}

	type userBill struct {
		items      []lineItem
		totalCents int
	}

	bills := map[int]*userBill{}

	for _, d := range deployments {
		ownerID, _ := toInt64(d["customer_id"])
		if ownerID == 0 {
			continue
		}

		var priceCents int
		var templateName string
		if tmplID, ok := toInt64(d["template_id"]); ok && tmplID > 0 {
			tmpl, err := ig.store.GetByID(ig.ctx, "templates", int(tmplID))
			if err == nil {
				if p, ok := toInt64(tmpl["price_monthly_cents"]); ok {
					priceCents = int(p)
				}
				templateName = strVal(tmpl["name"])
			}
		}

		uid := int(ownerID)
		if bills[uid] == nil {
			bills[uid] = &userBill{}
		}

		deplName := strVal(d["name"])
		bills[uid].items = append(bills[uid].items, lineItem{
			DeploymentID:   strVal(d["reference_id"]),
			DeploymentName: deplName,
			TemplateName:   templateName,
			MonthlyCents:   priceCents,
			Description:    fmt.Sprintf("%s (%s) — %s", deplName, templateName, periodStart.Format("Jan 2006")),
		})
		bills[uid].totalCents += priceCents
	}

	// Get existing invoices for this period (match by year-month to avoid format issues)
	allInvoices, err := ig.store.List(ig.ctx, "invoices", nil, Page{Limit: 1000})
	if err != nil {
		ig.logger.Error("failed to list invoices", "error", err)
		return
	}

	existingByUser := map[int]map[string]any{}
	for _, inv := range allInvoices {
		if timeToYearMonth(inv["period_start"]) == currentYM {
			ownerID, _ := toInt64(inv["user_id"])
			uid := int(ownerID)
			// Prefer paid/pending over draft (don't overwrite settled invoices)
			if prev, exists := existingByUser[uid]; exists {
				prevStatus, _ := prev["status"].(string)
				if prevStatus == "paid" || prevStatus == "pending" {
					continue
				}
			}
			existingByUser[uid] = inv
		}
	}

	// Create or update invoices per user
	for userID, bill := range bills {
		if bill.totalCents == 0 {
			continue
		}

		itemsJSON, _ := json.Marshal(bill.items)
		existing := existingByUser[userID]

		if existing != nil {
			status, _ := existing["status"].(string)
			if status == "paid" || status == "pending" {
				continue // already paid or in payment flow
			}
			// Update draft with latest costs
			refID := strVal(existing["reference_id"])
			ig.store.Update(ig.ctx, "invoices", refID, map[string]any{
				"items":          string(itemsJSON),
				"subtotal_cents": bill.totalCents,
				"total_cents":    bill.totalCents,
			})
			ig.logger.Debug("updated invoice", "invoice", refID, "user_id", userID, "total_cents", bill.totalCents)
		} else {
			// Create new invoice
			row, err := ig.store.Create(ig.ctx, "invoices", map[string]any{
				"user_id":        userID,
				"period_start":   periodStartStr,
				"period_end":     periodEnd.Format(time.RFC3339),
				"items":          string(itemsJSON),
				"subtotal_cents": bill.totalCents,
				"tax_cents":      0,
				"total_cents":    bill.totalCents,
				"currency":       "USD",
			})
			if err != nil {
				ig.logger.Error("failed to create invoice", "error", err, "user_id", userID)
				continue
			}
			ig.logger.Info("created invoice", "invoice", strVal(row["reference_id"]), "user_id", userID, "total_cents", bill.totalCents)
		}
	}
}
