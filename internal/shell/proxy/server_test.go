package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProxyStore implements ProxyStore for testing.
type mockProxyStore struct {
	deployments map[string]*domain.Deployment // keyed by hostname
	nodeHosts   map[string]string             // node reference_id → ssh_host
}

func (m *mockProxyStore) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
	d, ok := m.deployments[hostname]
	if !ok {
		return nil, fmt.Errorf("deployment %s: %w", hostname, engine.ErrNotFound)
	}
	return d, nil
}

func (m *mockProxyStore) CountRoutableDeployments(ctx context.Context) (int, error) {
	count := 0
	for _, d := range m.deployments {
		if d.Status == domain.StatusRunning && d.ProxyPort > 0 {
			count++
		}
	}
	return count, nil
}

func (m *mockProxyStore) GetNodeSSHHost(ctx context.Context, nodeRefID string) (string, error) {
	host, ok := m.nodeHosts[nodeRefID]
	if !ok {
		return "", fmt.Errorf("node %s: %w", nodeRefID, engine.ErrNotFound)
	}
	return host, nil
}

func TestServer_ServeHTTP_Health(t *testing.T) {
	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{
			"app1.apps.test.io": {
				ReferenceID: "depl_1",
				Status:      domain.StatusRunning,
				ProxyPort:   30001,
			},
			"app2.apps.test.io": {
				ReferenceID: "depl_2",
				Status:      domain.StatusRunning,
				ProxyPort:   30002,
			},
			"app3.apps.test.io": {
				ReferenceID: "depl_3",
				Status:      domain.StatusStopped,
				ProxyPort:   30003,
			},
		},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://anything.apps.test.io/health", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
	assert.Contains(t, rec.Body.String(), `"deployments_routable":2`)
	assert.Contains(t, rec.Body.String(), `"base_domain":"apps.test.io"`)
}

func TestServer_ServeHTTP_NotFound(t *testing.T) {
	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://unknown.apps.test.io/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, 404, rec.Code)
	assert.Contains(t, rec.Body.String(), "App Not Found")
}

func TestServer_ServeHTTP_Stopped(t *testing.T) {
	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{
			"my-app.apps.test.io": {
				ReferenceID: "depl_123",
				NodeID:      "local",
				ProxyPort:   30001,
				Status:      domain.StatusStopped,
				CustomerID:  1,
			},
		},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://my-app.apps.test.io/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, 503, rec.Code)
	assert.Contains(t, rec.Body.String(), "App Stopped")
}

func TestServer_ServeHTTP_WrongDomain(t *testing.T) {
	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	// Request with wrong base domain
	req := httptest.NewRequest("GET", "http://my-app.other.io/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, 404, rec.Code)
}

func TestServer_ServeHTTP_RunningDeployment(t *testing.T) {
	// Start a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	// Extract port from backend URL
	// The backend runs on a random port, so we need to mock the deployment with that port
	// For this test, we'll verify the proxy tries to connect correctly
	// but since ports won't match, we test the error handling

	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{
			"my-app.apps.test.io": {
				ReferenceID: "depl_123",
				NodeID:      "local",
				ProxyPort:   30001, // Wrong port for test
				Status:      domain.StatusRunning,
				CustomerID:  1,
			},
		},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://my-app.apps.test.io/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	// Since the proxy port (30001) doesn't actually have a server,
	// we expect a 503 unavailable error
	assert.Equal(t, 503, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unavailable")
}

func TestServer_ServeHTTP_RemoteNode(t *testing.T) {
	// Start a test backend to simulate the remote node
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello from remote node"))
	}))
	defer backend.Close()

	// Extract host and port from backend URL
	backendURL := backend.URL // e.g., "http://127.0.0.1:PORT"
	parts := strings.SplitN(strings.TrimPrefix(backendURL, "http://"), ":", 2)
	backendHost := parts[0]
	backendPort := 0
	fmt.Sscanf(parts[1], "%d", &backendPort)

	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{
			"remote-app.apps.test.io": {
				ReferenceID: "depl_remote",
				NodeID:      "node_abc123",
				ProxyPort:   backendPort,
				Status:      domain.StatusRunning,
				CustomerID:  1,
			},
		},
		nodeHosts: map[string]string{
			"node_abc123": backendHost,
		},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://remote-app.apps.test.io/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "Hello from remote node", rec.Body.String())
}

func TestServer_ServeHTTP_RemoteNodeNotFound(t *testing.T) {
	ms := &mockProxyStore{
		deployments: map[string]*domain.Deployment{
			"orphan-app.apps.test.io": {
				ReferenceID: "depl_orphan",
				NodeID:      "node_missing",
				ProxyPort:   30001,
				Status:      domain.StatusRunning,
				CustomerID:  1,
			},
		},
		nodeHosts: map[string]string{},
	}

	cfg := Config{
		BaseDomain: "apps.test.io",
	}

	server, err := NewServer(cfg, ms, nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://orphan-app.apps.test.io/", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	// Node not found should result in unavailable error
	assert.Equal(t, 503, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unavailable")
}

func TestGetRealIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		want     string
	}{
		{
			name:     "X-Real-IP header",
			headers:  map[string]string{"X-Real-IP": "1.2.3.4"},
			remoteIP: "127.0.0.1:1234",
			want:     "1.2.3.4",
		},
		{
			name:     "X-Forwarded-For header",
			headers:  map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			remoteIP: "127.0.0.1:1234",
			want:     "1.2.3.4",
		},
		{
			name:     "X-Forwarded-For single IP",
			headers:  map[string]string{"X-Forwarded-For": "9.8.7.6"},
			remoteIP: "127.0.0.1:1234",
			want:     "9.8.7.6",
		},
		{
			name:     "fall back to remote address",
			headers:  map[string]string{},
			remoteIP: "192.168.1.1:5555",
			want:     "192.168.1.1:5555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteIP
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := getRealIP(req)
			assert.Equal(t, tt.want, got)
		})
	}
}
