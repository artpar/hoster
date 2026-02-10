// Package e2e provides end-to-end tests for Hoster.
//
// These tests require a running Docker daemon and will create/destroy
// real containers. Run with:
//
//	go test -v -timeout 10m ./tests/e2e/...
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/shell/api"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
)

// =============================================================================
// Test Globals
// =============================================================================

var (
	testStore  store.Store
	testDocker docker.Client
	testClient *http.Client
	baseURL    string
	testServer *http.Server
)

// =============================================================================
// TestMain Setup
// =============================================================================

func TestMain(m *testing.M) {
	// Setup
	code := setup()
	if code != 0 {
		os.Exit(code)
	}

	// Run tests
	result := m.Run()

	// Teardown
	teardown()

	os.Exit(result)
}

func setup() int {
	log.Println("E2E Setup: Initializing test environment...")

	// 1. Create temp database
	tmpDir, err := os.MkdirTemp("", "hoster_e2e_")
	if err != nil {
		log.Printf("Failed to create temp dir: %v", err)
		return 1
	}
	tmpDB := filepath.Join(tmpDir, "test.db")
	log.Printf("E2E Setup: Using database: %s", tmpDB)

	// 2. Create SQLite store
	s, err := store.NewSQLiteStore(tmpDB)
	if err != nil {
		log.Printf("Failed to create store: %v", err)
		return 1
	}
	testStore = s
	log.Println("E2E Setup: SQLite store initialized")

	// 3. Create Docker client
	d, err := docker.NewDockerClient("")
	if err != nil {
		log.Printf("Failed to create Docker client: %v", err)
		return 1
	}
	testDocker = d
	log.Println("E2E Setup: Docker client created")

	// 4. Verify Docker connection
	if err := d.Ping(); err != nil {
		log.Printf("Failed to ping Docker: %v", err)
		log.Println("Make sure Docker daemon is running")
		return 1
	}
	log.Println("E2E Setup: Docker daemon is reachable")

	// 5. Cleanup any leftover test containers
	log.Println("E2E Setup: Cleaning up any leftover test containers...")
	if err := CleanupAllTestResources(context.Background(), d); err != nil {
		log.Printf("WARN: Failed to cleanup old containers: %v", err)
	}

	// 6. Create HTTP handler
	handler := api.SetupAPI(api.APIConfig{
		Store:      testStore,
		Docker:     testDocker,
		BaseDomain: "apps.localhost",
		ConfigDir:  tmpDir + "/configs",
	})
	log.Println("E2E Setup: HTTP handler created")

	// 7. Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Printf("Failed to find available port: %v", err)
		return 1
	}
	port := listener.Addr().(*net.TCPAddr).Port
	baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	log.Printf("E2E Setup: Server will listen on port %d", port)

	// 8. Create HTTP server
	testServer = &http.Server{
		Handler: handler,
	}

	// 9. Start server in goroutine
	go func() {
		if err := testServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()
	log.Println("E2E Setup: HTTP server started")

	// 10. Create HTTP client
	testClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// 11. Wait for server to be ready
	if err := waitForReady(baseURL+"/health", 10*time.Second); err != nil {
		log.Printf("Server failed to become ready: %v", err)
		return 1
	}
	log.Println("E2E Setup: Server is ready")

	log.Println("E2E Setup: Complete!")
	return 0
}

func teardown() {
	log.Println("E2E Teardown: Cleaning up...")

	// 1. Shutdown HTTP server
	if testServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		testServer.Shutdown(ctx)
		log.Println("E2E Teardown: HTTP server stopped")
	}

	// 2. Cleanup test containers
	if testDocker != nil {
		CleanupAllTestResources(context.Background(), testDocker)
		testDocker.Close()
		log.Println("E2E Teardown: Docker client closed")
	}

	// 3. Close database
	if testStore != nil {
		testStore.Close()
		log.Println("E2E Teardown: Database closed")
	}

	log.Println("E2E Teardown: Complete!")
}

// waitForReady polls the health endpoint until it responds.
func waitForReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server not ready after %v", timeout)
}

// =============================================================================
// JSON:API Envelope Types
// =============================================================================

// jsonAPIRequest is the JSON:API request envelope.
type jsonAPIRequest struct {
	Data jsonAPIResourceObj `json:"data"`
}

// jsonAPIResourceObj is a single JSON:API resource object in a response.
type jsonAPIResourceObj struct {
	Type       string         `json:"type"`
	ID         string         `json:"id,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// jsonAPISingleResponse is a JSON:API response with a single resource.
type jsonAPISingleResponse struct {
	Data json.RawMessage `json:"data"`
}

// jsonAPIListResponse is a JSON:API response with a list of resources.
type jsonAPIListResponse struct {
	Data json.RawMessage `json:"data"`
}

// =============================================================================
// API Client Helpers
// =============================================================================

// TemplateResponse represents a template from the API.
type TemplateResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Version     string         `json:"version"`
	ComposeSpec string         `json:"compose_spec"`
	Published   bool           `json:"published"`
	Variables   []VariableResp `json:"variables"`
	CreatorID   string         `json:"creator_id"`
}

type VariableResp struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Default  string `json:"default"`
}

// DeploymentResponse represents a deployment from the API.
type DeploymentResponse struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	TemplateID      string            `json:"template_id"`
	TemplateVersion string            `json:"template_version"`
	CustomerID      string            `json:"customer_id"`
	Status          string            `json:"status"`
	Variables       map[string]string `json:"variables"`
	ErrorMessage    string            `json:"error_message,omitempty"`
}

// parseJSONAPIResource extracts a typed struct from a JSON:API resource object.
// api2go marshals as: {"id":"...", "type":"...", "field1":"...", "field2":"..."} (flat, not nested under attributes)
func parseJSONAPIResource[T any](raw json.RawMessage) *T {
	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return &result
}

// jsonAPIBody builds a JSON:API create request body.
func jsonAPIBody(resourceType string, attrs map[string]any) []byte {
	body := map[string]any{
		"data": map[string]any{
			"type":       resourceType,
			"attributes": attrs,
		},
	}
	b, _ := json.Marshal(body)
	return b
}

// doJSONAPIRequest performs an HTTP request with JSON:API content type.
func doJSONAPIRequest(t *testing.T, method, url string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := testClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s failed: %v", method, url, err)
	}
	return resp
}

// CreateTemplate creates a template via the API (JSON:API format).
func CreateTemplate(t *testing.T, name, version, composeSpec string) *TemplateResponse {
	t.Helper()

	body := jsonAPIBody("templates", map[string]any{
		"name":         name,
		"version":      version,
		"compose_spec": composeSpec,
	})

	headers := map[string]string{"X-User-ID": "test-creator"}
	resp := doJSONAPIRequest(t, "POST", baseURL+"/api/v1/templates", body, headers)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to create template: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var envelope jsonAPISingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("Failed to decode template response: %v", err)
	}

	result := parseJSONAPIResource[TemplateResponse](envelope.Data)
	if result == nil {
		t.Fatal("Failed to parse template from JSON:API response")
	}

	t.Logf("Created template: %s (%s)", result.Name, result.ID)
	return result
}

// PublishTemplate publishes a template via the API.
func PublishTemplate(t *testing.T, templateID string) {
	t.Helper()

	headers := map[string]string{"X-User-ID": "test-creator"}
	resp := doJSONAPIRequest(t, "POST", baseURL+"/api/v1/templates/"+templateID+"/publish", nil, headers)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to publish template: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	t.Logf("Published template: %s", templateID)
}

// GetTemplate gets a template by ID.
func GetTemplate(t *testing.T, templateID string) *TemplateResponse {
	t.Helper()

	resp := doJSONAPIRequest(t, "GET", baseURL+"/api/v1/templates/"+templateID, nil, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to get template: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var envelope jsonAPISingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("Failed to decode template response: %v", err)
	}

	result := parseJSONAPIResource[TemplateResponse](envelope.Data)
	if result == nil {
		t.Fatal("Failed to parse template from JSON:API response")
	}
	return result
}

// ListTemplates lists all templates.
func ListTemplates(t *testing.T) []TemplateResponse {
	t.Helper()

	resp := doJSONAPIRequest(t, "GET", baseURL+"/api/v1/templates", nil, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to list templates: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var envelope jsonAPIListResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("Failed to decode templates response: %v", err)
	}

	var items []TemplateResponse
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		// May be null for empty list
		return []TemplateResponse{}
	}
	return items
}

// CreateDeployment creates a deployment via the API (JSON:API format).
// Uses "test-user" as the authenticated user ID (sent via X-User-ID header).
func CreateDeployment(t *testing.T, templateID string, variables map[string]string) *DeploymentResponse {
	t.Helper()

	attrs := map[string]any{
		"template_id": templateID,
	}
	if variables != nil {
		attrs["variables"] = variables
	}
	body := jsonAPIBody("deployments", attrs)

	headers := map[string]string{"X-User-ID": "test-user"}
	resp := doJSONAPIRequest(t, "POST", baseURL+"/api/v1/deployments", body, headers)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to create deployment: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var envelope jsonAPISingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("Failed to decode deployment response: %v", err)
	}

	result := parseJSONAPIResource[DeploymentResponse](envelope.Data)
	if result == nil {
		t.Fatal("Failed to parse deployment from JSON:API response")
	}

	t.Logf("Created deployment: %s (status=%s)", result.ID, result.Status)
	return result
}

// GetDeployment gets a deployment by ID.
func GetDeployment(t *testing.T, deploymentID string) *DeploymentResponse {
	t.Helper()

	resp := doJSONAPIRequest(t, "GET", baseURL+"/api/v1/deployments/"+deploymentID, nil, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to get deployment: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var envelope jsonAPISingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("Failed to decode deployment response: %v", err)
	}

	result := parseJSONAPIResource[DeploymentResponse](envelope.Data)
	if result == nil {
		t.Fatal("Failed to parse deployment from JSON:API response")
	}
	return result
}

// StartDeployment starts a deployment via the API.
func StartDeployment(t *testing.T, deploymentID string) *DeploymentResponse {
	t.Helper()

	resp := doJSONAPIRequest(t, "POST", baseURL+"/api/v1/deployments/"+deploymentID+"/start", nil, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to start deployment: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	// Custom action returns wrapped response, not pure JSON:API
	var result DeploymentResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	// Try parsing as {"data": {...}} first
	var envelope jsonAPISingleResponse
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && len(envelope.Data) > 0 {
		r := parseJSONAPIResource[DeploymentResponse](envelope.Data)
		if r != nil {
			result = *r
		}
	} else {
		json.Unmarshal(bodyBytes, &result)
	}

	t.Logf("Started deployment: %s (status=%s)", result.ID, result.Status)
	return &result
}

// StopDeployment stops a deployment via the API.
func StopDeployment(t *testing.T, deploymentID string) *DeploymentResponse {
	t.Helper()

	resp := doJSONAPIRequest(t, "POST", baseURL+"/api/v1/deployments/"+deploymentID+"/stop", nil, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to stop deployment: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var result DeploymentResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	var envelope jsonAPISingleResponse
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && len(envelope.Data) > 0 {
		r := parseJSONAPIResource[DeploymentResponse](envelope.Data)
		if r != nil {
			result = *r
		}
	} else {
		json.Unmarshal(bodyBytes, &result)
	}

	t.Logf("Stopped deployment: %s (status=%s)", result.ID, result.Status)
	return &result
}

// DeleteDeployment deletes a deployment via the API.
func DeleteDeployment(t *testing.T, deploymentID string) {
	t.Helper()

	resp := doJSONAPIRequest(t, "DELETE", baseURL+"/api/v1/deployments/"+deploymentID, nil, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to delete deployment: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	t.Logf("Deleted deployment: %s", deploymentID)
}

// =============================================================================
// HTTP Helpers
// =============================================================================

// HTTPGet performs an HTTP GET request and returns the response.
func HTTPGet(t *testing.T, url string) *http.Response {
	t.Helper()

	resp, err := testClient.Get(url)
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	return resp
}
