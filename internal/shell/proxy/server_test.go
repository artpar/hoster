package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore implements store.Store for testing
type mockStore struct {
	deployments map[string]*domain.Deployment // keyed by hostname
	usedPorts   map[string][]int              // keyed by nodeID
}

func (m *mockStore) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
	d, ok := m.deployments[hostname]
	if !ok {
		return nil, store.NewStoreError("GetDeploymentByDomain", "deployment", hostname, "not found", store.ErrNotFound)
	}
	return d, nil
}

func (m *mockStore) GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error) {
	return m.usedPorts[nodeID], nil
}

// Implement remaining Store interface methods as no-ops for testing
func (m *mockStore) CreateTemplate(ctx context.Context, t *domain.Template) error       { return nil }
func (m *mockStore) GetTemplate(ctx context.Context, id string) (*domain.Template, error) { return nil, nil }
func (m *mockStore) GetTemplateBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	return nil, nil
}
func (m *mockStore) UpdateTemplate(ctx context.Context, t *domain.Template) error       { return nil }
func (m *mockStore) DeleteTemplate(ctx context.Context, id string) error                { return nil }
func (m *mockStore) ListTemplates(ctx context.Context, opts store.ListOptions) ([]domain.Template, error) {
	return nil, nil
}
func (m *mockStore) CreateDeployment(ctx context.Context, d *domain.Deployment) error { return nil }
func (m *mockStore) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	return nil, nil
}
func (m *mockStore) UpdateDeployment(ctx context.Context, d *domain.Deployment) error { return nil }
func (m *mockStore) DeleteDeployment(ctx context.Context, id string) error            { return nil }
func (m *mockStore) ListDeployments(ctx context.Context, opts store.ListOptions) ([]domain.Deployment, error) {
	return nil, nil
}
func (m *mockStore) ListDeploymentsByTemplate(ctx context.Context, templateID string, opts store.ListOptions) ([]domain.Deployment, error) {
	return nil, nil
}
func (m *mockStore) ListDeploymentsByCustomer(ctx context.Context, customerID string, opts store.ListOptions) ([]domain.Deployment, error) {
	return nil, nil
}
func (m *mockStore) CreateUsageEvent(ctx context.Context, e *domain.MeterEvent) error { return nil }
func (m *mockStore) GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error) {
	return nil, nil
}
func (m *mockStore) MarkEventsReported(ctx context.Context, ids []string, t time.Time) error {
	return nil
}
func (m *mockStore) CreateContainerEvent(ctx context.Context, e *domain.ContainerEvent) error {
	return nil
}
func (m *mockStore) GetContainerEvents(ctx context.Context, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error) {
	return nil, nil
}
func (m *mockStore) CreateNode(ctx context.Context, n *domain.Node) error              { return nil }
func (m *mockStore) GetNode(ctx context.Context, id string) (*domain.Node, error)      { return nil, nil }
func (m *mockStore) UpdateNode(ctx context.Context, n *domain.Node) error              { return nil }
func (m *mockStore) DeleteNode(ctx context.Context, id string) error                   { return nil }
func (m *mockStore) ListNodesByCreator(ctx context.Context, creatorID string, opts store.ListOptions) ([]domain.Node, error) {
	return nil, nil
}
func (m *mockStore) ListOnlineNodes(ctx context.Context) ([]domain.Node, error)    { return nil, nil }
func (m *mockStore) ListCheckableNodes(ctx context.Context) ([]domain.Node, error) { return nil, nil }
func (m *mockStore) CreateSSHKey(ctx context.Context, k *domain.SSHKey) error      { return nil }
func (m *mockStore) GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error) {
	return nil, nil
}
func (m *mockStore) DeleteSSHKey(ctx context.Context, id string) error { return nil }
func (m *mockStore) ListSSHKeysByCreator(ctx context.Context, creatorID string, opts store.ListOptions) ([]domain.SSHKey, error) {
	return nil, nil
}
func (m *mockStore) WithTx(ctx context.Context, fn func(store.Store) error) error { return fn(m) }
func (m *mockStore) Close() error                                                 { return nil }

func TestServer_ServeHTTP_NotFound(t *testing.T) {
	ms := &mockStore{
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
	ms := &mockStore{
		deployments: map[string]*domain.Deployment{
			"my-app.apps.test.io": {
				ID:         "depl_123",
				NodeID:     "local",
				ProxyPort:  30001,
				Status:     domain.StatusStopped,
				CustomerID: "user_1",
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
	ms := &mockStore{
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

	ms := &mockStore{
		deployments: map[string]*domain.Deployment{
			"my-app.apps.test.io": {
				ID:         "depl_123",
				NodeID:     "local",
				ProxyPort:  30001, // Wrong port for test
				Status:     domain.StatusRunning,
				CustomerID: "user_1",
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
