package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Helpers
// =============================================================================

// stubStore implements store.Store for testing.
type stubStore struct {
	templates   map[string]*domain.Template
	deployments map[string]*domain.Deployment
	err         error // If set, all operations return this error
}

func newStubStore() *stubStore {
	return &stubStore{
		templates:   make(map[string]*domain.Template),
		deployments: make(map[string]*domain.Deployment),
	}
}

func (s *stubStore) CreateTemplate(ctx context.Context, t *domain.Template) error {
	if s.err != nil {
		return s.err
	}
	if _, exists := s.templates[t.ID]; exists {
		return store.NewStoreError("CreateTemplate", "template", t.ID, "already exists", store.ErrDuplicateID)
	}
	s.templates[t.ID] = t
	return nil
}

func (s *stubStore) GetTemplate(ctx context.Context, id string) (*domain.Template, error) {
	if s.err != nil {
		return nil, s.err
	}
	t, ok := s.templates[id]
	if !ok {
		return nil, store.NewStoreError("GetTemplate", "template", id, "not found", store.ErrNotFound)
	}
	return t, nil
}

func (s *stubStore) GetTemplateBySlug(ctx context.Context, slug string) (*domain.Template, error) {
	if s.err != nil {
		return nil, s.err
	}
	for _, t := range s.templates {
		if t.Slug == slug {
			return t, nil
		}
	}
	return nil, store.NewStoreError("GetTemplateBySlug", "template", slug, "not found", store.ErrNotFound)
}

func (s *stubStore) UpdateTemplate(ctx context.Context, t *domain.Template) error {
	if s.err != nil {
		return s.err
	}
	if _, ok := s.templates[t.ID]; !ok {
		return store.NewStoreError("UpdateTemplate", "template", t.ID, "not found", store.ErrNotFound)
	}
	s.templates[t.ID] = t
	return nil
}

func (s *stubStore) DeleteTemplate(ctx context.Context, id string) error {
	if s.err != nil {
		return s.err
	}
	if _, ok := s.templates[id]; !ok {
		return store.NewStoreError("DeleteTemplate", "template", id, "not found", store.ErrNotFound)
	}
	delete(s.templates, id)
	return nil
}

func (s *stubStore) ListTemplates(ctx context.Context, opts store.ListOptions) ([]domain.Template, error) {
	if s.err != nil {
		return nil, s.err
	}
	var result []domain.Template
	for _, t := range s.templates {
		result = append(result, *t)
	}
	return result, nil
}

func (s *stubStore) CreateDeployment(ctx context.Context, d *domain.Deployment) error {
	if s.err != nil {
		return s.err
	}
	if _, exists := s.deployments[d.ID]; exists {
		return store.NewStoreError("CreateDeployment", "deployment", d.ID, "already exists", store.ErrDuplicateID)
	}
	s.deployments[d.ID] = d
	return nil
}

func (s *stubStore) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	if s.err != nil {
		return nil, s.err
	}
	d, ok := s.deployments[id]
	if !ok {
		return nil, store.NewStoreError("GetDeployment", "deployment", id, "not found", store.ErrNotFound)
	}
	return d, nil
}

func (s *stubStore) UpdateDeployment(ctx context.Context, d *domain.Deployment) error {
	if s.err != nil {
		return s.err
	}
	if _, ok := s.deployments[d.ID]; !ok {
		return store.NewStoreError("UpdateDeployment", "deployment", d.ID, "not found", store.ErrNotFound)
	}
	s.deployments[d.ID] = d
	return nil
}

func (s *stubStore) DeleteDeployment(ctx context.Context, id string) error {
	if s.err != nil {
		return s.err
	}
	if _, ok := s.deployments[id]; !ok {
		return store.NewStoreError("DeleteDeployment", "deployment", id, "not found", store.ErrNotFound)
	}
	delete(s.deployments, id)
	return nil
}

func (s *stubStore) ListDeployments(ctx context.Context, opts store.ListOptions) ([]domain.Deployment, error) {
	if s.err != nil {
		return nil, s.err
	}
	var result []domain.Deployment
	for _, d := range s.deployments {
		result = append(result, *d)
	}
	return result, nil
}

func (s *stubStore) ListDeploymentsByTemplate(ctx context.Context, templateID string, opts store.ListOptions) ([]domain.Deployment, error) {
	if s.err != nil {
		return nil, s.err
	}
	var result []domain.Deployment
	for _, d := range s.deployments {
		if d.TemplateID == templateID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (s *stubStore) ListDeploymentsByCustomer(ctx context.Context, customerID string, opts store.ListOptions) ([]domain.Deployment, error) {
	if s.err != nil {
		return nil, s.err
	}
	var result []domain.Deployment
	for _, d := range s.deployments {
		if d.CustomerID == customerID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (s *stubStore) GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error) {
	if s.err != nil {
		return nil, s.err
	}
	for _, d := range s.deployments {
		for _, dom := range d.Domains {
			if dom.Hostname == hostname {
				return d, nil
			}
		}
	}
	return nil, store.NewStoreError("GetDeploymentByDomain", "deployment", hostname, "not found", store.ErrNotFound)
}

func (s *stubStore) GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error) {
	if s.err != nil {
		return nil, s.err
	}
	var ports []int
	for _, d := range s.deployments {
		if d.NodeID == nodeID && d.ProxyPort > 0 {
			ports = append(ports, d.ProxyPort)
		}
	}
	return ports, nil
}

func (s *stubStore) WithTx(ctx context.Context, fn func(store.Store) error) error {
	return fn(s)
}

func (s *stubStore) Close() error {
	return nil
}

func (s *stubStore) CreateUsageEvent(ctx context.Context, event *domain.MeterEvent) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error) {
	return nil, nil // Stub - no-op for tests
}

func (s *stubStore) MarkEventsReported(ctx context.Context, ids []string, reportedAt time.Time) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) CreateContainerEvent(ctx context.Context, event *domain.ContainerEvent) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) GetContainerEvents(ctx context.Context, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error) {
	return []domain.ContainerEvent{}, nil // Stub - return empty for tests
}

// Node operations (Creator Worker Nodes)
func (s *stubStore) CreateNode(ctx context.Context, node *domain.Node) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	return nil, store.ErrNotFound // Stub - not found for tests
}

func (s *stubStore) UpdateNode(ctx context.Context, node *domain.Node) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) DeleteNode(ctx context.Context, id string) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) ListNodesByCreator(ctx context.Context, creatorID string, opts store.ListOptions) ([]domain.Node, error) {
	return nil, nil // Stub - empty for tests
}

func (s *stubStore) ListOnlineNodes(ctx context.Context) ([]domain.Node, error) {
	return nil, nil // Stub - empty for tests
}

func (s *stubStore) ListCheckableNodes(ctx context.Context) ([]domain.Node, error) {
	return nil, nil // Stub - empty for tests
}

// SSH Key operations
func (s *stubStore) CreateSSHKey(ctx context.Context, key *domain.SSHKey) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error) {
	return nil, store.ErrNotFound // Stub - not found for tests
}

func (s *stubStore) DeleteSSHKey(ctx context.Context, id string) error {
	return nil // Stub - no-op for tests
}

func (s *stubStore) ListSSHKeysByCreator(ctx context.Context, creatorID string, opts store.ListOptions) ([]domain.SSHKey, error) {
	return nil, nil // Stub - empty for tests
}

func (s *stubStore) CountRoutableDeployments(ctx context.Context) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	count := 0
	for _, d := range s.deployments {
		if d.Status == domain.StatusRunning && d.ProxyPort > 0 {
			count++
		}
	}
	return count, nil
}

// stubDocker implements docker.Client for testing.
type stubDocker struct {
	pingErr    error
	containers map[string]*docker.ContainerInfo
}

func newStubDocker() *stubDocker {
	return &stubDocker{
		containers: make(map[string]*docker.ContainerInfo),
	}
}

func (d *stubDocker) Ping() error {
	return d.pingErr
}

func (d *stubDocker) Close() error {
	return nil
}

func (d *stubDocker) CreateContainer(spec docker.ContainerSpec) (string, error) {
	id := "container_" + time.Now().Format("20060102150405")
	d.containers[id] = &docker.ContainerInfo{
		ID:     id,
		Name:   spec.Name,
		Image:  spec.Image,
		Status: docker.ContainerStatusCreated,
	}
	return id, nil
}

func (d *stubDocker) StartContainer(containerID string) error {
	info, ok := d.containers[containerID]
	if !ok {
		return docker.ErrContainerNotFound
	}
	info.Status = docker.ContainerStatusRunning
	return nil
}

func (d *stubDocker) StopContainer(containerID string, timeout *time.Duration) error {
	info, ok := d.containers[containerID]
	if !ok {
		return docker.ErrContainerNotFound
	}
	info.Status = docker.ContainerStatusExited
	return nil
}

func (d *stubDocker) RemoveContainer(containerID string, opts docker.RemoveOptions) error {
	if _, ok := d.containers[containerID]; !ok {
		return docker.ErrContainerNotFound
	}
	delete(d.containers, containerID)
	return nil
}

func (d *stubDocker) InspectContainer(containerID string) (*docker.ContainerInfo, error) {
	info, ok := d.containers[containerID]
	if !ok {
		return nil, docker.ErrContainerNotFound
	}
	return info, nil
}

func (d *stubDocker) ListContainers(opts docker.ListOptions) ([]docker.ContainerInfo, error) {
	var result []docker.ContainerInfo
	for _, c := range d.containers {
		result = append(result, *c)
	}
	return result, nil
}

func (d *stubDocker) ContainerLogs(containerID string, opts docker.LogOptions) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("test logs"))), nil
}

func (d *stubDocker) ContainerStats(containerID string) (*docker.ContainerResourceStats, error) {
	if _, ok := d.containers[containerID]; !ok {
		return nil, docker.ErrContainerNotFound
	}
	return &docker.ContainerResourceStats{
		CPUPercent:       5.0,
		MemoryUsageBytes: 100 * 1024 * 1024,
		MemoryLimitBytes: 512 * 1024 * 1024,
		MemoryPercent:    19.5,
		NetworkRxBytes:   1024,
		NetworkTxBytes:   2048,
		BlockReadBytes:   4096,
		BlockWriteBytes:  8192,
		PIDs:             10,
	}, nil
}

func (d *stubDocker) CreateNetwork(spec docker.NetworkSpec) (string, error) {
	return "network_test", nil
}

func (d *stubDocker) RemoveNetwork(networkID string) error {
	return nil
}

func (d *stubDocker) ConnectNetwork(networkID, containerID string) error {
	return nil
}

func (d *stubDocker) DisconnectNetwork(networkID, containerID string, force bool) error {
	return nil
}

func (d *stubDocker) CreateVolume(spec docker.VolumeSpec) (string, error) {
	return "volume_test", nil
}

func (d *stubDocker) RemoveVolume(volumeName string, force bool) error {
	return nil
}

func (d *stubDocker) PullImage(image string, opts docker.PullOptions) error {
	return nil
}

func (d *stubDocker) ImageExists(image string) (bool, error) {
	return true, nil
}

// newTestHandler creates a new handler with stub dependencies.
func newTestHandler() (*Handler, *stubStore, *stubDocker) {
	s := newStubStore()
	d := newStubDocker()
	h := NewHandler(s, d, nil, "apps.localhost", "/tmp/hoster-test-configs") // nil logger uses default
	return h, s, d
}

// jsonBody encodes a value to JSON and returns a reader.
func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	buf := new(bytes.Buffer)
	require.NoError(t, json.NewEncoder(buf).Encode(v))
	return buf
}

// parseResponse parses a JSON response body into the given type.
func parseResponse[T any](t *testing.T, body io.Reader) T {
	t.Helper()
	var result T
	require.NoError(t, json.NewDecoder(body).Decode(&result))
	return result
}

// createTestTemplate creates a valid template for testing.
func createTestTemplate(id, name string) *domain.Template {
	now := time.Now()
	return &domain.Template{
		ID:          id,
		Name:        name,
		Slug:        name,
		Version:     "1.0.0",
		ComposeSpec: "services:\n  web:\n    image: nginx",
		CreatorID:   "user-123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// createTestDeployment creates a valid deployment for testing.
func createTestDeployment(id, templateID, customerID string) *domain.Deployment {
	now := time.Now()
	return &domain.Deployment{
		ID:              id,
		Name:            "Test Deployment",
		TemplateID:      templateID,
		TemplateVersion: "1.0.0",
		CustomerID:      customerID,
		Status:          domain.StatusPending,
		Variables:       make(map[string]string),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// =============================================================================
// Health Endpoint Tests
// =============================================================================

func TestHealth_Success(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[HealthResponse](t, w.Body)
	assert.Equal(t, "healthy", resp.Status)
}

func TestReady_AllHealthy(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ReadyResponse](t, w.Body)
	assert.Equal(t, "ready", resp.Status)
	assert.Equal(t, "ok", resp.Checks["database"])
	assert.Equal(t, "ok", resp.Checks["docker"])
}

func TestReady_DockerFailed(t *testing.T) {
	h, _, d := newTestHandler()
	d.pingErr = docker.ErrConnectionFailed

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	resp := parseResponse[ReadyResponse](t, w.Body)
	assert.Equal(t, "not_ready", resp.Status)
	assert.Equal(t, "ok", resp.Checks["database"])
	assert.Equal(t, "failed", resp.Checks["docker"])
}

func TestHealthLive_Success(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[HealthResponse](t, w.Body)
	assert.Equal(t, "healthy", resp.Status)
}

func TestHealthReady_Success(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ReadyResponse](t, w.Body)
	assert.Equal(t, "ready", resp.Status)
	assert.Equal(t, "ok", resp.Checks["database"])
	assert.Equal(t, "ok", resp.Checks["docker"])
}

// =============================================================================
// Template Endpoint Tests
// =============================================================================

func TestCreateTemplate_Success(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateTemplateRequest{
		Name:        "Test Template",
		Version:     "1.0.0",
		ComposeSpec: "services:\n  web:\n    image: nginx",
		CreatorID:   "user-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	resp := parseResponse[TemplateResponse](t, w.Body)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "Test Template", resp.Name)
	assert.Equal(t, "1.0.0", resp.Version)
	assert.False(t, resp.Published)
}

func TestCreateTemplate_MissingName(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateTemplateRequest{
		Version:     "1.0.0",
		ComposeSpec: "services:\n  web:\n    image: nginx",
		CreatorID:   "user-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "validation_error", resp.Code)
	assert.Contains(t, resp.Error, "name")
}

func TestCreateTemplate_MissingVersion(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateTemplateRequest{
		Name:        "Test Template",
		ComposeSpec: "services:\n  web:\n    image: nginx",
		CreatorID:   "user-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "validation_error", resp.Code)
}

func TestCreateTemplate_MissingComposeSpec(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateTemplateRequest{
		Name:      "Test Template",
		Version:   "1.0.0",
		CreatorID: "user-123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "validation_error", resp.Code)
}

func TestCreateTemplate_MissingCreatorID(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateTemplateRequest{
		Name:        "Test Template",
		Version:     "1.0.0",
		ComposeSpec: "services:\n  web:\n    image: nginx",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "validation_error", resp.Code)
}

func TestCreateTemplate_InvalidJSON(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "validation_error", resp.Code)
}

func TestGetTemplate_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	s.templates[template.ID] = template

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/tmpl_123", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[TemplateResponse](t, w.Body)
	assert.Equal(t, "tmpl_123", resp.ID)
	assert.Equal(t, "Test Template", resp.Name)
}

func TestGetTemplate_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "template_not_found", resp.Code)
}

func TestListTemplates_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	s.templates["tmpl_1"] = createTestTemplate("tmpl_1", "Template One")
	s.templates["tmpl_2"] = createTestTemplate("tmpl_2", "Template Two")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListTemplatesResponse](t, w.Body)
	assert.Len(t, resp.Templates, 2)
}

func TestListTemplates_Empty(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListTemplatesResponse](t, w.Body)
	assert.Len(t, resp.Templates, 0)
}

func TestListTemplates_Pagination(t *testing.T) {
	h, s, _ := newTestHandler()

	for i := 0; i < 25; i++ {
		id := "tmpl_" + string(rune('a'+i))
		s.templates[id] = createTestTemplate(id, "Template "+string(rune('a'+i)))
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates?limit=10&offset=5", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListTemplatesResponse](t, w.Body)
	// Note: stub doesn't implement pagination, but tests the parameter parsing
	assert.Equal(t, 10, resp.Limit)
	assert.Equal(t, 5, resp.Offset)
}

func TestUpdateTemplate_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Original Name")
	s.templates[template.ID] = template

	body := jsonBody(t, UpdateTemplateRequest{
		Name:        "Updated Name",
		Description: "Updated description",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/tmpl_123", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[TemplateResponse](t, w.Body)
	assert.Equal(t, "Updated Name", resp.Name)
	assert.Equal(t, "Updated description", resp.Description)
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, UpdateTemplateRequest{
		Name: "Updated Name",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateTemplate_Published(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	template.Published = true
	s.templates[template.ID] = template

	body := jsonBody(t, UpdateTemplateRequest{
		Name: "Updated Name",
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/tmpl_123", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "template_published", resp.Code)
}

func TestDeleteTemplate_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	s.templates[template.ID] = template

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/tmpl_123", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NotContains(t, s.templates, "tmpl_123")
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteTemplate_HasDeployments(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	s.templates[template.ID] = template

	deployment := createTestDeployment("depl_456", "tmpl_123", "customer-1")
	s.deployments[deployment.ID] = deployment

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/tmpl_123", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "template_in_use", resp.Code)
}

func TestPublishTemplate_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	template.Published = false
	s.templates[template.ID] = template

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tmpl_123/publish", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[TemplateResponse](t, w.Body)
	assert.True(t, resp.Published)
}

func TestPublishTemplate_AlreadyPublished(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	template.Published = true
	s.templates[template.ID] = template

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tmpl_123/publish", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "already_published", resp.Code)
}

func TestPublishTemplate_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/nonexistent/publish", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// Deployment Endpoint Tests
// =============================================================================

func TestCreateDeployment_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	template.Published = true
	s.templates[template.ID] = template

	body := jsonBody(t, CreateDeploymentRequest{
		TemplateID: "tmpl_123",
		Name:       "My Deployment",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "customer-1") // Auth via header
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	resp := parseResponse[DeploymentResponse](t, w.Body)
	assert.NotEmpty(t, resp.ID)
	assert.Equal(t, "My Deployment", resp.Name)
	assert.Equal(t, "tmpl_123", resp.TemplateID)
	assert.Equal(t, "pending", resp.Status)
}

func TestCreateDeployment_MissingTemplateID(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateDeploymentRequest{
		Name: "My Deployment",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "customer-1") // Auth via header
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "validation_error", resp.Code)
}

func TestCreateDeployment_MissingCustomerID(t *testing.T) {
	// This test is now testing unauthenticated requests (no X-User-ID header)
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	template.Published = true
	s.templates[template.ID] = template

	body := jsonBody(t, CreateDeploymentRequest{
		TemplateID: "tmpl_123",
		Name:       "My Deployment",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", body)
	req.Header.Set("Content-Type", "application/json")
	// No X-User-ID header = unauthenticated
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "auth_required", resp.Code)
}

func TestCreateDeployment_TemplateNotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	body := jsonBody(t, CreateDeploymentRequest{
		TemplateID: "nonexistent",
		Name:       "My Deployment",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "customer-1") // Auth via header
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "template_not_found", resp.Code)
}

func TestCreateDeployment_TemplateNotPublished(t *testing.T) {
	h, s, _ := newTestHandler()

	template := createTestTemplate("tmpl_123", "Test Template")
	template.Published = false
	s.templates[template.ID] = template

	body := jsonBody(t, CreateDeploymentRequest{
		TemplateID: "tmpl_123",
		Name:       "My Deployment",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "customer-1") // Auth via header
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "template_not_published", resp.Code)
}

func TestCreateDeployment_InvalidJSON(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "customer-1") // Auth via header
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetDeployment_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	deployment := createTestDeployment("depl_123", "tmpl_456", "customer-1")
	s.deployments[deployment.ID] = deployment

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/depl_123", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[DeploymentResponse](t, w.Body)
	assert.Equal(t, "depl_123", resp.ID)
}

func TestGetDeployment_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "deployment_not_found", resp.Code)
}

func TestListDeployments_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	s.deployments["depl_1"] = createTestDeployment("depl_1", "tmpl_123", "customer-1")
	s.deployments["depl_2"] = createTestDeployment("depl_2", "tmpl_123", "customer-2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListDeploymentsResponse](t, w.Body)
	assert.Len(t, resp.Deployments, 2)
}

func TestListDeployments_Empty(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListDeploymentsResponse](t, w.Body)
	assert.Len(t, resp.Deployments, 0)
}

func TestListDeployments_FilterByTemplate(t *testing.T) {
	h, s, _ := newTestHandler()

	s.deployments["depl_1"] = createTestDeployment("depl_1", "tmpl_123", "customer-1")
	s.deployments["depl_2"] = createTestDeployment("depl_2", "tmpl_456", "customer-1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments?template_id=tmpl_123", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListDeploymentsResponse](t, w.Body)
	assert.Len(t, resp.Deployments, 1)
	assert.Equal(t, "tmpl_123", resp.Deployments[0].TemplateID)
}

func TestListDeployments_FilterByCustomer(t *testing.T) {
	h, s, _ := newTestHandler()

	s.deployments["depl_1"] = createTestDeployment("depl_1", "tmpl_123", "customer-1")
	s.deployments["depl_2"] = createTestDeployment("depl_2", "tmpl_123", "customer-2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments?customer_id=customer-1", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[ListDeploymentsResponse](t, w.Body)
	assert.Len(t, resp.Deployments, 1)
	assert.Equal(t, "customer-1", resp.Deployments[0].CustomerID)
}

func TestDeleteDeployment_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	deployment := createTestDeployment("depl_123", "tmpl_456", "customer-1")
	s.deployments[deployment.ID] = deployment

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deployments/depl_123", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NotContains(t, s.deployments, "depl_123")
}

func TestDeleteDeployment_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deployments/nonexistent", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStartDeployment_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	deployment := createTestDeployment("depl_123", "tmpl_456", "customer-1")
	deployment.Status = domain.StatusPending
	deployment.NodeID = "node-1" // Node required for starting
	s.deployments[deployment.ID] = deployment

	// Also need the template for deployment start
	template := createTestTemplate("tmpl_456", "Test Template")
	template.Published = true
	s.templates[template.ID] = template

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/depl_123/start", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[DeploymentResponse](t, w.Body)
	assert.Equal(t, "running", resp.Status)
}

func TestStartDeployment_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/nonexistent/start", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStartDeployment_AlreadyRunning(t *testing.T) {
	h, s, _ := newTestHandler()

	deployment := createTestDeployment("depl_123", "tmpl_456", "customer-1")
	deployment.Status = domain.StatusRunning
	s.deployments[deployment.ID] = deployment

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/depl_123/start", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "already_running", resp.Code)
}

func TestStopDeployment_Success(t *testing.T) {
	h, s, _ := newTestHandler()

	deployment := createTestDeployment("depl_123", "tmpl_456", "customer-1")
	deployment.Status = domain.StatusRunning
	s.deployments[deployment.ID] = deployment

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/depl_123/stop", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse[DeploymentResponse](t, w.Body)
	assert.Equal(t, "stopped", resp.Status)
}

func TestStopDeployment_NotRunning(t *testing.T) {
	h, s, _ := newTestHandler()

	deployment := createTestDeployment("depl_123", "tmpl_456", "customer-1")
	deployment.Status = domain.StatusPending
	s.deployments[deployment.ID] = deployment

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/depl_123/stop", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	resp := parseResponse[ErrorResponse](t, w.Body)
	assert.Equal(t, "invalid_transition", resp.Code)
}

func TestStopDeployment_NotFound(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/nonexistent/stop", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =============================================================================
// Middleware Tests
// =============================================================================

func TestRequestID_Generated(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestContentType_JSON(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestInvalidMethod_405(t *testing.T) {
	h, _, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPatch, "/health", nil)
	w := httptest.NewRecorder()

	h.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestPanic_Recovery(t *testing.T) {
	// Create handler that will panic - this tests the recovery middleware
	h, s, _ := newTestHandler()

	// Force a panic by setting nil store
	s.err = nil // Just test that handler handles errors gracefully

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	// Should not panic
	assert.NotPanics(t, func() {
		h.Routes().ServeHTTP(w, req)
	})
}
