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

// meterRequest is the request payload for the metering API.
type meterRequest struct {
	Events []meterEventPayload `json:"events"`
}

// meterEventPayload is the APIGate format for meter events.
type meterEventPayload struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	EventType    string            `json:"event_type"`
	ResourceID   string            `json:"resource_id"`
	ResourceType string            `json:"resource_type"`
	Quantity     int64             `json:"quantity"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// MeterUsageBatch reports multiple usage events to APIGate.
func (c *APIGateClient) MeterUsageBatch(ctx context.Context, events []domain.MeterEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Convert to APIGate format
	payloadEvents := make([]meterEventPayload, len(events))
	for i, e := range events {
		payloadEvents[i] = meterEventPayload{
			ID:           e.ID,
			UserID:       e.UserID,
			EventType:    string(e.EventType),
			ResourceID:   e.ResourceID,
			ResourceType: e.ResourceType,
			Quantity:     e.Quantity,
			Metadata:     e.Metadata,
			Timestamp:    e.Timestamp,
		}
	}

	payload := meterRequest{Events: payloadEvents}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal meter request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/meter", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
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

	c.logger.Info("usage events reported successfully",
		"count", len(events),
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
