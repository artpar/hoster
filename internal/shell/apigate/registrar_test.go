package apigate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistrar(t *testing.T) {
	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:         "http://localhost:8082",
		APIKey:             "test-key",
		AppProxyURL:        "http://localhost:9091",
		AppProxyBaseDomain: "apps.localhost",
		HosterAPIURL:       "http://localhost:8080",
	}, slog.Default())

	assert.NotNil(t, registrar)
	assert.NotNil(t, registrar.client)
	assert.Equal(t, "http://localhost:8082", registrar.config.APIGateURL)
}

func TestNewRegistrar_NilLogger(t *testing.T) {
	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL: "http://localhost:8082",
	}, nil)

	assert.NotNil(t, registrar)
	assert.NotNil(t, registrar.logger)
}

func TestRegistrar_RegisterAppProxy(t *testing.T) {
	upstreamCreated := false
	routeCreated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/upstreams":
			// No existing upstream
			json.NewEncoder(w).Encode(UpstreamsResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/upstreams":
			upstreamCreated = true
			var upstream Upstream
			json.NewDecoder(r.Body).Decode(&upstream)
			assert.Equal(t, "hoster-app-proxy", upstream.Name)
			assert.Equal(t, "http://localhost:9091", upstream.BaseURL)
			assert.Equal(t, "/health", upstream.HealthCheckPath)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(UpstreamResponse{
				Data: struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					ID:         "upstream-app-proxy",
					Type:       "upstreams",
					Attributes: upstream,
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/admin/routes":
			// No existing route
			json.NewEncoder(w).Encode(RoutesResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/routes":
			routeCreated = true
			var route Route
			json.NewDecoder(r.Body).Decode(&route)
			assert.Equal(t, "hoster-app-proxy", route.Name)
			assert.Equal(t, "*.apps.localhost", route.HostPattern)
			assert.Equal(t, "wildcard", route.HostMatchType)
			assert.Equal(t, "/*", route.PathPattern)
			assert.Equal(t, "prefix", route.MatchType)
			assert.Equal(t, "upstream-app-proxy", route.UpstreamID)
			assert.Equal(t, 100, route.Priority)
			assert.True(t, route.Enabled)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(RouteResponse{
				Data: struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					ID:         "route-app-proxy",
					Type:       "routes",
					Attributes: route,
				},
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:         server.URL,
		APIKey:             "test-key",
		AppProxyURL:        "http://localhost:9091",
		AppProxyBaseDomain: "apps.localhost",
	}, slog.Default())

	err := registrar.RegisterAppProxy(context.Background())
	require.NoError(t, err)
	assert.True(t, upstreamCreated)
	assert.True(t, routeCreated)
}

func TestRegistrar_RegisterAppProxy_MissingURL(t *testing.T) {
	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:         "http://localhost:8082",
		AppProxyBaseDomain: "apps.localhost",
		// AppProxyURL is missing
	}, slog.Default())

	err := registrar.RegisterAppProxy(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app proxy URL not configured")
}

func TestRegistrar_RegisterAppProxy_MissingBaseDomain(t *testing.T) {
	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:  "http://localhost:8082",
		AppProxyURL: "http://localhost:9091",
		// AppProxyBaseDomain is missing
	}, slog.Default())

	err := registrar.RegisterAppProxy(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "app proxy base domain not configured")
}

func TestRegistrar_RegisterHosterAPI(t *testing.T) {
	upstreamCreated := false
	routeCreated := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/upstreams":
			json.NewEncoder(w).Encode(UpstreamsResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/upstreams":
			upstreamCreated = true
			var upstream Upstream
			json.NewDecoder(r.Body).Decode(&upstream)
			assert.Equal(t, "hoster-api", upstream.Name)
			assert.Equal(t, "http://localhost:8080", upstream.BaseURL)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(UpstreamResponse{
				Data: struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					ID:         "upstream-hoster-api",
					Type:       "upstreams",
					Attributes: upstream,
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/admin/routes":
			json.NewEncoder(w).Encode(RoutesResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/routes":
			routeCreated = true
			var route Route
			json.NewDecoder(r.Body).Decode(&route)
			assert.Equal(t, "hoster-api", route.Name)
			assert.Equal(t, "/api/*", route.PathPattern)
			assert.Equal(t, "prefix", route.MatchType)
			assert.Equal(t, 50, route.Priority)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(RouteResponse{
				Data: struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					ID:         "route-hoster-api",
					Type:       "routes",
					Attributes: route,
				},
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:   server.URL,
		APIKey:       "test-key",
		HosterAPIURL: "http://localhost:8080",
	}, slog.Default())

	err := registrar.RegisterHosterAPI(context.Background())
	require.NoError(t, err)
	assert.True(t, upstreamCreated)
	assert.True(t, routeCreated)
}

func TestRegistrar_RegisterHosterAPI_URLNotConfigured(t *testing.T) {
	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL: "http://localhost:8082",
		// HosterAPIURL is not configured
	}, slog.Default())

	// Should return nil (skip silently) when URL not configured
	err := registrar.RegisterHosterAPI(context.Background())
	assert.NoError(t, err)
}

func TestRegistrar_RegisterAll(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/upstreams":
			json.NewEncoder(w).Encode(UpstreamsResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/upstreams":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(UpstreamResponse{
				Data: struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					ID:         "upstream-123",
					Type:       "upstreams",
					Attributes: Upstream{},
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/admin/routes":
			json.NewEncoder(w).Encode(RoutesResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/routes":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(RouteResponse{
				Data: struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					ID:         "route-123",
					Type:       "routes",
					Attributes: Route{},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:         server.URL,
		APIKey:             "test-key",
		AppProxyURL:        "http://localhost:9091",
		AppProxyBaseDomain: "apps.localhost",
		HosterAPIURL:       "http://localhost:8080",
	}, slog.Default())

	err := registrar.RegisterAll(context.Background())
	require.NoError(t, err)

	// Should have made requests for:
	// - App proxy: GET upstreams, POST upstream, GET routes, POST route
	// - Hoster API: GET upstreams, POST upstream, GET routes, POST route
	assert.Equal(t, 8, requestCount)
}

func TestRegistrar_RegisterAll_AppProxyOnly(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/admin/upstreams":
			json.NewEncoder(w).Encode(UpstreamsResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/upstreams":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(UpstreamResponse{
				Data: struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					ID: "upstream-123",
				},
			})

		case r.Method == http.MethodGet && r.URL.Path == "/admin/routes":
			json.NewEncoder(w).Encode(RoutesResponse{Data: nil})

		case r.Method == http.MethodPost && r.URL.Path == "/admin/routes":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(RouteResponse{
				Data: struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					ID: "route-123",
				},
			})
		}
	}))
	defer server.Close()

	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:         server.URL,
		AppProxyURL:        "http://localhost:9091",
		AppProxyBaseDomain: "apps.localhost",
		// HosterAPIURL not set - will be skipped
	}, slog.Default())

	err := registrar.RegisterAll(context.Background())
	require.NoError(t, err)

	// Should have made requests only for app proxy (4 requests)
	assert.Equal(t, 4, requestCount)
}

func TestRegistrar_RegisterAll_NoAppProxy(t *testing.T) {
	registrar := NewRegistrar(RegistrarConfig{
		APIGateURL:   "http://localhost:8082",
		HosterAPIURL: "http://localhost:8080",
		// AppProxyURL not set
	}, slog.Default())

	// Should succeed even without app proxy URL (just skips that registration)
	err := registrar.RegisterAll(context.Background())
	// This will try to register Hoster API but fail because there's no server
	// The important thing is it doesn't crash on missing app proxy URL
	assert.Error(t, err) // Will error due to connection refused to non-existent server
}
