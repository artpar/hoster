// Package apigate provides a client for interacting with APIGate's admin API.
// This is used for automated route registration during Hoster startup.
package apigate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client provides methods for interacting with APIGate's admin API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

// Config holds APIGate client configuration.
type Config struct {
	BaseURL string // APIGate base URL, e.g., "http://localhost:8082"
	APIKey  string // Admin API key for authentication
	Timeout time.Duration
}

// NewClient creates a new APIGate client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// =============================================================================
// Upstream Types
// =============================================================================

// Upstream represents an APIGate upstream (backend service).
type Upstream struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name"`
	BaseURL         string `json:"base_url"`
	HealthCheckPath string `json:"health_check_path,omitempty"`
}

// UpstreamResponse wraps an upstream in JSON:API format.
type UpstreamResponse struct {
	Data struct {
		ID         string   `json:"id"`
		Type       string   `json:"type"`
		Attributes Upstream `json:"attributes"`
	} `json:"data"`
}

// UpstreamsResponse wraps multiple upstreams in JSON:API format.
type UpstreamsResponse struct {
	Data []struct {
		ID         string   `json:"id"`
		Type       string   `json:"type"`
		Attributes Upstream `json:"attributes"`
	} `json:"data"`
}

// =============================================================================
// Route Types
// =============================================================================

// RequestTransform defines how to transform requests before forwarding.
// Header values are Expr expressions evaluated with auth context (userID, planID, keyID).
type RequestTransform struct {
	SetHeaders    map[string]string `json:"set_headers,omitempty"`    // Headers to set (key=header name, value=Expr expression)
	DeleteHeaders []string          `json:"delete_headers,omitempty"` // Headers to delete
}

// Route represents an APIGate route.
type Route struct {
	ID               string            `json:"id,omitempty"`
	Name             string            `json:"name"`
	HostPattern      string            `json:"host_pattern,omitempty"`
	HostMatchType    string            `json:"host_match_type,omitempty"` // exact, wildcard, regex
	PathPattern      string            `json:"path_pattern"`
	MatchType        string            `json:"match_type"` // exact, prefix, regex
	UpstreamID       string            `json:"upstream_id"`
	Priority         int               `json:"priority,omitempty"`
	Enabled          bool              `json:"enabled"`
	RequestTransform *RequestTransform `json:"request_transform,omitempty"` // Optional request transformation
}

// RouteResponse wraps a route in JSON:API format.
type RouteResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes Route  `json:"attributes"`
	} `json:"data"`
}

// RoutesResponse wraps multiple routes in JSON:API format.
type RoutesResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes Route  `json:"attributes"`
	} `json:"data"`
}

// =============================================================================
// Upstream Operations
// =============================================================================

// CreateUpstream creates a new upstream in APIGate.
func (c *Client) CreateUpstream(ctx context.Context, upstream Upstream) (*Upstream, error) {
	body, err := json.Marshal(upstream)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/admin/upstreams", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result UpstreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result.Data.Attributes.ID = result.Data.ID
	return &result.Data.Attributes, nil
}

// GetUpstreamByName finds an upstream by name.
func (c *Client) GetUpstreamByName(ctx context.Context, name string) (*Upstream, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/admin/upstreams", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result UpstreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	for _, u := range result.Data {
		if u.Attributes.Name == name {
			u.Attributes.ID = u.ID
			return &u.Attributes, nil
		}
	}

	return nil, nil // Not found
}

// UpdateUpstream updates an existing upstream.
func (c *Client) UpdateUpstream(ctx context.Context, id string, upstream Upstream) (*Upstream, error) {
	body, err := json.Marshal(upstream)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/admin/upstreams/"+id, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result UpstreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result.Data.Attributes.ID = result.Data.ID
	return &result.Data.Attributes, nil
}

// =============================================================================
// Route Operations
// =============================================================================

// CreateRoute creates a new route in APIGate.
func (c *Client) CreateRoute(ctx context.Context, route Route) (*Route, error) {
	body, err := json.Marshal(route)
	if err != nil {
		return nil, fmt.Errorf("marshal route: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/admin/routes", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result RouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result.Data.Attributes.ID = result.Data.ID
	return &result.Data.Attributes, nil
}

// GetRouteByName finds a route by name.
func (c *Client) GetRouteByName(ctx context.Context, name string) (*Route, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/admin/routes", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result RoutesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	for _, r := range result.Data {
		if r.Attributes.Name == name {
			r.Attributes.ID = r.ID
			return &r.Attributes, nil
		}
	}

	return nil, nil // Not found
}

// UpdateRoute updates an existing route.
func (c *Client) UpdateRoute(ctx context.Context, id string, route Route) (*Route, error) {
	body, err := json.Marshal(route)
	if err != nil {
		return nil, fmt.Errorf("marshal route: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/admin/routes/"+id, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result RouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result.Data.Attributes.ID = result.Data.ID
	return &result.Data.Attributes, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

// EnsureUpstream creates or updates an upstream, returning the ID.
func (c *Client) EnsureUpstream(ctx context.Context, upstream Upstream) (string, error) {
	existing, err := c.GetUpstreamByName(ctx, upstream.Name)
	if err != nil {
		return "", fmt.Errorf("check existing upstream: %w", err)
	}

	if existing != nil {
		c.logger.Info("updating existing upstream",
			"name", upstream.Name,
			"id", existing.ID,
		)
		updated, err := c.UpdateUpstream(ctx, existing.ID, upstream)
		if err != nil {
			return "", fmt.Errorf("update upstream: %w", err)
		}
		return updated.ID, nil
	}

	c.logger.Info("creating new upstream", "name", upstream.Name)
	created, err := c.CreateUpstream(ctx, upstream)
	if err != nil {
		return "", fmt.Errorf("create upstream: %w", err)
	}
	return created.ID, nil
}

// EnsureRoute creates or updates a route.
func (c *Client) EnsureRoute(ctx context.Context, route Route) error {
	existing, err := c.GetRouteByName(ctx, route.Name)
	if err != nil {
		return fmt.Errorf("check existing route: %w", err)
	}

	if existing != nil {
		c.logger.Info("updating existing route",
			"name", route.Name,
			"id", existing.ID,
		)
		_, err := c.UpdateRoute(ctx, existing.ID, route)
		if err != nil {
			return fmt.Errorf("update route: %w", err)
		}
		return nil
	}

	c.logger.Info("creating new route", "name", route.Name)
	_, err = c.CreateRoute(ctx, route)
	if err != nil {
		return fmt.Errorf("create route: %w", err)
	}
	return nil
}
