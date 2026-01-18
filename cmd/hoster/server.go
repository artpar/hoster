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
	"github.com/artpar/hoster/internal/shell/store"
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
	store           store.Store
	docker          docker.Client
	billingReporter *billing.Reporter
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

	// Create HTTP handler using new JSON:API setup (ADR-003)
	handler := api.SetupAPI(api.APIConfig{
		Store:      s,
		Docker:     d,
		Logger:     logger,
		BaseDomain: cfg.Domain.BaseDomain,
		ConfigDir:  cfg.Domain.ConfigDir,
		// Auth configuration (ADR-005)
		AuthMode:         cfg.Auth.Mode,
		AuthRequire:      cfg.Auth.RequireAuth,
		AuthSharedSecret: cfg.Auth.SharedSecret,
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
		if cfg.Billing.APIGateURL != "" {
			billingClient = billing.NewAPIGateClient(billing.APIGateConfig{
				BaseURL: cfg.Billing.APIGateURL,
				APIKey:  cfg.Billing.APIKey,
			})
			logger.Info("billing enabled", "apigate_url", cfg.Billing.APIGateURL)
		} else {
			billingClient = billing.NewNoOpClient()
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

	return &Server{
		config:          cfg,
		httpServer:      httpServer,
		store:           s,
		docker:          d,
		billingReporter: billingReporter,
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

	// Start HTTP server in goroutine
	errCh := make(chan error, 1)
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

	// Stop billing reporter (F009: Billing Integration)
	if s.billingReporter != nil {
		s.billingReporter.Stop()
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
