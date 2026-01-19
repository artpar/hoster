package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIGateClient_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	client := NewAPIGateClient(cfg, nil)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
	assert.NotNil(t, client.httpClient)
}

func TestNewAPIGateClient_CustomConfig(t *testing.T) {
	cfg := Config{
		BaseURL:    "https://api.example.com",
		ServiceKey: "test-key",
		Timeout:    60 * time.Second,
	}
	client := NewAPIGateClient(cfg, nil)

	assert.Equal(t, "https://api.example.com", client.baseURL)
	assert.Equal(t, "test-key", client.serviceKey)
}

func TestAPIGateClient_MeterUsage_Success(t *testing.T) {
	// Create test server
	var receivedRequest jsonAPIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/meter", r.URL.Path)
		assert.Equal(t, "application/vnd.api+json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-service-key", r.Header.Get("X-API-Key"))

		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"meta": {"accepted": 1, "rejected": 0}}`))
	}))
	defer server.Close()

	client := NewAPIGateClient(Config{
		BaseURL:    server.URL,
		ServiceKey: "test-service-key",
	}, nil)

	event := domain.NewMeterEvent(
		"evt-123",
		"user-456",
		domain.EventDeploymentCreated,
		"depl-789",
		"deployment",
	).WithMetadata("template_id", "tmpl-001")

	err := client.MeterUsage(context.Background(), event)
	require.NoError(t, err)

	// Verify the JSON:API request format
	require.Len(t, receivedRequest.Data, 1)
	assert.Equal(t, "usage_events", receivedRequest.Data[0].Type)
	assert.Equal(t, "evt-123", receivedRequest.Data[0].Attributes.ID)
	assert.Equal(t, "user-456", receivedRequest.Data[0].Attributes.UserID)
	assert.Equal(t, "deployment.created", receivedRequest.Data[0].Attributes.EventType)
	assert.Equal(t, "depl-789", receivedRequest.Data[0].Attributes.ResourceID)
	assert.Equal(t, "tmpl-001", receivedRequest.Data[0].Attributes.Metadata["template_id"])
}

func TestAPIGateClient_MeterUsageBatch_Success(t *testing.T) {
	var receivedRequest jsonAPIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedRequest)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"meta": {"accepted": 2, "rejected": 0}}`))
	}))
	defer server.Close()

	client := NewAPIGateClient(Config{BaseURL: server.URL}, nil)

	events := []domain.MeterEvent{
		domain.NewMeterEvent("evt-1", "user-1", domain.EventDeploymentCreated, "depl-1", "deployment"),
		domain.NewMeterEvent("evt-2", "user-1", domain.EventDeploymentStarted, "depl-1", "deployment"),
	}

	err := client.MeterUsageBatch(context.Background(), events)
	require.NoError(t, err)

	assert.Len(t, receivedRequest.Data, 2)
	assert.Equal(t, "usage_events", receivedRequest.Data[0].Type)
	assert.Equal(t, "deployment.created", receivedRequest.Data[0].Attributes.EventType)
	assert.Equal(t, "deployment.started", receivedRequest.Data[1].Attributes.EventType)
}

func TestAPIGateClient_MeterUsageBatch_EmptyEvents(t *testing.T) {
	// Server should not be called for empty events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called for empty events")
	}))
	defer server.Close()

	client := NewAPIGateClient(Config{BaseURL: server.URL}, nil)

	err := client.MeterUsageBatch(context.Background(), []domain.MeterEvent{})
	require.NoError(t, err)
}

func TestAPIGateClient_MeterUsage_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewAPIGateClient(Config{BaseURL: server.URL}, nil)

	event := domain.NewMeterEvent("evt-1", "user-1", domain.EventDeploymentCreated, "depl-1", "deployment")

	err := client.MeterUsage(context.Background(), event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAPIGateClient_MeterUsage_NetworkError(t *testing.T) {
	// Use an invalid URL to trigger network error
	client := NewAPIGateClient(Config{
		BaseURL: "http://localhost:99999",
		Timeout: 1 * time.Second,
	}, nil)

	event := domain.NewMeterEvent("evt-1", "user-1", domain.EventDeploymentCreated, "depl-1", "deployment")

	err := client.MeterUsage(context.Background(), event)
	assert.Error(t, err)
}

func TestNoopClient_MeterUsage(t *testing.T) {
	client := NewNoopClient(nil)

	event := domain.NewMeterEvent("evt-1", "user-1", domain.EventDeploymentCreated, "depl-1", "deployment")

	err := client.MeterUsage(context.Background(), event)
	assert.NoError(t, err)
}

func TestNoopClient_MeterUsageBatch(t *testing.T) {
	client := NewNoopClient(nil)

	events := []domain.MeterEvent{
		domain.NewMeterEvent("evt-1", "user-1", domain.EventDeploymentCreated, "depl-1", "deployment"),
		domain.NewMeterEvent("evt-2", "user-1", domain.EventDeploymentStarted, "depl-1", "deployment"),
	}

	err := client.MeterUsageBatch(context.Background(), events)
	assert.NoError(t, err)
}
