package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// CommandBus dispatches commands emitted by state machine transitions.
type CommandBus interface {
	Dispatch(ctx context.Context, command string, row map[string]any) error
}

// noopBus is a CommandBus that does nothing.
type noopBus struct{}

func (noopBus) Dispatch(_ context.Context, _ string, _ map[string]any) error { return nil }

// APIConfig configures the generic REST API.
type APIConfig struct {
	Store  *Store
	Bus    CommandBus
	Logger *slog.Logger

	// ActionHandlers maps "resource:action" to custom HTTP handlers.
	// e.g., "deployments:start" → startDeploymentHandler
	ActionHandlers map[string]http.HandlerFunc
}

// RegisterRoutes registers generic CRUD routes for all resources in the schema.
// Routes follow JSON:API convention: /api/v1/{resource} and /api/v1/{resource}/{id}
func RegisterRoutes(router *mux.Router, cfg APIConfig) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Bus == nil {
		cfg.Bus = noopBus{}
	}

	for name, res := range cfg.Store.schema {
		prefix := "/api/v1/" + name
		r := res // capture for closures

		// GET /api/v1/{resource}
		router.HandleFunc(prefix, listHandler(cfg, r)).Methods("GET")

		// POST /api/v1/{resource}
		router.HandleFunc(prefix, createHandler(cfg, r)).Methods("POST")

		// GET /api/v1/{resource}/{id}
		router.HandleFunc(prefix+"/{id}", getHandler(cfg, r)).Methods("GET")

		// PATCH /api/v1/{resource}/{id}
		router.HandleFunc(prefix+"/{id}", updateHandler(cfg, r)).Methods("PATCH")

		// DELETE /api/v1/{resource}/{id}
		router.HandleFunc(prefix+"/{id}", deleteHandler(cfg, r)).Methods("DELETE")

		// State machine transition endpoints
		if r.StateMachine != nil {
			// POST /api/v1/{resource}/{id}/transition/{state}
			router.HandleFunc(prefix+"/{id}/transition/{state}", transitionHandler(cfg, r)).Methods("POST")
		}

		// Custom action handlers
		if cfg.ActionHandlers != nil {
			for _, action := range r.Actions {
				key := name + ":" + action.Name
				if handler, ok := cfg.ActionHandlers[key]; ok {
					route := prefix + "/{id}/" + action.Name
					router.HandleFunc(route, handler).Methods(action.Method)
				}
			}
		}

		cfg.Logger.Debug("registered routes", "resource", name, "prefix", prefix)
	}
}

// =============================================================================
// Generic Handlers
// =============================================================================

func listHandler(cfg APIConfig, res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)

		page := parsePage(r)

		// Build filters
		var filters []Filter

		// Owner scoping: if resource has an owner field and user is authenticated,
		// filter by owner
		if res.Owner != "" && authCtx.Authenticated && !res.PublicRead {
			filters = append(filters, Filter{Field: res.Owner, Value: authCtx.UserID})
		}

		// Parse filter query params: filter[field]=value
		for key, values := range r.URL.Query() {
			if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
				fieldName := key[7 : len(key)-1]
				if len(values) > 0 {
					filters = append(filters, Filter{Field: fieldName, Value: values[0]})
				}
			}
		}

		rows, err := cfg.Store.List(ctx, res.Name, filters, page)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Apply visibility filter
		if res.Visibility != nil {
			var visible []map[string]any
			for _, row := range rows {
				if res.Visibility(ctx, authCtx, row) {
					visible = append(visible, row)
				}
			}
			rows = visible
		}

		// Strip write-only and internal fields from responses
		for _, row := range rows {
			stripFields(res, row)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowsToJSONAPI(res.Name, rows),
			"meta": map[string]any{
				"total":  len(rows),
				"limit":  page.Limit,
				"offset": page.Offset,
			},
		})
	}
}

func getHandler(cfg APIConfig, res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		row, err := cfg.Store.Get(ctx, res.Name, id)
		if err != nil {
			if isNotFoundErr(err) {
				writeError(w, http.StatusNotFound, res.Name+" not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Check visibility
		if res.Visibility != nil && !res.Visibility(ctx, authCtx, row) {
			writeError(w, http.StatusNotFound, res.Name+" not found")
			return
		}

		// Check owner
		if res.Owner != "" && authCtx.Authenticated {
			if ownerID, ok := toInt64(row[res.Owner]); ok {
				if int(ownerID) != authCtx.UserID && !res.PublicRead {
					writeError(w, http.StatusNotFound, res.Name+" not found")
					return
				}
			}
		}

		stripFields(res, row)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI(res.Name, row),
		})
	}
}

func createHandler(cfg APIConfig, res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)

		// Require authentication for create
		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Parse request body (JSON:API format)
		data, err := parseJSONAPIBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		// Set owner field from auth context
		if res.Owner != "" {
			data[res.Owner] = authCtx.UserID
		}

		// Remove internal fields that shouldn't be set by the client
		// (except owner, which we just set)
		for _, f := range res.Fields {
			if f.Internal && f.Name != res.Owner {
				delete(data, f.Name)
			}
		}

		// BeforeCreate hook
		if res.BeforeCreate != nil {
			if err := res.BeforeCreate(ctx, authCtx, data); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		row, err := cfg.Store.Create(ctx, res.Name, data)
		if err != nil {
			if strings.Contains(err.Error(), "validation error") {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		stripFields(res, row)
		writeJSON(w, http.StatusCreated, map[string]any{
			"data": rowToJSONAPI(res.Name, row),
		})
	}
}

func updateHandler(cfg APIConfig, res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Check ownership
		existing, err := cfg.Store.Get(ctx, res.Name, id)
		if err != nil {
			if isNotFoundErr(err) {
				writeError(w, http.StatusNotFound, res.Name+" not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if res.Owner != "" {
			if ownerID, ok := toInt64(existing[res.Owner]); ok {
				if int(ownerID) != authCtx.UserID {
					writeError(w, http.StatusForbidden, "not authorized to modify this "+res.Name)
					return
				}
			}
		}

		// Parse update data
		data, err := parseJSONAPIBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		// Remove internal fields from update
		for _, f := range res.Fields {
			if f.Internal {
				delete(data, f.Name)
			}
		}

		row, err := cfg.Store.Update(ctx, res.Name, id, data)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		stripFields(res, row)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI(res.Name, row),
		})
	}
}

func deleteHandler(cfg APIConfig, res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Check ownership
		existing, err := cfg.Store.Get(ctx, res.Name, id)
		if err != nil {
			if isNotFoundErr(err) {
				writeError(w, http.StatusNotFound, res.Name+" not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if res.Owner != "" {
			if ownerID, ok := toInt64(existing[res.Owner]); ok {
				if int(ownerID) != authCtx.UserID {
					writeError(w, http.StatusForbidden, "not authorized to delete this "+res.Name)
					return
				}
			}
		}

		// BeforeDelete hook
		if res.BeforeDelete != nil {
			if err := res.BeforeDelete(ctx, authCtx, existing); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
		}

		if err := cfg.Store.Delete(ctx, res.Name, id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func transitionHandler(cfg APIConfig, res *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]
		state := mux.Vars(r)["state"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Check ownership
		existing, err := cfg.Store.Get(ctx, res.Name, id)
		if err != nil {
			if isNotFoundErr(err) {
				writeError(w, http.StatusNotFound, res.Name+" not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if res.Owner != "" {
			if ownerID, ok := toInt64(existing[res.Owner]); ok {
				if int(ownerID) != authCtx.UserID {
					writeError(w, http.StatusForbidden, "not authorized")
					return
				}
			}
		}

		row, cmd, err := cfg.Store.Transition(ctx, res.Name, id, state)
		if err != nil {
			if strings.Contains(err.Error(), "invalid state transition") {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			if strings.Contains(err.Error(), "guard failed") {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Dispatch command if state machine triggers one
		if cmd != "" && cfg.Bus != nil {
			if err := cfg.Bus.Dispatch(ctx, cmd, row); err != nil {
				cfg.Logger.Error("command dispatch failed", "command", cmd, "error", err)
				// Don't fail the transition — the state was already saved
			}
		}

		stripFields(res, row)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI(res.Name, row),
		})
	}
}

// =============================================================================
// JSON:API Response Helpers
// =============================================================================

// rowToJSONAPI converts a map row to a JSON:API resource object.
func rowToJSONAPI(resourceType string, row map[string]any) map[string]any {
	refID, _ := row["reference_id"].(string)

	attrs := make(map[string]any)
	for k, v := range row {
		if k == "id" || k == "reference_id" {
			continue
		}
		attrs[k] = v
	}

	return map[string]any{
		"type":       resourceType,
		"id":         refID,
		"attributes": attrs,
	}
}

// rowsToJSONAPI converts multiple rows to JSON:API format.
func rowsToJSONAPI(resourceType string, rows []map[string]any) []map[string]any {
	if rows == nil {
		return []map[string]any{}
	}
	result := make([]map[string]any, len(rows))
	for i, row := range rows {
		result[i] = rowToJSONAPI(resourceType, row)
	}
	return result
}

// stripFields removes write-only fields from a row before sending in a response.
func stripFields(res *Resource, row map[string]any) {
	for _, f := range res.Fields {
		if f.WriteOnly {
			delete(row, f.Name)
		}
	}
	// Don't expose internal integer PK in API responses
	delete(row, "id")
}

// parseJSONAPIBody parses a JSON:API request body and returns the attributes map.
func parseJSONAPIBody(r *http.Request) (map[string]any, error) {
	var body struct {
		Data struct {
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}

	// Try JSON:API format first
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		return nil, err
	}

	if body.Data.Attributes != nil {
		return body.Data.Attributes, nil
	}

	// If no attributes, try flat JSON (backward compat)
	return body.Data.Attributes, fmt.Errorf("missing data.attributes in request body")
}

// getAuthContext extracts AuthContext from an HTTP request.
// Uses the auth bridge (auth_bridge.go) which reads from the existing auth middleware.
func getAuthContext(r *http.Request) AuthContext {
	return AuthFromRequest(r)
}

// =============================================================================
// HTTP Response Helpers
// =============================================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]any{
			{
				"status": strconv.Itoa(status),
				"title":  http.StatusText(status),
				"detail": detail,
			},
		},
	})
}

// parsePage extracts pagination from query parameters.
func parsePage(r *http.Request) Page {
	p := DefaultPage()
	if v := r.URL.Query().Get("page[size]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Limit = n
		}
	}
	if v := r.URL.Query().Get("page[offset]"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.Offset = n
		}
	}
	if v := r.URL.Query().Get("page[number]"); v != "" {
		if pn, err := strconv.Atoi(v); err == nil && pn > 0 {
			p.Offset = (pn - 1) * p.Limit
		}
	}
	return p.Normalize()
}

func isNotFoundErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}
