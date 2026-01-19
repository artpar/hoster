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

func TestNewClient(t *testing.T) {
	logger := slog.Default()
	client := NewClient(Config{
		BaseURL: "http://localhost:8082",
		APIKey:  "test-key",
	}, logger)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8082", client.baseURL)
	assert.Equal(t, "test-key", client.apiKey)
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	client := NewClient(Config{
		BaseURL: "http://localhost:8082",
	}, nil)

	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.logger)
}

func TestClient_CreateUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/upstreams", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))

		var upstream Upstream
		err := json.NewDecoder(r.Body).Decode(&upstream)
		require.NoError(t, err)
		assert.Equal(t, "test-upstream", upstream.Name)
		assert.Equal(t, "http://localhost:9091", upstream.BaseURL)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(UpstreamResponse{
			Data: struct {
				ID         string   `json:"id"`
				Type       string   `json:"type"`
				Attributes Upstream `json:"attributes"`
			}{
				ID:   "upstream-123",
				Type: "upstreams",
				Attributes: Upstream{
					Name:    upstream.Name,
					BaseURL: upstream.BaseURL,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
	}, slog.Default())

	created, err := client.CreateUpstream(context.Background(), Upstream{
		Name:    "test-upstream",
		BaseURL: "http://localhost:9091",
	})

	require.NoError(t, err)
	assert.Equal(t, "upstream-123", created.ID)
	assert.Equal(t, "test-upstream", created.Name)
}

func TestClient_GetUpstreamByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/upstreams", r.URL.Path)

		json.NewEncoder(w).Encode(UpstreamsResponse{
			Data: []struct {
				ID         string   `json:"id"`
				Type       string   `json:"type"`
				Attributes Upstream `json:"attributes"`
			}{
				{
					ID:   "upstream-1",
					Type: "upstreams",
					Attributes: Upstream{
						Name:    "other-upstream",
						BaseURL: "http://localhost:8080",
					},
				},
				{
					ID:   "upstream-2",
					Type: "upstreams",
					Attributes: Upstream{
						Name:    "target-upstream",
						BaseURL: "http://localhost:9091",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	found, err := client.GetUpstreamByName(context.Background(), "target-upstream")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "upstream-2", found.ID)
	assert.Equal(t, "target-upstream", found.Name)
}

func TestClient_GetUpstreamByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(UpstreamsResponse{Data: nil})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	found, err := client.GetUpstreamByName(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestClient_UpdateUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/upstreams/upstream-123", r.URL.Path)

		json.NewEncoder(w).Encode(UpstreamResponse{
			Data: struct {
				ID         string   `json:"id"`
				Type       string   `json:"type"`
				Attributes Upstream `json:"attributes"`
			}{
				ID:   "upstream-123",
				Type: "upstreams",
				Attributes: Upstream{
					Name:    "updated-upstream",
					BaseURL: "http://localhost:9092",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	updated, err := client.UpdateUpstream(context.Background(), "upstream-123", Upstream{
		Name:    "updated-upstream",
		BaseURL: "http://localhost:9092",
	})

	require.NoError(t, err)
	assert.Equal(t, "upstream-123", updated.ID)
	assert.Equal(t, "updated-upstream", updated.Name)
}

func TestClient_CreateRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/routes", r.URL.Path)

		var route Route
		err := json.NewDecoder(r.Body).Decode(&route)
		require.NoError(t, err)
		assert.Equal(t, "test-route", route.Name)
		assert.Equal(t, "*.apps.localhost", route.HostPattern)
		assert.Equal(t, "wildcard", route.HostMatchType)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RouteResponse{
			Data: struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Attributes Route  `json:"attributes"`
			}{
				ID:   "route-123",
				Type: "routes",
				Attributes: Route{
					Name:          route.Name,
					HostPattern:   route.HostPattern,
					HostMatchType: route.HostMatchType,
					PathPattern:   route.PathPattern,
					MatchType:     route.MatchType,
					UpstreamID:    route.UpstreamID,
					Enabled:       route.Enabled,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	created, err := client.CreateRoute(context.Background(), Route{
		Name:          "test-route",
		HostPattern:   "*.apps.localhost",
		HostMatchType: "wildcard",
		PathPattern:   "/*",
		MatchType:     "prefix",
		UpstreamID:    "upstream-123",
		Enabled:       true,
	})

	require.NoError(t, err)
	assert.Equal(t, "route-123", created.ID)
	assert.Equal(t, "test-route", created.Name)
}

func TestClient_GetRouteByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/routes", r.URL.Path)

		json.NewEncoder(w).Encode(RoutesResponse{
			Data: []struct {
				ID         string `json:"id"`
				Type       string `json:"type"`
				Attributes Route  `json:"attributes"`
			}{
				{
					ID:   "route-1",
					Type: "routes",
					Attributes: Route{
						Name:        "target-route",
						PathPattern: "/*",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	found, err := client.GetRouteByName(context.Background(), "target-route")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "route-1", found.ID)
	assert.Equal(t, "target-route", found.Name)
}

func TestClient_EnsureUpstream_Create(t *testing.T) {
	createCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return empty list - upstream doesn't exist
			json.NewEncoder(w).Encode(UpstreamsResponse{Data: nil})
			return
		}
		if r.Method == http.MethodPost {
			createCalled = true
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(UpstreamResponse{
				Data: struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					ID:         "new-upstream",
					Type:       "upstreams",
					Attributes: Upstream{Name: "test"},
				},
			})
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	id, err := client.EnsureUpstream(context.Background(), Upstream{
		Name:    "test",
		BaseURL: "http://localhost:9091",
	})

	require.NoError(t, err)
	assert.True(t, createCalled)
	assert.Equal(t, "new-upstream", id)
}

func TestClient_EnsureUpstream_Update(t *testing.T) {
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return existing upstream
			json.NewEncoder(w).Encode(UpstreamsResponse{
				Data: []struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					{
						ID:         "existing-upstream",
						Type:       "upstreams",
						Attributes: Upstream{Name: "test"},
					},
				},
			})
			return
		}
		if r.Method == http.MethodPatch {
			updateCalled = true
			json.NewEncoder(w).Encode(UpstreamResponse{
				Data: struct {
					ID         string   `json:"id"`
					Type       string   `json:"type"`
					Attributes Upstream `json:"attributes"`
				}{
					ID:         "existing-upstream",
					Type:       "upstreams",
					Attributes: Upstream{Name: "test"},
				},
			})
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	id, err := client.EnsureUpstream(context.Background(), Upstream{
		Name:    "test",
		BaseURL: "http://localhost:9092",
	})

	require.NoError(t, err)
	assert.True(t, updateCalled)
	assert.Equal(t, "existing-upstream", id)
}

func TestClient_EnsureRoute_Create(t *testing.T) {
	createCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(RoutesResponse{Data: nil})
			return
		}
		if r.Method == http.MethodPost {
			createCalled = true
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(RouteResponse{
				Data: struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					ID:         "new-route",
					Type:       "routes",
					Attributes: Route{Name: "test-route"},
				},
			})
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	err := client.EnsureRoute(context.Background(), Route{
		Name:        "test-route",
		PathPattern: "/*",
		UpstreamID:  "upstream-123",
	})

	require.NoError(t, err)
	assert.True(t, createCalled)
}

func TestClient_EnsureRoute_Update(t *testing.T) {
	updateCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(RoutesResponse{
				Data: []struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					{
						ID:         "existing-route",
						Type:       "routes",
						Attributes: Route{Name: "test-route"},
					},
				},
			})
			return
		}
		if r.Method == http.MethodPatch {
			updateCalled = true
			json.NewEncoder(w).Encode(RouteResponse{
				Data: struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Attributes Route  `json:"attributes"`
				}{
					ID:         "existing-route",
					Type:       "routes",
					Attributes: Route{Name: "test-route"},
				},
			})
		}
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	err := client.EnsureRoute(context.Background(), Route{
		Name:        "test-route",
		PathPattern: "/*",
		UpstreamID:  "upstream-456",
	})

	require.NoError(t, err)
	assert.True(t, updateCalled)
}

func TestClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL}, slog.Default())

	_, err := client.CreateUpstream(context.Background(), Upstream{Name: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
