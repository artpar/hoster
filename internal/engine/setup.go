package engine

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
	coreprovider "github.com/artpar/hoster/internal/core/provider"
	"github.com/artpar/hoster/internal/shell/billing"
	"github.com/gorilla/mux"
)

//go:embed all:webui/dist
var webUI embed.FS

// SetupConfig holds configuration for the engine HTTP handler.
type SetupConfig struct {
	Store         *Store
	Bus           *Bus
	Logger        *slog.Logger
	BaseDomain    string
	ConfigDir     string
	SharedSecret  string
	EncryptionKey []byte
	Version       string
	StripeKey     string
}

// Setup creates the complete HTTP handler using the engine.
func Setup(cfg SetupConfig) http.Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Wire encryption key to store for encrypted fields
	if len(cfg.EncryptionKey) > 0 {
		cfg.Store.SetEncryptionKey(cfg.EncryptionKey)
	}

	router := mux.NewRouter()

	// Middleware
	router.Use(requestIDMiddleware)
	router.Use(recoveryMiddleware(cfg.Logger))
	router.Use(AuthMiddleware(cfg.Store, cfg.SharedSecret, cfg.Logger))

	// Health endpoints
	router.HandleFunc("/health", healthHandler(cfg.Version)).Methods("GET")
	router.HandleFunc("/ready", readyHandler).Methods("GET")

	// Wire SSH key BeforeCreate: compute fingerprint + public_key from private key
	if sshRes := cfg.Store.Resource("ssh_keys"); sshRes != nil {
		sshRes.BeforeCreate = func(ctx context.Context, authCtx AuthContext, data map[string]any) error {
			if pk, ok := data["private_key"].(string); ok && pk != "" {
				fp, err := crypto.GetSSHPublicKeyFingerprint([]byte(pk))
				if err != nil {
					return fmt.Errorf("invalid SSH private key: %w", err)
				}
				data["fingerprint"] = fp

				if _, hasPub := data["public_key"]; !hasPub {
					pubKey, err := crypto.GetSSHPublicKey([]byte(pk))
					if err == nil {
						data["public_key"] = pubKey
					}
				}
			}
			return nil
		}
	}

	// Wire template BeforeDelete: prevent deleting templates with active deployments
	if tmplRes := cfg.Store.Resource("templates"); tmplRes != nil {
		store := cfg.Store
		tmplRes.BeforeDelete = func(ctx context.Context, authCtx AuthContext, row map[string]any) error {
			tmplID, ok := toInt64(row["id"])
			if !ok {
				return fmt.Errorf("invalid template ID")
			}
			depls, err := store.List(ctx, "deployments", []Filter{
				{Field: "template_id", Value: tmplID},
			}, Page{Limit: 1, Offset: 0})
			if err == nil && len(depls) > 0 {
				return fmt.Errorf("cannot delete template: it has active deployments")
			}
			return nil
		}
	}

	// Wire deployment BeforeCreate: plan limit check + resolve template_version from template
	// Wire deployment AfterCreate: record billing event
	if deplRes := cfg.Store.Resource("deployments"); deplRes != nil {
		store := cfg.Store
		deplRes.BeforeCreate = func(ctx context.Context, authCtx AuthContext, data map[string]any) error {
			// Check plan limits
			if authCtx.PlanLimits.MaxDeployments > 0 {
				existing, err := store.List(ctx, "deployments", []Filter{
					{Field: "customer_id", Value: authCtx.UserID},
				}, Page{Limit: 1000, Offset: 0})
				if err == nil {
					// Count non-deleted deployments
					active := 0
					for _, d := range existing {
						if s, _ := d["status"].(string); s != "deleted" {
							active++
						}
					}
					if active >= authCtx.PlanLimits.MaxDeployments {
						return fmt.Errorf("plan limit reached: maximum %d deployments allowed", authCtx.PlanLimits.MaxDeployments)
					}
				}
			}
			// If template_version not set, copy from template
			if _, ok := data["template_version"]; !ok || data["template_version"] == nil || data["template_version"] == "" {
				if tid, ok := toInt64(data["template_id"]); ok && tid > 0 {
					tmpl, err := store.GetByID(ctx, "templates", int(tid))
					if err == nil {
						data["template_version"] = strVal(tmpl["version"])
					}
				}
			}
			return nil
		}
		deplRes.AfterCreate = func(ctx context.Context, authCtx AuthContext, row map[string]any) {
			refID, _ := row["reference_id"].(string)
			if refID != "" && authCtx.UserID > 0 {
				billing.RecordEvent(ctx, store, authCtx.UserID, domain.EventDeploymentCreated, refID, "deployment", nil)
			}
		}
	}

	// Wire cloud provision BeforeCreate: resolve provider from credential + verify ownership + auto-generate SSH key
	if provRes := cfg.Store.Resource("cloud_provisions"); provRes != nil {
		store := cfg.Store
		provRes.BeforeCreate = func(ctx context.Context, authCtx AuthContext, data map[string]any) error {
			credID, ok := toInt64(data["credential_id"])
			if !ok || credID == 0 {
				return fmt.Errorf("credential_id is required")
			}
			cred, err := store.GetByID(ctx, "cloud_credentials", int(credID))
			if err != nil {
				return fmt.Errorf("credential not found")
			}
			// Verify credential belongs to authenticated user
			credOwnerID, ok := toInt64(cred["creator_id"])
			if !ok || int(credOwnerID) != authCtx.UserID {
				return fmt.Errorf("access denied: credential does not belong to you")
			}
			data["provider"] = strVal(cred["provider"])

			instanceName := strVal(data["instance_name"])
			keyName := "cloud-" + instanceName

			// Reuse orphaned SSH key from previous failed provision (restores b880707 fix lost in engine rewrite)
			existing, err := store.List(ctx, "ssh_keys", []Filter{
				{Field: "creator_id", Value: authCtx.UserID},
				{Field: "name", Value: keyName},
			}, Page{Limit: 1})
			if err == nil && len(existing) > 0 {
				data["ssh_key_id"] = strVal(existing[0]["reference_id"])
				return nil
			}

			// Auto-generate SSH key pair for the provision
			privateKeyPEM, publicKey, err := crypto.GenerateSSHKeyPair()
			if err != nil {
				return fmt.Errorf("generate SSH key pair: %w", err)
			}

			fingerprint, err := crypto.GetSSHPublicKeyFingerprint(privateKeyPEM)
			if err != nil {
				return fmt.Errorf("compute SSH key fingerprint: %w", err)
			}

			sshKeyRow, err := store.Create(ctx, "ssh_keys", map[string]any{
				"creator_id":  authCtx.UserID,
				"name":        keyName,
				"private_key": string(privateKeyPEM),
				"public_key":  publicKey,
				"fingerprint": fingerprint,
			})
			if err != nil {
				return fmt.Errorf("create SSH key: %w", err)
			}

			data["ssh_key_id"] = strVal(sshKeyRow["reference_id"])
			return nil
		}
	}

	// Register generic CRUD + state machine routes for all resources
	RegisterRoutes(router, APIConfig{
		Store:          cfg.Store,
		Bus:            cfg.Bus,
		Logger:         cfg.Logger,
		ActionHandlers: buildActionHandlers(cfg),
	})

	// Domain sub-resource routes (require hostname in path, can't use action pattern)
	router.HandleFunc("/api/v1/deployments/{id}/domains/{hostname}", domainRemoveHandler(cfg)).Methods("DELETE")
	router.HandleFunc("/api/v1/deployments/{id}/domains/{hostname}/verify", domainVerifyHandler(cfg)).Methods("POST")

	// Billing endpoints
	router.HandleFunc("/api/v1/billing/verify-payment", verifyPaymentHandler(cfg)).Methods("GET")

	// Serve embedded Web UI for all other paths (SPA pattern)
	router.PathPrefix("/").Handler(spaHandler())

	return router
}

// buildActionHandlers creates custom action handlers beyond standard CRUD.
func buildActionHandlers(cfg SetupConfig) map[string]http.HandlerFunc {
	handlers := map[string]http.HandlerFunc{}

	// Template: publish
	handlers["templates:publish"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		tmpl, err := cfg.Store.Get(ctx, "templates", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}

		// Check ownership — fail closed
		ownerID, ok := toInt64(tmpl["creator_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable creator_id",
				"resource", "templates", "value", tmpl["creator_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		row, err := cfg.Store.Update(ctx, "templates", id, map[string]any{"published": 1})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		res := cfg.Store.Resource("templates")
		stripFields(res, row, cfg.Store, authCtx)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("templates", row),
		})
	}

	// Deployment: start (transition pending → scheduled, triggers schedule command)
	handlers["deployments:start"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		existing, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		// Check ownership — fail closed
		ownerID, ok := toInt64(existing["customer_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable customer_id",
				"resource", "deployments", "value", existing["customer_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		status, _ := existing["status"].(string)

		// Determine target state based on current status
		var targetState string
		switch status {
		case "pending":
			targetState = "scheduled"
		case "scheduled":
			targetState = "starting"
		case "stopped", "failed":
			targetState = "starting"
		default:
			writeError(w, http.StatusConflict, "cannot start deployment in state: "+status)
			return
		}

		row, cmd, err := cfg.Store.Transition(ctx, "deployments", id, targetState)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		// Dispatch command in background so the HTTP response returns immediately.
		// Long-running commands (like StartDeployment) would otherwise block the
		// response and risk context cancellation when the client disconnects.
		if cmd != "" && cfg.Bus != nil {
			go func() {
				bgCtx := context.Background()
				if err := cfg.Bus.Dispatch(bgCtx, cmd, row); err != nil {
					cfg.Logger.Error("command dispatch failed", "command", cmd, "error", err)
				}
			}()
		}

		res := cfg.Store.Resource("deployments")
		stripFields(res, row, cfg.Store, authCtx)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("deployments", row),
		})
	}

	// Deployment: stop (transition running → stopping, triggers stop command)
	handlers["deployments:stop"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		existing, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		ownerID, ok := toInt64(existing["customer_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable customer_id",
				"resource", "deployments", "value", existing["customer_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		row, cmd, err := cfg.Store.Transition(ctx, "deployments", id, "stopping")
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		if cmd != "" && cfg.Bus != nil {
			go func() {
				bgCtx := context.Background()
				if err := cfg.Bus.Dispatch(bgCtx, cmd, row); err != nil {
					cfg.Logger.Error("command dispatch failed", "command", cmd, "error", err)
				}
			}()
		}

		res := cfg.Store.Resource("deployments")
		stripFields(res, row, cfg.Store, authCtx)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("deployments", row),
		})
	}

	// Deployment: monitoring/health
	handlers["deployments:monitoring/health"] = monitoringHandler(cfg, "deployment-health", func(ctx context.Context, cfg SetupConfig, depl map[string]any, r *http.Request) map[string]any {
		refID, _ := depl["reference_id"].(string)
		now := time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"data": map[string]any{
				"type": "deployment-health",
				"id":   refID,
				"attributes": map[string]any{
					"status":     "unknown",
					"containers": []any{},
					"checked_at": now,
				},
			},
		}
	})

	// Deployment: monitoring/stats
	handlers["deployments:monitoring/stats"] = monitoringHandler(cfg, "deployment-stats", func(ctx context.Context, cfg SetupConfig, depl map[string]any, r *http.Request) map[string]any {
		refID, _ := depl["reference_id"].(string)
		now := time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"data": map[string]any{
				"type": "deployment-stats",
				"id":   refID,
				"attributes": map[string]any{
					"containers":   []any{},
					"collected_at": now,
				},
			},
		}
	})

	// Deployment: monitoring/logs
	handlers["deployments:monitoring/logs"] = monitoringHandler(cfg, "deployment-logs", func(ctx context.Context, cfg SetupConfig, depl map[string]any, r *http.Request) map[string]any {
		refID, _ := depl["reference_id"].(string)
		return map[string]any{
			"data": map[string]any{
				"type": "deployment-logs",
				"id":   refID,
				"attributes": map[string]any{
					"logs": []any{},
				},
			},
		}
	})

	// Deployment: domains (list + add, dispatched by HTTP method)
	handlers["deployments:domains"] = domainHandler(cfg)

	// Node: maintenance (enter via POST, exit via DELETE)
	handlers["nodes:maintenance"] = nodeMaintenanceHandler(cfg)

	// Cloud Credentials: regions catalog
	handlers["cloud_credentials:regions"] = cloudCatalogHandler(cfg, func(provider string) any {
		return coreprovider.StaticRegions(provider)
	})

	// Cloud Credentials: sizes catalog
	handlers["cloud_credentials:sizes"] = cloudCatalogHandler(cfg, func(provider string) any {
		return coreprovider.StaticSizes(provider)
	})

	// Invoice: pay (create Stripe Checkout session)
	handlers["invoices:pay"] = invoicePayHandler(cfg)

	// Deployment: monitoring/events
	handlers["deployments:monitoring/events"] = monitoringHandler(cfg, "deployment-events", func(ctx context.Context, cfg SetupConfig, depl map[string]any, r *http.Request) map[string]any {
		refID, _ := depl["reference_id"].(string)
		deplID, _ := toInt64(depl["id"])

		// Query persisted container_events
		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
				limit = n
			}
		}

		query := "SELECT id, type, container, message, timestamp FROM container_events WHERE deployment_id = ? ORDER BY timestamp DESC LIMIT ?"
		args := []any{deplID, limit}

		if eventType := r.URL.Query().Get("type"); eventType != "" {
			query = "SELECT id, type, container, message, timestamp FROM container_events WHERE deployment_id = ? AND type = ? ORDER BY timestamp DESC LIMIT ?"
			args = []any{deplID, eventType, limit}
		}

		rows, err := cfg.Store.RawQuery(ctx, query, args...)
		if err != nil {
			cfg.Logger.Warn("failed to query container events", "deployment", refID, "error", err)
			rows = nil
		}

		events := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			events = append(events, map[string]any{
				"id":        strVal(row["id"]),
				"type":      strVal(row["type"]),
				"container": strVal(row["container"]),
				"message":   strVal(row["message"]),
				"timestamp": strVal(row["timestamp"]),
			})
		}

		return map[string]any{
			"data": map[string]any{
				"type": "deployment-events",
				"id":   refID,
				"attributes": map[string]any{
					"events": events,
				},
			},
		}
	})

	// Cloud Provision: retry (transition failed → pending or failed → destroying)
	handlers["cloud_provisions:retry"] = func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		prov, err := cfg.Store.Get(ctx, "cloud_provisions", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "provision not found")
			return
		}

		ownerID, ok := toInt64(prov["creator_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable creator_id",
				"resource", "cloud_provisions", "value", prov["creator_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		status, _ := prov["status"].(string)
		if status != "failed" {
			writeError(w, http.StatusConflict, "can only retry failed provisions")
			return
		}

		// If the instance was previously created (has provider_instance_id and completed_at),
		// transition to destroying for cleanup; otherwise retry creation from pending.
		targetState := "pending"
		instanceID := strVal(prov["provider_instance_id"])
		completedAt := strVal(prov["completed_at"])
		if instanceID != "" && completedAt != "" {
			targetState = "destroying"
		}

		row, cmd, err := cfg.Store.Transition(ctx, "cloud_provisions", id, targetState)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		if cmd != "" && cfg.Bus != nil {
			if err := cfg.Bus.Dispatch(ctx, cmd, row); err != nil {
				cfg.Logger.Error("command dispatch failed", "command", cmd, "error", err)
			}
		}

		res := cfg.Store.Resource("cloud_provisions")
		stripFields(res, row, cfg.Store, authCtx)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("cloud_provisions", row),
		})
	}

	return handlers
}

// cloudCatalogHandler creates a handler that returns static provider catalog data (regions or sizes)
// for a given cloud credential.
func cloudCatalogHandler(cfg SetupConfig, catalogFn func(provider string) any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		cred, err := cfg.Store.Get(ctx, "cloud_credentials", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "credential not found")
			return
		}

		ownerID, ok := toInt64(cred["creator_id"])
		if !ok {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		provider, _ := cred["provider"].(string)
		data := catalogFn(provider)
		if data == nil {
			data = []any{}
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": data})
	}
}

// =============================================================================
// Node Maintenance Handler
// =============================================================================

// nodeMaintenanceHandler toggles a node in/out of maintenance mode.
// POST = enter maintenance, DELETE = exit maintenance.
func nodeMaintenanceHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		node, err := cfg.Store.Get(ctx, "nodes", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}

		ownerID, ok := toInt64(node["creator_id"])
		if !ok || int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		var newStatus string
		if r.Method == http.MethodPost {
			newStatus = "maintenance"
		} else {
			newStatus = "online"
		}

		row, err := cfg.Store.Update(ctx, "nodes", id, map[string]any{"status": newStatus})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		res := cfg.Store.Resource("nodes")
		stripFields(res, row, cfg.Store, authCtx)
		writeJSON(w, http.StatusOK, map[string]any{
			"data": rowToJSONAPI("nodes", row),
		})
	}
}

// =============================================================================
// Domain Management Handlers
// =============================================================================

// domainHandler handles GET (list) and POST (add) for deployment domains.
func domainHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			domainListHandler(cfg).ServeHTTP(w, r)
		} else {
			domainAddHandler(cfg).ServeHTTP(w, r)
		}
	}
}

// domainListHandler returns domains for a deployment with DNS instructions.
func domainListHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		depl, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		ownerID, ok := toInt64(depl["customer_id"])
		if !ok || int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		domains := parseDomainsList(depl["domains"])

		// Add auto-generated domain
		refID, _ := depl["reference_id"].(string)
		if cfg.BaseDomain != "" && refID != "" {
			autoDomain := DomainInfo{
				Hostname:           refID + ".apps." + cfg.BaseDomain,
				Type:               "auto",
				SSLEnabled:         true,
				VerificationStatus: "verified",
			}
			domains = append([]DomainInfo{autoDomain}, domains...)
		}

		writeJSON(w, http.StatusOK, domains)
	}
}

// domainAddHandler adds a custom domain to a deployment.
func domainAddHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		depl, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		ownerID, ok := toInt64(depl["customer_id"])
		if !ok || int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		var body struct {
			Hostname string `json:"hostname"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Hostname == "" {
			writeError(w, http.StatusBadRequest, "hostname is required")
			return
		}

		domains := parseDomainsList(depl["domains"])

		// Check for duplicates
		for _, d := range domains {
			if d.Hostname == body.Hostname {
				writeError(w, http.StatusConflict, "domain already exists")
				return
			}
		}

		refID, _ := depl["reference_id"].(string)
		newDomain := DomainInfo{
			Hostname:           body.Hostname,
			Type:               "custom",
			SSLEnabled:         false,
			VerificationStatus: "pending",
			VerificationMethod: "cname",
			Instructions: []DNSInstruction{
				{
					Type:     "CNAME",
					Name:     body.Hostname,
					Value:    refID + ".apps." + cfg.BaseDomain,
					Priority: "required",
				},
			},
		}
		domains = append(domains, newDomain)

		domainsJSON, _ := json.Marshal(domains)
		if _, err := cfg.Store.Update(ctx, "deployments", id, map[string]any{"domains": string(domainsJSON)}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update domains")
			return
		}

		writeJSON(w, http.StatusCreated, newDomain)
	}
}

// domainRemoveHandler removes a custom domain from a deployment.
func domainRemoveHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		vars := mux.Vars(r)
		id := vars["id"]
		hostname := vars["hostname"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		depl, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		ownerID, ok := toInt64(depl["customer_id"])
		if !ok || int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		domains := parseDomainsList(depl["domains"])
		found := false
		filtered := make([]DomainInfo, 0, len(domains))
		for _, d := range domains {
			if d.Hostname == hostname {
				found = true
				continue
			}
			filtered = append(filtered, d)
		}

		if !found {
			writeError(w, http.StatusNotFound, "domain not found")
			return
		}

		domainsJSON, _ := json.Marshal(filtered)
		if _, err := cfg.Store.Update(ctx, "deployments", id, map[string]any{"domains": string(domainsJSON)}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update domains")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// domainVerifyHandler checks DNS configuration for a custom domain.
func domainVerifyHandler(cfg SetupConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		vars := mux.Vars(r)
		id := vars["id"]
		hostname := vars["hostname"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		depl, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		ownerID, ok := toInt64(depl["customer_id"])
		if !ok || int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		refID, _ := depl["reference_id"].(string)
		expectedTarget := refID + ".apps." + cfg.BaseDomain

		domains := parseDomainsList(depl["domains"])
		found := false
		for i, d := range domains {
			if d.Hostname != hostname {
				continue
			}
			found = true

			// Check DNS CNAME
			verified := false
			checkErr := ""
			cnames, err := lookupCNAME(hostname)
			if err != nil {
				checkErr = err.Error()
			} else {
				for _, cname := range cnames {
					if strings.TrimSuffix(cname, ".") == expectedTarget {
						verified = true
						break
					}
				}
				if !verified {
					checkErr = "CNAME not pointing to " + expectedTarget
				}
			}

			if verified {
				domains[i].VerificationStatus = "verified"
				domains[i].SSLEnabled = true
				now := time.Now().UTC().Format(time.RFC3339)
				domains[i].VerifiedAt = now
				domains[i].LastCheckError = ""
			} else {
				domains[i].VerificationStatus = "failed"
				domains[i].LastCheckError = checkErr
			}

			domainsJSON, _ := json.Marshal(domains)
			if _, err := cfg.Store.Update(ctx, "deployments", id, map[string]any{"domains": string(domainsJSON)}); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to update domains")
				return
			}

			writeJSON(w, http.StatusOK, domains[i])
			return
		}

		if !found {
			writeError(w, http.StatusNotFound, "domain not found")
		}
	}
}

// Domain types matching the frontend
type DomainInfo struct {
	Hostname           string           `json:"hostname"`
	Type               string           `json:"type"`
	SSLEnabled         bool             `json:"ssl_enabled"`
	VerificationStatus string           `json:"verification_status,omitempty"`
	VerificationMethod string           `json:"verification_method,omitempty"`
	VerifiedAt         string           `json:"verified_at,omitempty"`
	LastCheckError     string           `json:"last_check_error,omitempty"`
	Instructions       []DNSInstruction `json:"instructions,omitempty"`
}

type DNSInstruction struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Priority string `json:"priority"`
}

// parseDomainsList parses the domains JSON field from a deployment row.
// The value may be a string (raw from DB), []byte, or already-parsed Go value
// (after Store.Get parses JSON fields).
func parseDomainsList(v any) []DomainInfo {
	if v == nil {
		return nil
	}
	var raw string
	switch val := v.(type) {
	case string:
		raw = val
	case []byte:
		raw = string(val)
	default:
		// Already parsed by Store.Get — re-marshal to decode into typed struct
		b, err := json.Marshal(val)
		if err != nil {
			return nil
		}
		raw = string(b)
	}
	if raw == "" || raw == "null" {
		return nil
	}
	var domains []DomainInfo
	if err := json.Unmarshal([]byte(raw), &domains); err != nil {
		return nil
	}
	return domains
}

// lookupCNAME performs a DNS CNAME lookup.
func lookupCNAME(hostname string) ([]string, error) {
	cname, err := net.LookupCNAME(hostname)
	if err != nil {
		return nil, err
	}
	return []string{cname}, nil
}

// monitoringHandler creates a handler that verifies auth/ownership then delegates to a builder function.
type monitoringBuilderFunc func(ctx context.Context, cfg SetupConfig, depl map[string]any, r *http.Request) map[string]any

func monitoringHandler(cfg SetupConfig, _ string, builder monitoringBuilderFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := getAuthContext(r)
		id := mux.Vars(r)["id"]

		if !authCtx.Authenticated {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		depl, err := cfg.Store.Get(ctx, "deployments", id)
		if err != nil {
			writeError(w, http.StatusNotFound, "deployment not found")
			return
		}

		// Check ownership — fail closed
		ownerID, ok := toInt64(depl["customer_id"])
		if !ok {
			cfg.Logger.Warn("ownership check failed: unparseable customer_id",
				"resource", "deployments", "value", depl["customer_id"])
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if int(ownerID) != authCtx.UserID {
			writeError(w, http.StatusForbidden, "not authorized")
			return
		}

		result := builder(ctx, cfg, depl, r)
		writeJSON(w, http.StatusOK, result)
	}
}

// =============================================================================
// Middleware
// =============================================================================

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = "req_" + randomString(12)
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered", "error", err)
					writeError(w, http.StatusInternalServerError, "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// =============================================================================
// Health
// =============================================================================

func healthHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "version": version})
	}
}

func readyHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ready",
		"checks": map[string]string{"database": "ok"},
	})
}

// =============================================================================
// SPA Handler
// =============================================================================

func spaHandler() http.Handler {
	distFS, err := fs.Sub(webUI, "webui/dist")
	if err != nil {
		// Fallback if dist not embedded
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html><html><head><title>Hoster</title></head><body><p>Frontend not built. Run: cd web && npm run build</p></body></html>`))
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For API paths that weren't matched, return 404
		if len(r.URL.Path) > 4 && r.URL.Path[:5] == "/api/" {
			writeError(w, http.StatusNotFound, "not found")
			return
		}

		// Try to serve the file directly
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(distFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for all unmatched paths
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[idx.Int64()]
	}
	return string(b)
}
