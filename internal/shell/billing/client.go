// Package billing provides integration with APIGate for usage metering and billing.
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
)

// Client interface for billing operations.
type Client interface {
	// MeterUsage reports a usage event to APIGate.
	MeterUsage(ctx context.Context, event domain.MeterEvent) error
	// MeterUsageBatch reports multiple usage events to APIGate.
	MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error
}

// APIGateClient implements the billing client for APIGate.
type APIGateClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
	logger     *slog.Logger
}

// Config holds the configuration for the APIGate billing client.
type Config struct {
	// BaseURL is the base URL of the APIGate API (e.g., "http://localhost:8080").
	BaseURL string
	// ServiceKey is the API key for authenticating with APIGate.
	ServiceKey string
	// Timeout is the HTTP client timeout.
	Timeout time.Duration
}

// DefaultConfig returns a default configuration for the billing client.
func DefaultConfig() Config {
	return Config{
		BaseURL:    "http://localhost:8080",
		ServiceKey: "",
		Timeout:    30 * time.Second,
	}
}

// NewAPIGateClient creates a new APIGate billing client.
func NewAPIGateClient(cfg Config, logger *slog.Logger) *APIGateClient {
	if logger == nil {
		logger = slog.Default()
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &APIGateClient{
		baseURL:    cfg.BaseURL,
		serviceKey: cfg.ServiceKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger.With("component", "billing_client"),
	}
}

// MeterUsage reports a single usage event to APIGate.
func (c *APIGateClient) MeterUsage(ctx context.Context, event domain.MeterEvent) error {
	return c.MeterUsageBatch(ctx, []domain.MeterEvent{event})
}

// jsonAPIRequest is the JSON:API format request payload for the metering API.
type jsonAPIRequest struct {
	Data []jsonAPIResource `json:"data"`
}

// jsonAPIResource represents a single resource in JSON:API format.
type jsonAPIResource struct {
	Type       string                  `json:"type"`
	Attributes meterEventAttributes    `json:"attributes"`
}

// meterEventAttributes contains the event data in JSON:API attributes format.
type meterEventAttributes struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	EventType    string            `json:"event_type"`
	ResourceID   string            `json:"resource_id"`
	ResourceType string            `json:"resource_type"`
	Quantity     int64             `json:"quantity"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    string            `json:"timestamp,omitempty"`
}

// jsonAPIResponse is the JSON:API format response from the metering API.
type jsonAPIResponse struct {
	Meta jsonAPIMeta `json:"meta"`
}

// jsonAPIMeta contains response metadata.
type jsonAPIMeta struct {
	Accepted int              `json:"accepted"`
	Rejected int              `json:"rejected"`
	Errors   []jsonAPIError   `json:"errors,omitempty"`
}

// jsonAPIError represents an error in the response.
type jsonAPIError struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// MeterUsageBatch reports multiple usage events to APIGate using JSON:API format.
func (c *APIGateClient) MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Convert to JSON:API format
	resources := make([]jsonAPIResource, len(events))
	for i, e := range events {
		// Use the user's UUID reference_id for APIGate, fall back to integer
		userID := e.UserRefID
		if userID == "" {
			userID = fmt.Sprintf("%d", e.UserID)
		}
		resources[i] = jsonAPIResource{
			Type: "usage_events",
			Attributes: meterEventAttributes{
				ID:           e.ReferenceID,
				UserID:       userID,
				EventType:    string(e.EventType),
				ResourceID:   e.ResourceID,
				ResourceType: e.ResourceType,
				Quantity:     e.Quantity,
				Metadata:     e.Metadata,
				Timestamp:    e.Timestamp.UTC().Format(time.RFC3339),
			},
		}
	}

	payload := jsonAPIRequest{Data: resources}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal meter request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/meter", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/vnd.api+json")
	if c.serviceKey != "" {
		req.Header.Set("X-API-Key", c.serviceKey)
	}

	c.logger.Debug("reporting usage events",
		"count", len(events),
		"first_event_type", events[0].EventType,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("failed to report usage",
			"error", err,
			"count", len(events),
		)
		return fmt.Errorf("send meter request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		c.logger.Warn("meter request failed",
			"status", resp.StatusCode,
			"body", string(respBody),
		)
		return fmt.Errorf("meter request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse JSON:API response
	var apiResp jsonAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		// Non-fatal: log warning but don't fail if response parsing fails
		c.logger.Debug("could not parse response body", "error", err)
	} else if apiResp.Meta.Rejected > 0 {
		c.logger.Warn("some events were rejected",
			"accepted", apiResp.Meta.Accepted,
			"rejected", apiResp.Meta.Rejected,
			"errors", apiResp.Meta.Errors,
		)
	}

	c.logger.Info("usage events reported successfully",
		"count", len(events),
		"accepted", apiResp.Meta.Accepted,
	)

	return nil
}

// NoopClient is a billing client that does nothing.
// Useful for development/testing when APIGate is not available.
type NoopClient struct {
	logger *slog.Logger
}

// NewNoopClient creates a new no-op billing client.
func NewNoopClient(logger *slog.Logger) *NoopClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &NoopClient{
		logger: logger.With("component", "billing_client_noop"),
	}
}

// MeterUsage logs the event but does not send it anywhere.
func (c *NoopClient) MeterUsage(ctx context.Context, event domain.MeterEvent) error {
	c.logger.Debug("noop: would meter usage",
		"event_type", event.EventType,
		"user_id", event.UserID,
		"resource_id", event.ResourceID,
	)
	return nil
}

// MeterUsageBatch logs the events but does not send them anywhere.
func (c *NoopClient) MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error {
	c.logger.Debug("noop: would meter usage batch",
		"count", len(events),
	)
	return nil
}
