// Package proxy implements the App Proxy HTTP server that routes incoming
// requests to deployed containers based on hostname.
package proxy

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/proxy"
	"github.com/artpar/hoster/internal/shell/store"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Config holds proxy server configuration.
type Config struct {
	Address      string        // Listen address, e.g., "0.0.0.0:9091"
	BaseDomain   string        // Base domain for apps, e.g., "apps.hoster.io"
	ReadTimeout  time.Duration // HTTP read timeout
	WriteTimeout time.Duration // HTTP write timeout
	IdleTimeout  time.Duration // HTTP idle timeout
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Address:      "0.0.0.0:9091",
		BaseDomain:   "apps.localhost",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// Server is the HTTP server that handles app routing.
type Server struct {
	store   store.Store
	parser  proxy.HostnameParser
	logger  *slog.Logger
	config  Config
	errTmpl *template.Template
}

// NewServer creates a new proxy server.
func NewServer(cfg Config, s store.Store, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Parse error templates
	errTmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Server{
		store:   s,
		parser:  proxy.HostnameParser{BaseDomain: cfg.BaseDomain},
		logger:  logger,
		config:  cfg,
		errTmpl: errTmpl,
	}, nil
}

// Start starts the proxy server (non-blocking).
func (s *Server) Start() *http.Server {
	srv := &http.Server{
		Addr:         s.config.Address,
		Handler:      s,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	go func() {
		s.logger.Info("starting app proxy server",
			"address", s.config.Address,
			"base_domain", s.config.BaseDomain,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("proxy server error", "error", err)
		}
	}()

	return srv
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hostname := r.Host

	// Health endpoint - responds regardless of hostname
	if r.URL.Path == "/health" && r.Method == http.MethodGet {
		s.serveHealth(w, r)
		return
	}

	// Strip port from hostname for matching (browsers include port in Host header)
	hostnameWithoutPort := hostname
	if idx := strings.LastIndex(hostname, ":"); idx != -1 {
		hostnameWithoutPort = hostname[:idx]
	}

	s.logger.Debug("proxy request",
		"hostname", hostname,
		"hostname_stripped", hostnameWithoutPort,
		"path", r.URL.Path,
		"method", r.Method,
	)

	// 1. Parse hostname to extract slug (base domain pattern)
	slug, ok := s.parser.Parse(hostname)

	// 2. Resolve target from database
	var target proxy.ProxyTarget
	var err error
	if ok {
		// Base domain match: resolve by parsed hostname
		target, err = s.resolveTarget(ctx, slug, hostnameWithoutPort)
	} else {
		// Custom domain fallback: try direct hostname lookup
		target, err = s.resolveTarget(ctx, "", hostnameWithoutPort)
	}
	if err != nil {
		var proxyErr proxy.ProxyError
		if errors.As(err, &proxyErr) {
			s.serveError(w, r, proxyErr)
			return
		}
		s.logger.Error("failed to resolve target", "hostname", hostname, "error", err)
		s.serveError(w, r, proxy.NewUnavailableError(hostname))
		return
	}

	// 3. Check if routable
	if !target.CanRoute() {
		s.serveError(w, r, proxy.NewStoppedError(hostname))
		return
	}

	// 4. Get upstream URL
	upstreamURL, err := s.getUpstreamURL(ctx, target)
	if err != nil {
		s.logger.Error("failed to get upstream URL", "hostname", hostname, "error", err)
		s.serveError(w, r, proxy.NewUnavailableError(hostname))
		return
	}

	// 5. Proxy the request
	s.proxyRequest(w, r, upstreamURL, target)
}

func (s *Server) resolveTarget(ctx context.Context, slug, hostname string) (proxy.ProxyTarget, error) {
	// Query database for deployment by domain hostname
	deployment, err := s.store.GetDeploymentByDomain(ctx, hostname)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return proxy.ProxyTarget{}, proxy.NewNotFoundError(hostname)
		}
		return proxy.ProxyTarget{}, err
	}

	// For custom domains, check that the domain is verified
	for _, d := range deployment.Domains {
		if strings.EqualFold(d.Hostname, hostname) && d.Type == domain.DomainTypeCustom {
			if d.VerificationStatus != domain.DomainVerificationVerified {
				return proxy.ProxyTarget{}, proxy.NewVerificationPendingError(hostname)
			}
			break
		}
	}

	return proxy.ProxyTarget{
		DeploymentID: deployment.ReferenceID,
		NodeID:       deployment.NodeID,
		Port:         deployment.ProxyPort,
		Status:       string(deployment.Status),
		CustomerID:   fmt.Sprintf("%d", deployment.CustomerID),
	}, nil
}

func (s *Server) getUpstreamURL(ctx context.Context, target proxy.ProxyTarget) (*url.URL, error) {
	if target.IsLocal() {
		return url.Parse("http://" + target.LocalAddress())
	}

	// For remote nodes, we would use SSH tunneling via NodePool
	// For now, only support local deployments
	return nil, errors.New("remote node proxying not implemented")
}

func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request, upstream *url.URL, target proxy.ProxyTarget) {
	reverseProxy := httputil.NewSingleHostReverseProxy(upstream)

	// Customize director to set proper headers
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Real-IP", getRealIP(r))
		req.Header.Set("X-Deployment-ID", target.DeploymentID)
	}

	// Handle errors
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Error("proxy error",
			"hostname", r.Host,
			"deployment", target.DeploymentID,
			"error", err,
		)
		s.serveError(w, r, proxy.NewUnavailableError(r.Host))
	}

	reverseProxy.ServeHTTP(w, r)
}

func (s *Server) serveError(w http.ResponseWriter, r *http.Request, err proxy.ProxyError) {
	s.logger.Warn("proxy error",
		"type", err.Type,
		"hostname", err.Hostname,
		"status", err.StatusCode,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(err.StatusCode)

	// Select template based on error type
	var tmplName string
	switch err.Type {
	case proxy.ErrorNotFound:
		tmplName = "not_found.html"
	case proxy.ErrorStopped:
		tmplName = "stopped.html"
	case proxy.ErrorVerificationPending:
		tmplName = "verification_pending.html"
	default:
		tmplName = "unavailable.html"
	}

	data := map[string]interface{}{
		"Hostname": err.Hostname,
		"Message":  err.Message,
	}

	if execErr := s.errTmpl.ExecuteTemplate(w, tmplName, data); execErr != nil {
		s.logger.Error("failed to execute error template", "error", execErr)
		http.Error(w, err.Message, err.StatusCode)
	}
}

// getRealIP extracts the real client IP from the request.
func getRealIP(r *http.Request) string {
	// Check X-Real-IP header first (from upstream proxy)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Fall back to remote address
	return r.RemoteAddr
}

// HealthResponse is the JSON response for the health endpoint.
type HealthResponse struct {
	Status               string `json:"status"`
	DeploymentsRoutable  int    `json:"deployments_routable"`
	BaseDomain           string `json:"base_domain"`
}

// serveHealth handles the /health endpoint for APIGate health checks.
func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Count routable deployments (running with proxy port assigned)
	count, err := s.store.CountRoutableDeployments(ctx)
	if err != nil {
		s.logger.Error("failed to count routable deployments", "error", err)
		// Still return healthy but with 0 count
		count = 0
	}

	resp := HealthResponse{
		Status:              "ok",
		DeploymentsRoutable: count,
		BaseDomain:          s.config.BaseDomain,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
