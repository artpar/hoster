// Package billing provides integration with APIGate for usage metering and billing.
// Following ADR-005: APIGate Integration and F009: Billing Integration
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
)

// =============================================================================
// Client Interface
// =============================================================================

// Client defines the interface for reporting usage events to APIGate.
type Client interface {
	// MeterUsage reports a single usage event to APIGate.
	MeterUsage(ctx context.Context, event domain.MeterEvent) error

	// MeterUsageBatch reports multiple usage events at once.
	MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error
}

// =============================================================================
// APIGate Client Implementation
// =============================================================================

// APIGateClient implements Client for the APIGate billing API.
type APIGateClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// APIGateConfig holds configuration for the APIGate client.
type APIGateConfig struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// DefaultConfig returns default APIGate client configuration.
func DefaultConfig() APIGateConfig {
	return APIGateConfig{
		BaseURL:       "http://localhost:8080",
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
	}
}

// NewAPIGateClient creates a new APIGate billing client.
func NewAPIGateClient(cfg APIGateConfig) *APIGateClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &APIGateClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// meterRequest represents the request body for metering usage.
type meterRequest struct {
	Events []meterEventPayload `json:"events"`
}

// meterEventPayload represents a single event in the meter request.
type meterEventPayload struct {
	EventID      string            `json:"event_id"`
	UserID       string            `json:"user_id"`
	EventType    string            `json:"event_type"`
	ResourceID   string            `json:"resource_id"`
	ResourceType string            `json:"resource_type"`
	Quantity     int64             `json:"quantity"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    string            `json:"timestamp"`
}

// MeterUsage reports a single usage event to APIGate.
func (c *APIGateClient) MeterUsage(ctx context.Context, event domain.MeterEvent) error {
	return c.MeterUsageBatch(ctx, []domain.MeterEvent{event})
}

// MeterUsageBatch reports multiple usage events to APIGate.
func (c *APIGateClient) MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Convert to API payload
	payload := meterRequest{
		Events: make([]meterEventPayload, len(events)),
	}

	for i, event := range events {
		payload.Events[i] = meterEventPayload{
			EventID:      event.ID,
			UserID:       event.UserID,
			EventType:    string(event.EventType),
			ResourceID:   event.ResourceID,
			ResourceType: event.ResourceType,
			Quantity:     event.Quantity,
			Metadata:     event.Metadata,
			Timestamp:    event.Timestamp.Format(time.RFC3339),
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal meter request: %w", err)
	}

	// Create request
	url := c.baseURL + "/api/v1/meter"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send meter request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("APIGate returned error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// =============================================================================
// No-Op Client (for development/testing)
// =============================================================================

// NoOpClient is a billing client that does nothing (for development mode).
type NoOpClient struct{}

// NewNoOpClient creates a no-op billing client.
func NewNoOpClient() *NoOpClient {
	return &NoOpClient{}
}

// MeterUsage does nothing.
func (c *NoOpClient) MeterUsage(ctx context.Context, event domain.MeterEvent) error {
	return nil
}

// MeterUsageBatch does nothing.
func (c *NoOpClient) MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error {
	return nil
}
