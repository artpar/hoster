package resources

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/artpar/hoster/internal/core/auth"
	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
	coreprovider "github.com/artpar/hoster/internal/core/provider"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/manyminds/api2go"
	"github.com/manyminds/api2go/jsonapi"
)

// =============================================================================
// CloudCredential JSON:API Model
// =============================================================================

// CloudCredential wraps domain.CloudCredential to implement JSON:API interfaces.
// Note: Encrypted credentials are NEVER included in responses.
type CloudCredential struct {
	ID            string `json:"-"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	DefaultRegion string `json:"default_region,omitempty"`
	CreatorID     string `json:"creator_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	// Credentials is write-only - used on Create, never returned
	Credentials string `json:"credentials,omitempty"`
}

func (c CloudCredential) GetID() string   { return c.ID }
func (c *CloudCredential) SetID(id string) error { c.ID = id; return nil }
func (c CloudCredential) GetName() string  { return "cloud_credentials" }

func (c CloudCredential) GetReferences() []jsonapi.Reference     { return nil }
func (c CloudCredential) GetReferencedIDs() []jsonapi.ReferenceID { return nil }
func (c CloudCredential) GetReferencedStructs() []jsonapi.MarshalIdentifier { return nil }

// CloudCredentialFromDomain converts a domain.CloudCredential to a JSON:API CloudCredential.
func CloudCredentialFromDomain(c *domain.CloudCredential) CloudCredential {
	return CloudCredential{
		ID:            c.ReferenceID,
		Name:          c.Name,
		Provider:      string(c.Provider),
		DefaultRegion: c.DefaultRegion,
		CreatorID:     "",
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// =============================================================================
// CloudCredentialResource - CRUD Operations
// =============================================================================

type CloudCredentialResource struct {
	Store         store.Store
	EncryptionKey []byte
}

func NewCloudCredentialResource(s store.Store, encryptionKey []byte) *CloudCredentialResource {
	return &CloudCredentialResource{Store: s, EncryptionKey: encryptionKey}
}

// FindAll returns all cloud credentials for the authenticated user.
func (r CloudCredentialResource) FindAll(req api2go.Request) (api2go.Responder, error) {
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

	creds, err := r.Store.ListCloudCredentialsByCreator(ctx, authCtx.UserID, opts)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	result := make([]CloudCredential, 0, len(creds))
	for _, c := range creds {
		result = append(result, CloudCredentialFromDomain(&c))
	}

	return &Response{
		Code: http.StatusOK,
		Res:  result,
		Meta: map[string]interface{}{"total": len(result), "limit": opts.Limit, "offset": opts.Offset},
	}, nil
}

// FindOne returns a single cloud credential by ID.
func (r CloudCredentialResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	cred, err := r.Store.GetCloudCredential(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanViewCloudCredential(authCtx, *cred) {
		return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
			fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
	}

	return &Response{Code: http.StatusOK, Res: CloudCredentialFromDomain(cred)}, nil
}

// Create creates a new cloud credential.
func (r CloudCredentialResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	if !auth.CanCreateCloudCredential(authCtx) {
		return &Response{Code: http.StatusUnauthorized}, api2go.NewHTTPError(
			fmt.Errorf("authentication required"), "Authentication required", http.StatusUnauthorized)
	}

	cred, ok := obj.(CloudCredential)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"), "Invalid request body", http.StatusBadRequest)
	}

	if cred.Credentials == "" {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("credentials are required"), "Credentials are required", http.StatusBadRequest)
	}

	// Validate credentials JSON for the provider
	credJSON := []byte(cred.Credentials)
	if err := coreprovider.ValidateCredentialsJSON(cred.Provider, credJSON); err != nil {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(err, err.Error(), http.StatusBadRequest)
	}

	// Encrypt credentials
	if len(r.EncryptionKey) == 0 {
		return &Response{Code: http.StatusInternalServerError}, api2go.NewHTTPError(
			fmt.Errorf("server encryption key not configured"),
			"Server encryption is not configured. Set HOSTER_NODES_ENCRYPTION_KEY.",
			http.StatusInternalServerError)
	}
	encryptedCreds, err := crypto.EncryptSSHKey(credJSON, r.EncryptionKey)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	domainCred, err := domain.NewCloudCredential(
		authCtx.UserID, cred.Name, domain.ProviderType(cred.Provider), encryptedCreds, cred.DefaultRegion,
	)
	if err != nil {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(err, err.Error(), http.StatusBadRequest)
	}

	if err := r.Store.CreateCloudCredential(ctx, domainCred); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusCreated, Res: CloudCredentialFromDomain(domainCred)}, nil
}

// Update is not supported for cloud credentials.
func (r CloudCredentialResource) Update(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	return &Response{Code: http.StatusMethodNotAllowed}, api2go.NewHTTPError(
		fmt.Errorf("cloud credentials cannot be updated"),
		"Cloud credentials cannot be updated. Delete and create a new one.",
		http.StatusMethodNotAllowed)
}

// Delete removes a cloud credential by ID.
func (r CloudCredentialResource) Delete(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	cred, err := r.Store.GetCloudCredential(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanManageCloudCredential(authCtx, *cred) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized"), "Not authorized to delete this credential", http.StatusForbidden)
	}

	if err := r.Store.DeleteCloudCredential(ctx, id); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusNoContent}, nil
}

// =============================================================================
// Custom Actions - Region/Size Listing
// =============================================================================

// ListRegions returns available regions for a credential's provider.
func (r CloudCredentialResource) ListRegions(id string, req *http.Request) (api2go.Responder, error) {
	ctx := req.Context()
	authCtx := auth.FromContext(ctx)

	cred, err := r.Store.GetCloudCredential(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanViewCloudCredential(authCtx, *cred) {
		return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
			fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
	}

	// Return static regions (no API call needed - avoids exposing decrypted creds)
	regions := coreprovider.StaticRegions(string(cred.Provider))

	return &Response{Code: http.StatusOK, Res: regions}, nil
}

// ListSizes returns available instance sizes for a credential's provider.
func (r CloudCredentialResource) ListSizes(id string, req *http.Request) (api2go.Responder, error) {
	ctx := req.Context()
	authCtx := auth.FromContext(ctx)

	cred, err := r.Store.GetCloudCredential(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	if !auth.CanViewCloudCredential(authCtx, *cred) {
		return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
			fmt.Errorf("credential not found"), "Credential not found", http.StatusNotFound)
	}

	// Return static sizes (no API call needed)
	sizes := coreprovider.StaticSizes(string(cred.Provider))

	return &Response{Code: http.StatusOK, Res: sizes}, nil
}
