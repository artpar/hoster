package resources

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/artpar/hoster/internal/core/auth"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/manyminds/api2go"
	"github.com/manyminds/api2go/jsonapi"
)

// =============================================================================
// CloudProvision JSON:API Model
// =============================================================================

type CloudProvision struct {
	ID                 string     `json:"-"`
	CreatorID          string     `json:"creator_id"`
	CredentialID       string     `json:"credential_id"`
	Provider           string     `json:"provider"`
	Status             string     `json:"status"`
	InstanceName       string     `json:"instance_name"`
	Region             string     `json:"region"`
	Size               string     `json:"size"`
	ProviderInstanceID string     `json:"provider_instance_id,omitempty"`
	PublicIP           string     `json:"public_ip,omitempty"`
	NodeID             string     `json:"node_id,omitempty"`
	SSHKeyID           string     `json:"ssh_key_id,omitempty"`
	CurrentStep        string     `json:"current_step,omitempty"`
	ErrorMessage       string     `json:"error_message,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
}

func (p CloudProvision) GetID() string   { return p.ID }
func (p *CloudProvision) SetID(id string) error { p.ID = id; return nil }
func (p CloudProvision) GetName() string  { return "cloud_provisions" }

func (p CloudProvision) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{Type: "cloud_credentials", Name: "credential"},
		{Type: "nodes", Name: "node"},
	}
}

func (p CloudProvision) GetReferencedIDs() []jsonapi.ReferenceID {
	refs := []jsonapi.ReferenceID{}
	if p.CredentialID != "" {
		refs = append(refs, jsonapi.ReferenceID{ID: p.CredentialID, Type: "cloud_credentials", Name: "credential"})
	}
	if p.NodeID != "" {
		refs = append(refs, jsonapi.ReferenceID{ID: p.NodeID, Type: "nodes", Name: "node"})
	}
	return refs
}

func (p CloudProvision) GetReferencedStructs() []jsonapi.MarshalIdentifier { return nil }

func CloudProvisionFromDomain(p *domain.CloudProvision) CloudProvision {
	return CloudProvision{
		ID:                 p.ID,
		CreatorID:          p.CreatorID,
		CredentialID:       p.CredentialID,
		Provider:           string(p.Provider),
		Status:             string(p.Status),
		InstanceName:       p.InstanceName,
		Region:             p.Region,
		Size:               p.Size,
		ProviderInstanceID: p.ProviderInstanceID,
		PublicIP:           p.PublicIP,
		NodeID:             p.NodeID,
		SSHKeyID:           p.SSHKeyID,
		CurrentStep:        p.CurrentStep,
		ErrorMessage:       p.ErrorMessage,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		CompletedAt:        p.CompletedAt,
	}
}

// =============================================================================
// CloudProvisionResource - CRUD Operations
// =============================================================================

type CloudProvisionResource struct {
	Store store.Store
}

func NewCloudProvisionResource(s store.Store) *CloudProvisionResource {
	return &CloudProvisionResource{Store: s}
}

// FindAll returns all cloud provisions for the authenticated user.
func (r CloudProvisionResource) FindAll(req api2go.Request) (api2go.Responder, error) {
	opts := store.DefaultListOptions()
	if limit, ok := req.QueryParams["page[size]"]; ok && len(limit) > 0 {
		if l, err := strconv.Atoi(limit[0]); err == nil {
			opts.Limit = l
		}
	}
	if offset, ok := req.QueryParams["page[offset]"]; ok && len(offset) > 0 {
		if o, err := strconv.Atoi(offset[0]); err == nil {
			opts.Offset = o
		}
	}

	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)
	if !authCtx.Authenticated {
		return &Response{Code: http.StatusUnauthorized}, api2go.NewHTTPError(
			fmt.Errorf("authentication required"), "Authentication required", http.StatusUnauthorized)
	}

	provisions, err := r.Store.ListCloudProvisionsByCreator(ctx, authCtx.UserID, opts)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	result := make([]CloudProvision, 0, len(provisions))
	for _, p := range provisions {
		result = append(result, CloudProvisionFromDomain(&p))
	}

	return &Response{
		Code: http.StatusOK,
		Res:  result,
		Meta: map[string]interface{}{"total": len(result), "limit": opts.Limit, "offset": opts.Offset},
	}, nil
}

// FindOne returns a single cloud provision by ID.
func (r CloudProvisionResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	prov, err := r.Store.GetCloudProvision(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("provision not found"), "Provision not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanViewCloudProvision(authCtx, *prov) {
		return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
			fmt.Errorf("provision not found"), "Provision not found", http.StatusNotFound)
	}

	return &Response{Code: http.StatusOK, Res: CloudProvisionFromDomain(prov)}, nil
}

// Create starts a new cloud provisioning job.
func (r CloudProvisionResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	if !auth.CanCreateCloudProvision(authCtx) {
		return &Response{Code: http.StatusUnauthorized}, api2go.NewHTTPError(
			fmt.Errorf("authentication required"), "Authentication required", http.StatusUnauthorized)
	}

	prov, ok := obj.(CloudProvision)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"), "Invalid request body", http.StatusBadRequest)
	}

	// Verify the credential belongs to this user
	cred, err := r.Store.GetCloudCredential(ctx, prov.CredentialID)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				fmt.Errorf("credential not found"), "Credential not found", http.StatusBadRequest)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}
	if !auth.CanViewCloudCredential(authCtx, *cred) {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("credential not found"), "Credential not found", http.StatusBadRequest)
	}

	domainProv, err := domain.NewCloudProvision(
		authCtx.UserID, prov.CredentialID, cred.Provider, prov.InstanceName, prov.Region, prov.Size,
	)
	if err != nil {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(err, err.Error(), http.StatusBadRequest)
	}

	if err := r.Store.CreateCloudProvision(ctx, domainProv); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusCreated, Res: CloudProvisionFromDomain(domainProv)}, nil
}

// Update is not supported - provisions are managed via custom actions.
func (r CloudProvisionResource) Update(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	return &Response{Code: http.StatusMethodNotAllowed}, api2go.NewHTTPError(
		fmt.Errorf("provisions cannot be updated directly"),
		"Use /destroy or /retry actions instead",
		http.StatusMethodNotAllowed)
}

// Delete initiates destruction of a cloud provision's instance.
func (r CloudProvisionResource) Delete(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	prov, err := r.Store.GetCloudProvision(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("provision not found"), "Provision not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanManageCloudProvision(authCtx, *prov) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized"), "Not authorized", http.StatusForbidden)
	}

	// Transition to destroying - the provisioner worker will handle actual destruction
	if err := prov.Transition(domain.ProvisionStatusDestroying); err != nil {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			err, fmt.Sprintf("Cannot destroy provision in %s status", prov.Status), http.StatusConflict)
	}
	prov.SetStep("Scheduled for destruction")

	if err := r.Store.UpdateCloudProvision(ctx, prov); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusOK, Res: CloudProvisionFromDomain(prov)}, nil
}

// =============================================================================
// Custom Actions
// =============================================================================

// RetryProvision retries a failed provision.
func (r CloudProvisionResource) RetryProvision(id string, req *http.Request) (api2go.Responder, error) {
	ctx := req.Context()
	authCtx := auth.FromContext(ctx)

	prov, err := r.Store.GetCloudProvision(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("provision not found"), "Provision not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanManageCloudProvision(authCtx, *prov) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized"), "Not authorized", http.StatusForbidden)
	}

	if err := prov.Transition(domain.ProvisionStatusPending); err != nil {
		return &Response{Code: http.StatusConflict}, api2go.NewHTTPError(
			err, fmt.Sprintf("Cannot retry provision in %s status", prov.Status), http.StatusConflict)
	}

	if err := r.Store.UpdateCloudProvision(ctx, prov); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusOK, Res: CloudProvisionFromDomain(prov)}, nil
}
