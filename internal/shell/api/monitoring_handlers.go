// Package api provides HTTP handlers for the Hoster API.
package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/auth"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/core/monitoring"
	"github.com/artpar/hoster/internal/shell/docker"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/gorilla/mux"
)

// =============================================================================
// Monitoring Handlers (F010: Monitoring Dashboard)
// =============================================================================

// MonitoringHandlers provides monitoring endpoints for deployments.
type MonitoringHandlers struct {
	store  store.Store
	docker docker.Client
}

// NewMonitoringHandlers creates a new monitoring handlers instance.
func NewMonitoringHandlers(s store.Store, d docker.Client) *MonitoringHandlers {
	return &MonitoringHandlers{
		store:  s,
		docker: d,
	}
}

// RegisterRoutes registers the monitoring routes.
func (h *MonitoringHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/deployments/{id}/monitoring/health", h.HealthHandler).Methods("GET")
	r.HandleFunc("/api/v1/deployments/{id}/monitoring/logs", h.LogsHandler).Methods("GET")
	r.HandleFunc("/api/v1/deployments/{id}/monitoring/stats", h.StatsHandler).Methods("GET")
	r.HandleFunc("/api/v1/deployments/{id}/monitoring/events", h.EventsHandler).Methods("GET")
}

// =============================================================================
// Health Endpoint
// =============================================================================

// healthResponse represents the JSON:API response for deployment health.
type healthResponse struct {
	Data healthData `json:"data"`
}

type healthData struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Attributes healthAttributes `json:"attributes"`
}

type healthAttributes struct {
	Status     string                    `json:"status"`
	Containers []containerHealthResponse `json:"containers"`
	CheckedAt  string                    `json:"checked_at"`
}

type containerHealthResponse struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Health    string  `json:"health"`
	StartedAt *string `json:"started_at,omitempty"`
	Restarts  int     `json:"restarts"`
}

// HealthHandler returns the health status of a deployment.
// GET /api/v1/deployments/{id}/monitoring/health
func (h *MonitoringHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	// Get deployment
	deployment, err := h.store.GetDeployment(ctx, deploymentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Check authorization
	if !auth.CanViewDeployment(authCtx, *deployment) {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Get container health for each container in the deployment
	var containerHealths []domain.ContainerHealth
	for _, c := range deployment.Containers {
		info, err := h.docker.InspectContainer(c.ID)
		if err != nil {
			containerHealths = append(containerHealths, domain.ContainerHealth{
				Name:   c.ServiceName,
				Status: "unknown",
				Health: domain.HealthStatusUnknown,
			})
			continue
		}

		health := monitoring.DetermineContainerHealth(
			info.State,
			ptrOrNil(info.Health),
			0, // TODO: Get restart count
		)

		containerHealths = append(containerHealths, domain.ContainerHealth{
			Name:      c.ServiceName,
			Status:    info.State,
			Health:    health,
			StartedAt: info.StartedAt,
			Restarts:  0,
		})
	}

	// Aggregate health
	overallHealth := monitoring.AggregateHealth(containerHealths)
	now := time.Now()

	// Build response
	response := healthResponse{
		Data: healthData{
			Type: "deployment-health",
			ID:   deploymentID,
			Attributes: healthAttributes{
				Status:    string(overallHealth),
				CheckedAt: now.Format(time.RFC3339),
			},
		},
	}

	for _, ch := range containerHealths {
		var startedAt *string
		if ch.StartedAt != nil {
			s := ch.StartedAt.Format(time.RFC3339)
			startedAt = &s
		}
		response.Data.Attributes.Containers = append(response.Data.Attributes.Containers, containerHealthResponse{
			Name:      ch.Name,
			Status:    ch.Status,
			Health:    string(ch.Health),
			StartedAt: startedAt,
			Restarts:  ch.Restarts,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// =============================================================================
// Logs Endpoint
// =============================================================================

// logsResponse represents the JSON:API response for deployment logs.
type logsResponse struct {
	Data logData `json:"data"`
	Meta logMeta `json:"meta"`
}

type logData struct {
	Type       string        `json:"type"`
	ID         string        `json:"id"`
	Attributes logAttributes `json:"attributes"`
}

type logAttributes struct {
	Logs []logEntry `json:"logs"`
}

type logEntry struct {
	Container string `json:"container"`
	Timestamp string `json:"timestamp"`
	Stream    string `json:"stream"`
	Message   string `json:"message"`
}

type logMeta struct {
	ContainerFilter *string `json:"container_filter"`
	Tail            int     `json:"tail"`
	Since           *string `json:"since"`
}

// LogsHandler returns logs from deployment containers.
// GET /api/v1/deployments/{id}/monitoring/logs
func (h *MonitoringHandlers) LogsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	// Get deployment
	deployment, err := h.store.GetDeployment(ctx, deploymentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Check authorization
	if !auth.CanViewDeployment(authCtx, *deployment) {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Parse query parameters
	tail := 100
	if t := r.URL.Query().Get("tail"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil && parsed > 0 {
			tail = parsed
		}
	}
	containerFilter := r.URL.Query().Get("container")
	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		since, _ = time.Parse(time.RFC3339, sinceStr)
	}

	// Collect logs from containers
	var allLogs []logEntry
	for _, c := range deployment.Containers {
		// Skip if filtering by container and this isn't the one
		if containerFilter != "" && c.ServiceName != containerFilter {
			continue
		}

		logOpts := docker.LogOptions{
			Tail:       strconv.Itoa(tail),
			Timestamps: true,
			Since:      since,
		}

		reader, err := h.docker.ContainerLogs(c.ID, logOpts)
		if err != nil {
			continue
		}

		logs := parseDockerLogs(reader, c.ServiceName)
		reader.Close()
		allLogs = append(allLogs, logs...)
	}

	// Build response
	var filterPtr *string
	if containerFilter != "" {
		filterPtr = &containerFilter
	}
	var sincePtr *string
	if sinceStr != "" {
		sincePtr = &sinceStr
	}

	response := logsResponse{
		Data: logData{
			Type: "deployment-logs",
			ID:   deploymentID,
			Attributes: logAttributes{
				Logs: allLogs,
			},
		},
		Meta: logMeta{
			ContainerFilter: filterPtr,
			Tail:            tail,
			Since:           sincePtr,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// parseDockerLogs parses Docker log output into log entries.
func parseDockerLogs(reader interface{ Read([]byte) (int, error) }, containerName string) []logEntry {
	var entries []logEntry
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) < 8 {
			continue
		}

		// Docker multiplexes stdout/stderr with an 8-byte header
		// First byte: 1=stdout, 2=stderr
		stream := "stdout"
		if line[0] == 2 {
			stream = "stderr"
		}

		// Skip header
		message := string(line[8:])
		if len(line) <= 8 {
			message = string(line)
		}

		// Parse timestamp if present (format: 2006-01-02T15:04:05.000000000Z)
		timestamp := time.Now()
		if len(message) > 30 && message[4] == '-' && message[7] == '-' {
			if t, err := time.Parse(time.RFC3339Nano, message[:30]); err == nil {
				timestamp = t
				message = strings.TrimSpace(message[31:])
			}
		}

		entries = append(entries, logEntry{
			Container: containerName,
			Timestamp: timestamp.Format(time.RFC3339),
			Stream:    stream,
			Message:   message,
		})
	}

	return entries
}

// =============================================================================
// Stats Endpoint
// =============================================================================

// statsResponse represents the JSON:API response for deployment stats.
type statsResponse struct {
	Data statsData `json:"data"`
}

type statsData struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Attributes statsAttributes `json:"attributes"`
}

type statsAttributes struct {
	Containers  []containerStatsResponse `json:"containers"`
	CollectedAt string                   `json:"collected_at"`
}

type containerStatsResponse struct {
	Name             string  `json:"name"`
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	MemoryLimitBytes int64   `json:"memory_limit_bytes"`
	MemoryPercent    float64 `json:"memory_percent"`
	NetworkRxBytes   int64   `json:"network_rx_bytes"`
	NetworkTxBytes   int64   `json:"network_tx_bytes"`
	BlockReadBytes   int64   `json:"block_read_bytes"`
	BlockWriteBytes  int64   `json:"block_write_bytes"`
	PIDs             int     `json:"pids"`
}

// StatsHandler returns resource statistics for deployment containers.
// GET /api/v1/deployments/{id}/monitoring/stats
func (h *MonitoringHandlers) StatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	// Get deployment
	deployment, err := h.store.GetDeployment(ctx, deploymentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Check authorization
	if !auth.CanViewDeployment(authCtx, *deployment) {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Collect stats from containers
	var containerStats []containerStatsResponse
	for _, c := range deployment.Containers {
		stats, err := h.docker.ContainerStats(c.ID)
		if err != nil {
			continue
		}

		containerStats = append(containerStats, containerStatsResponse{
			Name:             c.ServiceName,
			CPUPercent:       stats.CPUPercent,
			MemoryUsageBytes: stats.MemoryUsageBytes,
			MemoryLimitBytes: stats.MemoryLimitBytes,
			MemoryPercent:    stats.MemoryPercent,
			NetworkRxBytes:   stats.NetworkRxBytes,
			NetworkTxBytes:   stats.NetworkTxBytes,
			BlockReadBytes:   stats.BlockReadBytes,
			BlockWriteBytes:  stats.BlockWriteBytes,
			PIDs:             stats.PIDs,
		})
	}

	// Build response
	response := statsResponse{
		Data: statsData{
			Type: "deployment-stats",
			ID:   deploymentID,
			Attributes: statsAttributes{
				Containers:  containerStats,
				CollectedAt: time.Now().Format(time.RFC3339),
			},
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// =============================================================================
// Events Endpoint
// =============================================================================

// eventsResponse represents the JSON:API response for deployment events.
type eventsResponse struct {
	Data eventData `json:"data"`
	Meta eventMeta `json:"meta"`
}

type eventData struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Attributes eventsAttributes `json:"attributes"`
}

type eventsAttributes struct {
	Events []eventEntry `json:"events"`
}

type eventEntry struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Container string `json:"container"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type eventMeta struct {
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// EventsHandler returns lifecycle events for a deployment.
// GET /api/v1/deployments/{id}/monitoring/events
func (h *MonitoringHandlers) EventsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	// Get deployment
	deployment, err := h.store.GetDeployment(ctx, deploymentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Check authorization
	if !auth.CanViewDeployment(authCtx, *deployment) {
		writeError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	// Parse query parameters
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	eventType := r.URL.Query().Get("type")
	var eventTypePtr *string
	if eventType != "" {
		eventTypePtr = &eventType
	}

	// Get events from store
	events, err := h.store.GetContainerEvents(ctx, deploymentID, limit, eventTypePtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get events")
		return
	}

	// Build response
	var eventEntries []eventEntry
	for _, e := range events {
		eventEntries = append(eventEntries, eventEntry{
			ID:        e.ReferenceID,
			Type:      string(e.Type),
			Container: e.Container,
			Message:   e.Message,
			Timestamp: e.Timestamp.Format(time.RFC3339),
		})
	}

	response := eventsResponse{
		Data: eventData{
			Type: "deployment-events",
			ID:   deploymentID,
			Attributes: eventsAttributes{
				Events: eventEntries,
			},
		},
		Meta: eventMeta{
			Limit: limit,
			Total: len(eventEntries),
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// =============================================================================
// Helpers
// =============================================================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"status": fmt.Sprintf("%d", status),
				"title":  http.StatusText(status),
				"detail": message,
			},
		},
	})
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
