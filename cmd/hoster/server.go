package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/artpar/hoster/internal/shell/api"
	"github.com/artpar/hoster/internal/shell/billing"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/proxy"
	"github.com/artpar/hoster/internal/shell/scheduler"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/artpar/hoster/internal/shell/workers"
)

// =============================================================================
// Exit Codes
// =============================================================================

const (
	ExitSuccess          = 0
	ExitConfigError      = 1
	ExitDatabaseError    = 2
	ExitDockerError      = 3
	ExitHTTPServerError  = 4
)

// =============================================================================
// Server
// =============================================================================

// Server represents the Hoster application server.
type Server struct {
	config          *Config
	httpServer      *http.Server
	proxyServer     *http.Server
	store           store.Store
	docker          docker.Client
	nodePool        *docker.NodePool
	billingReporter *billing.Reporter
	healthChecker   *workers.HealthChecker
	provisioner     *workers.Provisioner
	dnsVerifier     *workers.DNSVerifier
	logger          *slog.Logger
}

// NewServer creates a new server with the given config.
func NewServer(cfg *Config, logger *slog.Logger) (*Server, error) {
	// Connect to database
	s, err := store.NewSQLiteStore(cfg.Database.DSN)
	if err != nil {
		return nil, &ServerError{
			Op:       "NewServer",
			Err:      err,
			ExitCode: ExitDatabaseError,
		}
	}

	// Connect to Docker
	d, err := docker.NewDockerClient(cfg.Docker.Host)
	if err != nil {
		s.Close()
		return nil, &ServerError{
			Op:       "NewServer",
			Err:      err,
			ExitCode: ExitDockerError,
		}
	}

	// Verify Docker connection
	if err := d.Ping(); err != nil {
		s.Close()
		d.Close()
		return nil, &ServerError{
			Op:       "NewServer",
			Err:      err,
			ExitCode: ExitDockerError,
		}
	}

	// Initialize encryption key (needed for SSH keys, cloud credentials, etc.)
	var encryptionKey []byte
	if cfg.Nodes.EncryptionKey != "" {
		encryptionKey = []byte(cfg.Nodes.EncryptionKey)
		if len(encryptionKey) != 32 {
			s.Close()
			d.Close()
			return nil, &ServerError{
				Op:       "NewServer",
				Err:      errors.New("nodes.encryption_key must be exactly 32 bytes for AES-256-GCM"),
				ExitCode: ExitConfigError,
			}
		}
	}

	// Create NodePool and health checker if encryption key is configured
	var nodePool *docker.NodePool
	var healthChecker *workers.HealthChecker

	if encryptionKey != nil {
		nodePool = docker.NewNodePool(s, encryptionKey, docker.DefaultNodePoolConfig())

		healthChecker = workers.NewHealthChecker(s, nodePool, encryptionKey, workers.HealthCheckerConfig{
			Interval:      cfg.Nodes.HealthCheckInterval,
			NodeTimeout:   cfg.Nodes.HealthCheckTimeout,
			MaxConcurrent: cfg.Nodes.HealthCheckMaxConcurrent,
		}, logger)

		logger.Info("remote nodes enabled",
			"health_check_interval", cfg.Nodes.HealthCheckInterval,
		)
	}

	// Create provisioner worker for cloud provider provisioning
	var provisioner *workers.Provisioner
	if encryptionKey != nil {
		provisioner = workers.NewProvisioner(s, encryptionKey, workers.DefaultProvisionerConfig(), logger)
	}

	// Create DNS verifier worker for custom domain verification
	dnsVerifier := workers.NewDNSVerifier(s, workers.DNSVerifierConfig{
		BaseDomain: cfg.Domain.BaseDomain,
	}, logger)

	// Create scheduler service for node selection
	sched := scheduler.NewService(s, nodePool, d, logger)

	// Create HTTP handler using new JSON:API setup (ADR-003)
	handler := api.SetupAPI(api.APIConfig{
		Store:      s,
		Docker:     d,
		Scheduler:  sched,
		Logger:     logger,
		BaseDomain: cfg.Domain.BaseDomain,
		ConfigDir:  cfg.Domain.ConfigDir,
		// Auth configuration (ADR-005)
		AuthSharedSecret: cfg.Auth.SharedSecret,
		// Encryption key for SSH keys (required for node management)
		EncryptionKey: encryptionKey,
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Create billing reporter (F009: Billing Integration)
	var billingReporter *billing.Reporter
	if cfg.Billing.Enabled {
		var billingClient billing.Client
		apiGateURL := cfg.Billing.APIGateURL
		apiKey := cfg.Billing.APIKey

		if apiGateURL != "" {
			billingClient = billing.NewAPIGateClient(billing.Config{
				BaseURL:    apiGateURL,
				ServiceKey: apiKey,
			}, logger)
			logger.Info("billing enabled", "apigate_url", apiGateURL)
		} else {
			billingClient = billing.NewNoopClient(logger)
			logger.Warn("billing enabled but no APIGate URL configured, using no-op client")
		}

		billingReporter = billing.NewReporter(billing.ReporterConfig{
			Store:     s,
			Client:    billingClient,
			Interval:  cfg.Billing.ReportInterval,
			BatchSize: cfg.Billing.BatchSize,
			Logger:    logger,
		})
	} else {
		logger.Info("billing disabled")
	}

	// Create App Proxy server (specs/domain/proxy.md)
	var proxyServer *http.Server
	if cfg.Proxy.Enabled {
		proxyHandler, err := proxy.NewServer(proxy.Config{
			Address:      cfg.Proxy.Address(),
			BaseDomain:   cfg.Proxy.BaseDomain,
			ReadTimeout:  cfg.Proxy.ReadTimeout,
			WriteTimeout: cfg.Proxy.WriteTimeout,
			IdleTimeout:  cfg.Proxy.IdleTimeout,
		}, s, logger)
		if err != nil {
			s.Close()
			d.Close()
			return nil, &ServerError{
				Op:       "NewServer",
				Err:      err,
				ExitCode: ExitConfigError,
			}
		}

		proxyServer = &http.Server{
			Addr:         cfg.Proxy.Address(),
			Handler:      proxyHandler,
			ReadTimeout:  cfg.Proxy.ReadTimeout,
			WriteTimeout: cfg.Proxy.WriteTimeout,
			IdleTimeout:  cfg.Proxy.IdleTimeout,
		}

		logger.Info("app proxy enabled",
			"address", cfg.Proxy.Address(),
			"base_domain", cfg.Proxy.BaseDomain,
		)
	} else {
		logger.Info("app proxy disabled")
	}

	return &Server{
		config:          cfg,
		httpServer:      httpServer,
		proxyServer:     proxyServer,
		store:           s,
		docker:          d,
		nodePool:        nodePool,
		billingReporter: billingReporter,
		healthChecker:   healthChecker,
		provisioner:     provisioner,
		dnsVerifier:     dnsVerifier,
		logger:          logger,
	}, nil
}

// Start starts the server and blocks until shutdown.
func (s *Server) Start(ctx context.Context) error {
	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start billing reporter in background (F009: Billing Integration)
	if s.billingReporter != nil {
		go s.billingReporter.Start(ctx)
	}

	// Start health checker in background (Creator Worker Nodes Phase 7)
	if s.healthChecker != nil {
		s.healthChecker.Start()
	}

	// Start cloud provisioner worker
	if s.provisioner != nil {
		s.provisioner.Start()
	}

	// Start DNS verifier worker
	if s.dnsVerifier != nil {
		s.dnsVerifier.Start()
	}

	// Start App Proxy server in goroutine (specs/domain/proxy.md)
	errCh := make(chan error, 2)
	if s.proxyServer != nil {
		go func() {
			s.logger.Info("starting App Proxy server",
				"address", s.config.Proxy.Address(),
				"base_domain", s.config.Proxy.BaseDomain)
			if err := s.proxyServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()
	}

	// Start HTTP server in goroutine
	go func() {
		s.logger.Info("starting HTTP server",
			"address", s.config.Server.Address())
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigCh:
		s.logger.Info("received shutdown signal", "signal", sig)
	case err := <-errCh:
		return &ServerError{
			Op:       "Start",
			Err:      err,
			ExitCode: ExitHTTPServerError,
		}
	case <-ctx.Done():
		s.logger.Info("context cancelled")
	}

	return s.Shutdown(context.Background())
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("initiating graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
	}

	// Shutdown App Proxy server (specs/domain/proxy.md)
	if s.proxyServer != nil {
		if err := s.proxyServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("App Proxy server shutdown error", "error", err)
		}
	}

	// Stop billing reporter (F009: Billing Integration)
	if s.billingReporter != nil {
		s.billingReporter.Stop()
	}

	// Stop health checker (Creator Worker Nodes Phase 7)
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}

	// Stop cloud provisioner worker
	if s.provisioner != nil {
		s.provisioner.Stop()
	}

	// Stop DNS verifier worker
	if s.dnsVerifier != nil {
		s.dnsVerifier.Stop()
	}

	// Close node pool connections
	if s.nodePool != nil {
		if err := s.nodePool.CloseAll(); err != nil {
			s.logger.Error("node pool close error", "error", err)
		}
	}

	// Close Docker client
	if err := s.docker.Close(); err != nil {
		s.logger.Error("Docker client close error", "error", err)
	}

	// Close database
	if err := s.store.Close(); err != nil {
		s.logger.Error("database close error", "error", err)
	}

	s.logger.Info("shutdown complete")
	return nil
}

// =============================================================================
// Server Error
// =============================================================================

// ServerError represents an error during server operation.
type ServerError struct {
	Op       string
	Err      error
	ExitCode int
}

func (e *ServerError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

func (e *ServerError) Unwrap() error {
	return e.Err
}
