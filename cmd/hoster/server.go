package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/artpar/hoster/internal/engine"
	"github.com/artpar/hoster/internal/shell/billing"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/proxy"
)

// =============================================================================
// Exit Codes
// =============================================================================

const (
	ExitSuccess         = 0
	ExitConfigError     = 1
	ExitDatabaseError   = 2
	ExitHTTPServerError = 3
)

// =============================================================================
// Server
// =============================================================================

// Server represents the Hoster application server.
type Server struct {
	config          *Config
	httpServer      *http.Server
	proxyServer     *http.Server
	store           *engine.Store
	nodePool        *docker.NodePool
	billingReporter *billing.Reporter
	healthChecker   *engine.HealthChecker
	provisioner     *engine.Provisioner
	dnsVerifier     *engine.DNSVerifier
	logger          *slog.Logger
}

// NewServer creates a new server with the given config.
func NewServer(cfg *Config, logger *slog.Logger) (*Server, error) {
	// Open database and run migrations via engine
	store, err := engine.OpenDB(cfg.Database.DSN, engine.Schema(), logger)
	if err != nil {
		return nil, &ServerError{
			Op:       "NewServer",
			Err:      err,
			ExitCode: ExitDatabaseError,
		}
	}

	// Initialize encryption key (needed for SSH keys, cloud credentials, etc.)
	var encryptionKey []byte
	if cfg.Nodes.EncryptionKey != "" {
		encryptionKey = []byte(cfg.Nodes.EncryptionKey)
		if len(encryptionKey) != 32 {
			store.Close()
			return nil, &ServerError{
				Op:       "NewServer",
				Err:      errors.New("nodes.encryption_key must be exactly 32 bytes for AES-256-GCM"),
				ExitCode: ExitConfigError,
			}
		}
	}

	// Create NodePool and health checker if encryption key is configured
	var nodePool *docker.NodePool
	var healthChecker *engine.HealthChecker

	if encryptionKey != nil {
		nodePool = docker.NewNodePool(store, encryptionKey, docker.DefaultNodePoolConfig())

		healthChecker = engine.NewHealthChecker(store, nodePool, encryptionKey, 0, logger)

		logger.Info("remote nodes enabled",
			"health_check_interval", cfg.Nodes.HealthCheckInterval,
		)
	}

	// Create provisioner worker for cloud provider provisioning
	var provisioner *engine.Provisioner
	if encryptionKey != nil {
		provisioner = engine.NewProvisioner(store, encryptionKey, 0, logger)
		if healthChecker != nil {
			provisioner.SetHealthChecker(healthChecker)
		}
	}

	// Create DNS verifier worker for custom domain verification
	dnsVerifier := engine.NewDNSVerifier(store, cfg.Domain.BaseDomain, 0, logger)

	// Create command bus and register handlers
	bus := engine.NewBus(store, logger)
	engine.RegisterHandlers(bus)

	// Set extra dependencies for command handlers
	if nodePool != nil {
		bus.SetExtra("node_pool", nodePool)
	}
	bus.SetExtra("base_domain", cfg.Domain.BaseDomain)
	bus.SetExtra("config_dir", cfg.Domain.ConfigDir)

	// Create HTTP handler using the engine
	handler := engine.Setup(engine.SetupConfig{
		Store:         store,
		Bus:           bus,
		Logger:        logger,
		BaseDomain:    cfg.Domain.BaseDomain,
		ConfigDir:     cfg.Domain.ConfigDir,
		SharedSecret:  cfg.Auth.SharedSecret,
		EncryptionKey: encryptionKey,
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Create billing reporter — always enabled
	var billingClient billing.Client
	if cfg.Billing.APIGateURL != "" {
		billingClient = billing.NewAPIGateClient(billing.Config{
			BaseURL:    cfg.Billing.APIGateURL,
			ServiceKey: cfg.Billing.APIKey,
		}, logger)
		logger.Info("billing enabled", "apigate_url", cfg.Billing.APIGateURL)
	} else {
		billingClient = billing.NewNoopClient(logger)
		logger.Warn("billing: no APIGate URL configured, using no-op client")
	}

	billingReporter := billing.NewReporter(billing.ReporterConfig{
		Store:     store,
		Client:    billingClient,
		Interval:  cfg.Billing.ReportInterval,
		BatchSize: cfg.Billing.BatchSize,
		Logger:    logger,
	})

	// Create App Proxy server (specs/domain/proxy.md)
	var proxyHTTPServer *http.Server
	if cfg.Proxy.Enabled {
		proxyHandler, err := proxy.NewServer(proxy.Config{
			Address:      cfg.Proxy.Address(),
			BaseDomain:   cfg.Proxy.BaseDomain,
			ReadTimeout:  cfg.Proxy.ReadTimeout,
			WriteTimeout: cfg.Proxy.WriteTimeout,
			IdleTimeout:  cfg.Proxy.IdleTimeout,
		}, store, logger)
		if err != nil {
			store.Close()
			return nil, &ServerError{
				Op:       "NewServer",
				Err:      err,
				ExitCode: ExitConfigError,
			}
		}

		proxyHTTPServer = &http.Server{
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
		proxyServer:     proxyHTTPServer,
		store:           store,
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

	// Start billing reporter in background
	go s.billingReporter.Start(ctx)

	// Start health checker in background
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

	// Start App Proxy server in goroutine
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

	// Shutdown App Proxy server
	if s.proxyServer != nil {
		if err := s.proxyServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("App Proxy server shutdown error", "error", err)
		}
	}

	// Stop billing reporter
	s.billingReporter.Stop()

	// Stop health checker
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
