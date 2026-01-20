package apigate

import (
	"context"
	"fmt"
	"log/slog"
)

// RegistrarConfig holds configuration for automatic route registration.
type RegistrarConfig struct {
	// APIGate connection
	APIGateURL string // e.g., "http://localhost:8082"
	APIKey     string // Admin API key

	// App Proxy configuration
	AppProxyURL        string // e.g., "http://localhost:9091"
	AppProxyBaseDomain string // e.g., "apps.localhost"

	// Hoster API configuration (optional, for registering API route)
	HosterAPIURL string // e.g., "http://localhost:8080"
}

// Registrar handles automatic registration of Hoster services with APIGate.
type Registrar struct {
	client *Client
	config RegistrarConfig
	logger *slog.Logger
}

// NewRegistrar creates a new registrar.
func NewRegistrar(cfg RegistrarConfig, logger *slog.Logger) *Registrar {
	if logger == nil {
		logger = slog.Default()
	}

	client := NewClient(Config{
		BaseURL: cfg.APIGateURL,
		APIKey:  cfg.APIKey,
	}, logger)

	return &Registrar{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// RegisterAppProxy registers the app proxy upstream and route with APIGate.
// This should be called during Hoster startup.
func (r *Registrar) RegisterAppProxy(ctx context.Context) error {
	if r.config.AppProxyURL == "" {
		return fmt.Errorf("app proxy URL not configured")
	}
	if r.config.AppProxyBaseDomain == "" {
		return fmt.Errorf("app proxy base domain not configured")
	}

	r.logger.Info("registering app proxy with APIGate",
		"apigate_url", r.config.APIGateURL,
		"app_proxy_url", r.config.AppProxyURL,
		"base_domain", r.config.AppProxyBaseDomain,
	)

	// 1. Create/update upstream for app proxy
	upstreamID, err := r.client.EnsureUpstream(ctx, Upstream{
		Name:            "hoster-app-proxy",
		BaseURL:         r.config.AppProxyURL,
		HealthCheckPath: "/health",
	})
	if err != nil {
		return fmt.Errorf("ensure app proxy upstream: %w", err)
	}

	r.logger.Info("app proxy upstream configured", "upstream_id", upstreamID)

	// 2. Create/update route with wildcard host pattern
	hostPattern := "*." + r.config.AppProxyBaseDomain
	err = r.client.EnsureRoute(ctx, Route{
		Name:          "hoster-app-proxy",
		HostPattern:   hostPattern,
		HostMatchType: "wildcard",
		PathPattern:   "/*",
		MatchType:     "prefix",
		UpstreamID:    upstreamID,
		Priority:      100,
		Enabled:       true,
	})
	if err != nil {
		return fmt.Errorf("ensure app proxy route: %w", err)
	}

	r.logger.Info("app proxy route configured",
		"host_pattern", hostPattern,
		"upstream_id", upstreamID,
	)

	return nil
}

// RegisterHosterAPI registers the Hoster API upstream and route with APIGate.
// This is optional - only needed if Hoster API should also go through APIGate.
// The route includes request transforms to inject auth headers from APIGate context.
func (r *Registrar) RegisterHosterAPI(ctx context.Context) error {
	if r.config.HosterAPIURL == "" {
		r.logger.Debug("skipping Hoster API registration - URL not configured")
		return nil
	}

	r.logger.Info("registering Hoster API with APIGate",
		"apigate_url", r.config.APIGateURL,
		"hoster_api_url", r.config.HosterAPIURL,
	)

	// 1. Create/update upstream for Hoster API
	upstreamID, err := r.client.EnsureUpstream(ctx, Upstream{
		Name:            "hoster-api",
		BaseURL:         r.config.HosterAPIURL,
		HealthCheckPath: "/health",
	})
	if err != nil {
		return fmt.Errorf("ensure hoster api upstream: %w", err)
	}

	r.logger.Info("hoster API upstream configured", "upstream_id", upstreamID)

	// 2. Create/update route for API with header injection
	// APIGate transform context provides: userID, planID, keyID
	// These are injected as X-User-ID, X-Plan-ID, X-Key-ID headers for Hoster auth middleware
	err = r.client.EnsureRoute(ctx, Route{
		Name:        "hoster-api",
		PathPattern: "/api/*",
		MatchType:   "prefix",
		UpstreamID:  upstreamID,
		Priority:    50,
		Enabled:     true,
		RequestTransform: &RequestTransform{
			SetHeaders: map[string]string{
				"X-User-ID": "userID", // User's unique identifier from APIGate auth
				"X-Plan-ID": "planID", // User's subscription plan ID
				"X-Key-ID":  "keyID",  // API key identifier used for authentication
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ensure hoster api route: %w", err)
	}

	r.logger.Info("hoster API route configured with header injection",
		"upstream_id", upstreamID,
		"headers", []string{"X-User-ID", "X-Plan-ID", "X-Key-ID"},
	)

	return nil
}

// RegisterAll registers all Hoster services with APIGate.
func (r *Registrar) RegisterAll(ctx context.Context) error {
	// Register app proxy (required if proxy is enabled)
	if r.config.AppProxyURL != "" {
		if err := r.RegisterAppProxy(ctx); err != nil {
			return fmt.Errorf("register app proxy: %w", err)
		}
	}

	// Register Hoster API (optional)
	if err := r.RegisterHosterAPI(ctx); err != nil {
		return fmt.Errorf("register hoster api: %w", err)
	}

	return nil
}
