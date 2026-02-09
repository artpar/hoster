// Package resources provides JSON:API resource implementations for the Hoster API.
// Following ADR-003: JSON:API Standard with api2go
package resources

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/artpar/hoster/internal/core/auth"
	"github.com/artpar/hoster/internal/core/crypto"
	"github.com/artpar/hoster/internal/core/domain"
	"github.com/artpar/hoster/internal/shell/store"
	"github.com/manyminds/api2go"
	"github.com/manyminds/api2go/jsonapi"
)

// =============================================================================
// SSH Key JSON:API Model
// =============================================================================

// SSHKey wraps domain.SSHKey to implement JSON:API interfaces.
// Note: The private key is NEVER included in responses.
type SSHKey struct {
	ID          string    `json:"-"`
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"`
	CreatorID   string    `json:"creator_id"`
	CreatedAt   time.Time `json:"created_at"`
	// PrivateKey is write-only - used on Create, never returned
	PrivateKey string `json:"private_key,omitempty"`
}

// GetID returns the SSH key ID for JSON:API.
func (k SSHKey) GetID() string {
	return k.ID
}

// SetID sets the SSH key ID for JSON:API.
func (k *SSHKey) SetID(id string) error {
	k.ID = id
	return nil
}

// GetName returns the JSON:API resource type name.
func (k SSHKey) GetName() string {
	return "ssh_keys"
}

// GetReferences returns the relationships this resource has.
func (k SSHKey) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "nodes",
			Name: "nodes",
		},
	}
}

// GetReferencedIDs returns IDs of referenced resources.
func (k SSHKey) GetReferencedIDs() []jsonapi.ReferenceID {
	return nil
}

// GetReferencedStructs returns the actual referenced objects.
func (k SSHKey) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

// SSHKeyFromDomain converts a domain.SSHKey to a JSON:API SSHKey.
// Note: The encrypted private key is NEVER included in the response.
func SSHKeyFromDomain(k *domain.SSHKey) SSHKey {
	return SSHKey{
		ID:          k.ReferenceID,
		Name:        k.Name,
		Fingerprint: k.Fingerprint,
		CreatorID:   "",
		CreatedAt:   k.CreatedAt,
		// PrivateKey is intentionally NOT included for security
	}
}

// =============================================================================
// SSHKeyResource - CRUD Operations
// =============================================================================

// SSHKeyResource implements the api2go resource interface for SSH keys.
type SSHKeyResource struct {
	Store         store.Store
	EncryptionKey []byte // Key for encrypting SSH private keys
}

// NewSSHKeyResource creates a new SSH key resource handler.
func NewSSHKeyResource(s store.Store, encryptionKey []byte) *SSHKeyResource {
	return &SSHKeyResource{
		Store:         s,
		EncryptionKey: encryptionKey,
	}
}

// FindAll returns all SSH keys for the authenticated user.
// GET /api/v1/ssh_keys
// Auth: Only returns keys belonging to the authenticated user
func (r SSHKeyResource) FindAll(req api2go.Request) (api2go.Responder, error) {
	opts := store.DefaultListOptions()

	// Parse pagination
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
	if pageNum, ok := req.QueryParams["page[number]"]; ok && len(pageNum) > 0 {
		if pn, err := strconv.Atoi(pageNum[0]); err == nil && pn > 0 {
			opts.Offset = (pn - 1) * opts.Limit
		}
	}

	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	// Require authentication
	if !authCtx.Authenticated {
		return &Response{Code: http.StatusUnauthorized}, api2go.NewHTTPError(
			fmt.Errorf("authentication required"),
			"Authentication required",
			http.StatusUnauthorized,
		)
	}

	// Only list keys belonging to the authenticated user
	keys, err := r.Store.ListSSHKeysByCreator(ctx, authCtx.UserID, opts)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	result := make([]SSHKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, SSHKeyFromDomain(&k))
	}

	return &Response{
		Code: http.StatusOK,
		Res:  result,
		Meta: map[string]interface{}{
			"total":  len(result),
			"limit":  opts.Limit,
			"offset": opts.Offset,
		},
	}, nil
}

// FindOne returns a single SSH key by ID.
// GET /api/v1/ssh_keys/{id}
// Auth: Only key creator can view
func (r SSHKeyResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	key, err := r.Store.GetSSHKey(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("SSH key not found"),
				"SSH key not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if user can view this key
	if !auth.CanViewSSHKey(authCtx, *key) {
		return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
			fmt.Errorf("SSH key not found"),
			"SSH key not found",
			http.StatusNotFound,
		)
	}

	return &Response{
		Code: http.StatusOK,
		Res:  SSHKeyFromDomain(key),
	}, nil
}

// Create creates a new SSH key.
// POST /api/v1/ssh_keys
// Auth: Requires authentication. CreatorID is set from auth context.
// The private key is validated, encrypted, and stored.
// The fingerprint is calculated and stored.
// The private key is NEVER returned.
func (r SSHKeyResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	// Require authentication
	if !auth.CanCreateSSHKey(authCtx) {
		return &Response{Code: http.StatusUnauthorized}, api2go.NewHTTPError(
			fmt.Errorf("authentication required"),
			"Authentication required",
			http.StatusUnauthorized,
		)
	}

	sshKey, ok := obj.(SSHKey)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"),
			"Invalid request body",
			http.StatusBadRequest,
		)
	}

	// Validate name
	if sshKey.Name == "" {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("SSH key name is required"),
			"SSH key name is required",
			http.StatusBadRequest,
		)
	}

	// Validate private key
	if sshKey.PrivateKey == "" {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("private key is required"),
			"Private key is required",
			http.StatusBadRequest,
		)
	}

	privateKeyBytes := []byte(sshKey.PrivateKey)

	// Validate SSH key format
	if err := crypto.ValidateSSHPrivateKey(privateKeyBytes); err != nil {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid SSH private key"),
			"Invalid SSH private key format",
			http.StatusBadRequest,
		)
	}

	// Calculate fingerprint
	fingerprint, err := crypto.GetSSHPublicKeyFingerprint(privateKeyBytes)
	if err != nil {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			err,
			"Failed to generate key fingerprint",
			http.StatusBadRequest,
		)
	}

	// Encrypt the private key
	encryptedKey, err := crypto.EncryptSSHKey(privateKeyBytes, r.EncryptionKey)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Create domain SSH key
	domainKey := &domain.SSHKey{
		ReferenceID:         domain.GenerateSSHKeyID(),
		CreatorID:           authCtx.UserID,
		Name:                sshKey.Name,
		PrivateKeyEncrypted: encryptedKey,
		Fingerprint:         fingerprint,
		CreatedAt:           time.Now(),
	}

	if err := r.Store.CreateSSHKey(ctx, domainKey); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Return without the private key
	return &Response{
		Code: http.StatusCreated,
		Res:  SSHKeyFromDomain(domainKey),
	}, nil
}

// Update is not supported for SSH keys.
// SSH keys are immutable once created. Delete and create a new one instead.
func (r SSHKeyResource) Update(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	return &Response{Code: http.StatusMethodNotAllowed}, api2go.NewHTTPError(
		fmt.Errorf("SSH keys cannot be updated"),
		"SSH keys cannot be updated. Delete and create a new one.",
		http.StatusMethodNotAllowed,
	)
}

// Delete removes an SSH key by ID.
// DELETE /api/v1/ssh_keys/{id}
// Auth: Only creator can delete their SSH keys
func (r SSHKeyResource) Delete(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	// Check if key exists
	key, err := r.Store.GetSSHKey(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("SSH key not found"),
				"SSH key not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if user can delete this key
	if !auth.CanManageSSHKey(authCtx, *key) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized to delete this SSH key"),
			"Not authorized to delete this SSH key",
			http.StatusForbidden,
		)
	}

	// TODO: Check if any nodes are using this key before deleting
	// For now, just delete

	if err := r.Store.DeleteSSHKey(ctx, id); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusNoContent}, nil
}
