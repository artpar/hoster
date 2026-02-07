package workers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	coredns "github.com/artpar/hoster/internal/core/dns"
	"github.com/artpar/hoster/internal/core/domain"
	shelldns "github.com/artpar/hoster/internal/shell/dns"
	"github.com/artpar/hoster/internal/shell/store"
)

// DNSVerifierConfig configures the DNS verification worker.
type DNSVerifierConfig struct {
	Interval      time.Duration
	MaxConcurrent int
	BaseDomain    string
}

// DefaultDNSVerifierConfig returns default configuration.
func DefaultDNSVerifierConfig() DNSVerifierConfig {
	return DNSVerifierConfig{
		Interval:      60 * time.Second,
		MaxConcurrent: 5,
	}
}

// DNSVerifier polls for unverified custom domains and verifies them.
type DNSVerifier struct {
	store      store.Store
	resolver   *shelldns.Resolver
	config     DNSVerifierConfig
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewDNSVerifier creates a new DNS verification worker.
func NewDNSVerifier(s store.Store, config DNSVerifierConfig, logger *slog.Logger) *DNSVerifier {
	if config.Interval == 0 {
		config.Interval = 60 * time.Second
	}
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 5
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &DNSVerifier{
		store:    s,
		resolver: shelldns.NewResolver(),
		config:   config,
		logger:   logger.With("component", "dns_verifier"),
	}
}

// Start begins the DNS verifier background goroutine.
func (v *DNSVerifier) Start() {
	v.ctx, v.cancel = context.WithCancel(context.Background())
	v.wg.Add(1)
	go v.run()
	v.logger.Info("DNS verifier started", "interval", v.config.Interval)
}

// Stop gracefully stops the DNS verifier.
func (v *DNSVerifier) Stop() {
	if v.cancel != nil {
		v.cancel()
	}
	v.wg.Wait()
	v.logger.Info("DNS verifier stopped")
}

func (v *DNSVerifier) run() {
	defer v.wg.Done()

	// Run after a short delay on start
	time.Sleep(10 * time.Second)
	v.runCycle()

	ticker := time.NewTicker(v.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			v.runCycle()
		}
	}
}

func (v *DNSVerifier) runCycle() {
	ctx, cancel := context.WithTimeout(v.ctx, 2*time.Minute)
	defer cancel()

	// Get all deployments and check for unverified custom domains
	deployments, err := v.store.ListDeployments(ctx, store.ListOptions{Limit: 1000})
	if err != nil {
		v.logger.Error("failed to list deployments for DNS verification", "error", err)
		return
	}

	var toVerify []verifyTask
	for i := range deployments {
		d := &deployments[i]
		for j := range d.Domains {
			dom := &d.Domains[j]
			if dom.Type == domain.DomainTypeCustom && dom.VerificationStatus != domain.DomainVerificationVerified {
				toVerify = append(toVerify, verifyTask{deployment: d, domainIdx: j})
			}
		}
	}

	if len(toVerify) == 0 {
		return
	}

	v.logger.Debug("verifying custom domains", "count", len(toVerify))

	sem := make(chan struct{}, v.config.MaxConcurrent)
	var wg sync.WaitGroup

	for _, task := range toVerify {
		wg.Add(1)
		go func(t verifyTask) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			v.verifyDomain(ctx, t)
		}(task)
	}

	wg.Wait()
}

type verifyTask struct {
	deployment *domain.Deployment
	domainIdx  int
}

func (v *DNSVerifier) verifyDomain(ctx context.Context, task verifyTask) {
	d := task.deployment
	dom := &d.Domains[task.domainIdx]

	// Find the auto domain for this deployment
	autoDomain := coredns.FindAutoDomain(d.Domains)

	// Get node IP for A record verification
	var nodeIP string
	if d.NodeID != "" {
		node, err := v.store.GetNode(ctx, d.NodeID)
		if err == nil {
			nodeIP = node.SSHHost
		}
	}

	// Resolve DNS
	input := v.resolver.Resolve(ctx, dom.Hostname)

	// Run pure verification
	var expectedIPs []string
	if nodeIP != "" {
		expectedIPs = []string{nodeIP}
	}
	result := coredns.Verify(input, autoDomain, expectedIPs)

	// Update domain status
	if result.Verified {
		dom.VerificationStatus = domain.DomainVerificationVerified
		dom.VerificationMethod = result.Method
		now := time.Now()
		dom.VerifiedAt = &now
		dom.LastCheckError = ""
		v.logger.Info("domain verified",
			"deployment", d.ID, "hostname", dom.Hostname, "method", result.Method)
	} else {
		dom.VerificationStatus = domain.DomainVerificationFailed
		dom.LastCheckError = result.Error
	}

	if err := v.store.UpdateDeployment(ctx, d); err != nil {
		v.logger.Error("failed to update deployment domain status",
			"deployment", d.ID, "hostname", dom.Hostname, "error", err)
	}
}
