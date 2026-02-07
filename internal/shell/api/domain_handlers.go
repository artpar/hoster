package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/artpar/hoster/internal/core/auth"
	coredns "github.com/artpar/hoster/internal/core/dns"
	"github.com/artpar/hoster/internal/core/domain"
	shelldns "github.com/artpar/hoster/internal/shell/dns"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/gorilla/mux"
)

// =============================================================================
// Domain Management Handlers
// =============================================================================

// DomainHandlers provides custom domain management endpoints for deployments.
type DomainHandlers struct {
	store    store.Store
	resolver *shelldns.Resolver
}

// NewDomainHandlers creates a new domain handlers instance.
func NewDomainHandlers(s store.Store) *DomainHandlers {
	return &DomainHandlers{
		store:    s,
		resolver: shelldns.NewResolver(),
	}
}

// RegisterRoutes registers the domain management routes.
func (h *DomainHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/deployments/{id}/domains", h.ListDomains).Methods("GET")
	r.HandleFunc("/api/v1/deployments/{id}/domains", h.AddDomain).Methods("POST")
	r.HandleFunc("/api/v1/deployments/{id}/domains/{hostname}", h.RemoveDomain).Methods("DELETE")
	r.HandleFunc("/api/v1/deployments/{id}/domains/{hostname}/verify", h.VerifyDomain).Methods("POST")
}

// =============================================================================
// List Domains
// =============================================================================

type domainResponse struct {
	Hostname           string                          `json:"hostname"`
	Type               string                          `json:"type"`
	SSLEnabled         bool                            `json:"ssl_enabled"`
	VerificationStatus string                          `json:"verification_status,omitempty"`
	VerificationMethod string                          `json:"verification_method,omitempty"`
	VerifiedAt         *time.Time                      `json:"verified_at,omitempty"`
	LastCheckError     string                          `json:"last_check_error,omitempty"`
	Instructions       []coredns.DNSInstruction        `json:"instructions,omitempty"`
}

// ListDomains returns all domains for a deployment with DNS instructions.
func (h *DomainHandlers) ListDomains(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	id := vars["id"]

	dep, err := h.store.GetDeployment(ctx, id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	if !auth.CanViewDeployment(authCtx, *dep) {
		writeJSONError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	autoDomain := coredns.FindAutoDomain(dep.Domains)

	// Get node IP for DNS instructions
	var nodeIP string
	if dep.NodeID != "" {
		node, err := h.store.GetNode(ctx, dep.NodeID)
		if err == nil {
			nodeIP = node.SSHHost
		}
	}

	result := make([]domainResponse, 0, len(dep.Domains))
	for _, d := range dep.Domains {
		resp := domainResponse{
			Hostname:           d.Hostname,
			Type:               string(d.Type),
			SSLEnabled:         d.SSLEnabled,
			VerificationStatus: string(d.VerificationStatus),
			VerificationMethod: string(d.VerificationMethod),
			VerifiedAt:         d.VerifiedAt,
			LastCheckError:     d.LastCheckError,
		}

		// Add DNS instructions for unverified custom domains
		if d.Type == domain.DomainTypeCustom && d.VerificationStatus != domain.DomainVerificationVerified {
			resp.Instructions = coredns.GenerateInstructions(d.Hostname, autoDomain, nodeIP)
		}

		result = append(result, resp)
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": result,
	})
}

// =============================================================================
// Add Domain
// =============================================================================

type addDomainRequest struct {
	Hostname string `json:"hostname"`
}

// AddDomain adds a custom domain to a deployment.
func (h *DomainHandlers) AddDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	id := vars["id"]

	dep, err := h.store.GetDeployment(ctx, id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	if !auth.CanManageDeployment(authCtx, *dep) {
		writeJSONError(w, http.StatusForbidden, "Not authorized")
		return
	}

	var req addDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	hostname := strings.TrimSpace(strings.ToLower(req.Hostname))

	// Validate hostname format
	if err := coredns.ValidateCustomDomain(hostname); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if domain can be added
	if err := coredns.CanAddCustomDomain(dep.Domains, hostname); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}

	// Add the custom domain
	dep.Domains = append(dep.Domains, domain.NewCustomDomain(hostname))
	dep.UpdatedAt = time.Now()

	if err := h.store.UpdateDeployment(ctx, dep); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update deployment")
		return
	}

	// Return the new domain with instructions
	autoDomain := coredns.FindAutoDomain(dep.Domains)
	var nodeIP string
	if dep.NodeID != "" {
		node, err := h.store.GetNode(ctx, dep.NodeID)
		if err == nil {
			nodeIP = node.SSHHost
		}
	}

	resp := domainResponse{
		Hostname:           hostname,
		Type:               string(domain.DomainTypeCustom),
		VerificationStatus: string(domain.DomainVerificationPending),
		Instructions:       coredns.GenerateInstructions(hostname, autoDomain, nodeIP),
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": resp,
	})
}

// =============================================================================
// Remove Domain
// =============================================================================

// RemoveDomain removes a custom domain from a deployment.
func (h *DomainHandlers) RemoveDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	id := vars["id"]
	hostname := vars["hostname"]

	dep, err := h.store.GetDeployment(ctx, id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	if !auth.CanManageDeployment(authCtx, *dep) {
		writeJSONError(w, http.StatusForbidden, "Not authorized")
		return
	}

	// Find and remove the custom domain
	found := false
	newDomains := make([]domain.Domain, 0, len(dep.Domains))
	for _, d := range dep.Domains {
		if strings.EqualFold(d.Hostname, hostname) {
			if d.Type == domain.DomainTypeAuto {
				writeJSONError(w, http.StatusBadRequest, "Cannot remove auto-generated domain")
				return
			}
			found = true
			continue
		}
		newDomains = append(newDomains, d)
	}

	if !found {
		writeJSONError(w, http.StatusNotFound, "Domain not found on this deployment")
		return
	}

	dep.Domains = newDomains
	dep.UpdatedAt = time.Now()

	if err := h.store.UpdateDeployment(ctx, dep); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update deployment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// Verify Domain
// =============================================================================

// VerifyDomain triggers immediate DNS verification for a domain.
func (h *DomainHandlers) VerifyDomain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.FromContext(ctx)
	vars := mux.Vars(r)
	id := vars["id"]
	hostname := vars["hostname"]

	dep, err := h.store.GetDeployment(ctx, id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	if !auth.CanManageDeployment(authCtx, *dep) {
		writeJSONError(w, http.StatusForbidden, "Not authorized")
		return
	}

	// Find the domain
	var domIdx int = -1
	for i, d := range dep.Domains {
		if strings.EqualFold(d.Hostname, hostname) && d.Type == domain.DomainTypeCustom {
			domIdx = i
			break
		}
	}

	if domIdx == -1 {
		writeJSONError(w, http.StatusNotFound, "Custom domain not found on this deployment")
		return
	}

	dom := &dep.Domains[domIdx]
	autoDomain := coredns.FindAutoDomain(dep.Domains)

	// Get node IP
	var expectedIPs []string
	if dep.NodeID != "" {
		node, err := h.store.GetNode(ctx, dep.NodeID)
		if err == nil && node.SSHHost != "" {
			expectedIPs = []string{node.SSHHost}
		}
	}

	// Resolve and verify
	input := h.resolver.Resolve(ctx, dom.Hostname)
	result := coredns.Verify(input, autoDomain, expectedIPs)

	if result.Verified {
		dom.VerificationStatus = domain.DomainVerificationVerified
		dom.VerificationMethod = result.Method
		now := time.Now()
		dom.VerifiedAt = &now
		dom.LastCheckError = ""
	} else {
		dom.VerificationStatus = domain.DomainVerificationFailed
		dom.LastCheckError = result.Error
	}

	if err := h.store.UpdateDeployment(ctx, dep); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update deployment")
		return
	}

	resp := domainResponse{
		Hostname:           dom.Hostname,
		Type:               string(dom.Type),
		SSLEnabled:         dom.SSLEnabled,
		VerificationStatus: string(dom.VerificationStatus),
		VerificationMethod: string(dom.VerificationMethod),
		VerifiedAt:         dom.VerifiedAt,
		LastCheckError:     dom.LastCheckError,
	}

	w.Header().Set("Content-Type", "application/vnd.api+json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": resp,
	})
}

// =============================================================================
// Helpers
// =============================================================================

func writeJSONError(w http.ResponseWriter, status int, message string) {
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
