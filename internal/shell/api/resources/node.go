// Package resources provides JSON:API resource implementations for the Hoster API.
// Following ADR-003: JSON:API Standard with api2go
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
// Node JSON:API Model
// =============================================================================

// Node wraps domain.Node to implement JSON:API interfaces.
type Node struct {
	ID              string              `json:"-"`
	Name            string              `json:"name"`
	SSHHost         string              `json:"ssh_host"`
	SSHPort         int                 `json:"ssh_port"`
	SSHUser         string              `json:"ssh_user"`
	SSHKeyID        string              `json:"ssh_key_id,omitempty"`
	DockerSocket    string              `json:"docker_socket"`
	Status          string              `json:"status"`
	Capabilities    []string            `json:"capabilities"`
	Capacity        domain.NodeCapacity `json:"capacity"`
	Location        string              `json:"location,omitempty"`
	LastHealthCheck *time.Time          `json:"last_health_check,omitempty"`
	ErrorMessage    string              `json:"error_message,omitempty"`
	ProviderType    string              `json:"provider_type,omitempty"`
	ProvisionID     string              `json:"provision_id,omitempty"`
	CreatorID       string              `json:"creator_id"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

// GetID returns the node ID for JSON:API.
func (n Node) GetID() string {
	return n.ID
}

// SetID sets the node ID for JSON:API.
func (n *Node) SetID(id string) error {
	n.ID = id
	return nil
}

// GetName returns the JSON:API resource type name.
func (n Node) GetName() string {
	return "nodes"
}

// GetReferences returns the relationships this resource has.
func (n Node) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "ssh_keys",
			Name: "ssh_key",
		},
		{
			Type: "deployments",
			Name: "deployments",
		},
	}
}

// GetReferencedIDs returns IDs of referenced resources.
func (n Node) GetReferencedIDs() []jsonapi.ReferenceID {
	refs := []jsonapi.ReferenceID{}
	if n.SSHKeyID != "" {
		refs = append(refs, jsonapi.ReferenceID{
			ID:   n.SSHKeyID,
			Type: "ssh_keys",
			Name: "ssh_key",
		})
	}
	return refs
}

// GetReferencedStructs returns the actual referenced objects.
func (n Node) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return nil
}

// =============================================================================
// Conversion Functions
// =============================================================================

// NodeFromDomain converts a domain.Node to a JSON:API Node.
func NodeFromDomain(n *domain.Node) Node {
	return Node{
		ID:              n.ReferenceID,
		Name:            n.Name,
		SSHHost:         n.SSHHost,
		SSHPort:         n.SSHPort,
		SSHUser:         n.SSHUser,
		SSHKeyID:        n.SSHKeyRefID,
		DockerSocket:    n.DockerSocket,
		Status:          string(n.Status),
		Capabilities:    n.Capabilities,
		Capacity:        n.Capacity,
		Location:        n.Location,
		LastHealthCheck: n.LastHealthCheck,
		ErrorMessage:    n.ErrorMessage,
		ProviderType:    n.ProviderType,
		ProvisionID:     n.ProvisionID,
		CreatorID:       "",
		CreatedAt:       n.CreatedAt,
		UpdatedAt:       n.UpdatedAt,
	}
}

// ToDomain converts the JSON:API Node to a domain.Node.
func (n Node) ToDomain() *domain.Node {
	return &domain.Node{
		ReferenceID:     n.ID,
		Name:            n.Name,
		SSHHost:         n.SSHHost,
		SSHPort:         n.SSHPort,
		SSHUser:         n.SSHUser,
		SSHKeyRefID:     n.SSHKeyID,
		DockerSocket:    n.DockerSocket,
		Status:          domain.NodeStatus(n.Status),
		Capabilities:    n.Capabilities,
		Capacity:        n.Capacity,
		Location:        n.Location,
		LastHealthCheck: n.LastHealthCheck,
		ErrorMessage:    n.ErrorMessage,
		ProviderType:    n.ProviderType,
		ProvisionID:     n.ProvisionID,
		CreatedAt:       n.CreatedAt,
		UpdatedAt:       n.UpdatedAt,
	}
}

// =============================================================================
// NodeResource - CRUD Operations
// =============================================================================

// NodeResource implements the api2go resource interface for nodes.
type NodeResource struct {
	Store store.Store
}

// NewNodeResource creates a new node resource handler.
func NewNodeResource(s store.Store) *NodeResource {
	return &NodeResource{Store: s}
}

// FindAll returns all nodes for the authenticated user.
// GET /api/v1/nodes
// Auth: Only returns nodes belonging to the authenticated user
func (r NodeResource) FindAll(req api2go.Request) (api2go.Responder, error) {
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

	// Only list nodes belonging to the authenticated user
	nodes, err := r.Store.ListNodesByCreator(ctx, authCtx.UserID, opts)
	if err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	result := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, NodeFromDomain(&n))
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

// FindOne returns a single node by ID.
// GET /api/v1/nodes/{id}
// Auth: Only node creator can view
func (r NodeResource) FindOne(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	node, err := r.Store.GetNode(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("node not found"),
				"Node not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if user can view this node
	if !auth.CanViewNode(authCtx, *node) {
		return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
			fmt.Errorf("node not found"),
			"Node not found",
			http.StatusNotFound,
		)
	}

	return &Response{
		Code: http.StatusOK,
		Res:  NodeFromDomain(node),
	}, nil
}

// Create creates a new node.
// POST /api/v1/nodes
// Auth: Requires authentication. CreatorID is set from auth context.
func (r NodeResource) Create(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	// Require authentication
	if !auth.CanCreateNode(authCtx) {
		return &Response{Code: http.StatusUnauthorized}, api2go.NewHTTPError(
			fmt.Errorf("authentication required"),
			"Authentication required",
			http.StatusUnauthorized,
		)
	}

	node, ok := obj.(Node)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"),
			"Invalid request body",
			http.StatusBadRequest,
		)
	}

	// Set defaults
	if node.SSHPort == 0 {
		node.SSHPort = 22
	}
	if node.DockerSocket == "" {
		node.DockerSocket = "/var/run/docker.sock"
	}
	if len(node.Capabilities) == 0 {
		node.Capabilities = domain.DefaultCapabilities()
	}

	// Create domain node with validation
	domainNode, err := domain.NewNode(
		authCtx.UserID, // CreatorID from auth context
		node.Name,
		node.SSHHost,
		node.SSHUser,
		node.SSHPort,
		node.Capabilities,
	)
	if err != nil {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			err,
			err.Error(),
			http.StatusBadRequest,
		)
	}

	// Apply optional fields
	if node.SSHKeyID != "" {
		sshKey, err := r.Store.GetSSHKey(ctx, node.SSHKeyID)
		if err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				fmt.Errorf("SSH key not found"),
				"SSH key not found",
				http.StatusBadRequest,
			)
		}
		domainNode.SSHKeyID = sshKey.ID
		domainNode.SSHKeyRefID = sshKey.ReferenceID
	}
	domainNode.DockerSocket = node.DockerSocket
	domainNode.Location = node.Location

	if err := r.Store.CreateNode(ctx, domainNode); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusCreated,
		Res:  NodeFromDomain(domainNode),
	}, nil
}

// Update updates an existing node.
// PATCH /api/v1/nodes/{id}
// Auth: Only creator can update their nodes
func (r NodeResource) Update(obj interface{}, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	node, ok := obj.(Node)
	if !ok {
		return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
			fmt.Errorf("invalid request body"),
			"Invalid request body",
			http.StatusBadRequest,
		)
	}

	existing, err := r.Store.GetNode(ctx, node.ID)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("node not found"),
				"Node not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if user can manage this node
	if !auth.CanManageNode(authCtx, *existing) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized to modify this node"),
			"Not authorized to modify this node",
			http.StatusForbidden,
		)
	}

	// Apply updates (only non-empty/non-zero fields)
	if node.Name != "" {
		if err := domain.ValidateNodeName(node.Name); err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				err,
				err.Error(),
				http.StatusBadRequest,
			)
		}
		existing.Name = node.Name
	}
	if node.SSHHost != "" {
		if err := domain.ValidateSSHHost(node.SSHHost); err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				err,
				err.Error(),
				http.StatusBadRequest,
			)
		}
		existing.SSHHost = node.SSHHost
	}
	if node.SSHPort > 0 {
		if err := domain.ValidateSSHPort(node.SSHPort); err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				err,
				err.Error(),
				http.StatusBadRequest,
			)
		}
		existing.SSHPort = node.SSHPort
	}
	if node.SSHUser != "" {
		if err := domain.ValidateSSHUser(node.SSHUser); err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				err,
				err.Error(),
				http.StatusBadRequest,
			)
		}
		existing.SSHUser = node.SSHUser
	}
	if len(node.Capabilities) > 0 {
		if err := domain.ValidateCapabilities(node.Capabilities); err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				err,
				err.Error(),
				http.StatusBadRequest,
			)
		}
		existing.Capabilities = node.Capabilities
	}
	if node.SSHKeyID != "" {
		sshKey, err := r.Store.GetSSHKey(ctx, node.SSHKeyID)
		if err != nil {
			return &Response{Code: http.StatusBadRequest}, api2go.NewHTTPError(
				fmt.Errorf("SSH key not found"),
				"SSH key not found",
				http.StatusBadRequest,
			)
		}
		existing.SSHKeyID = sshKey.ID
		existing.SSHKeyRefID = sshKey.ReferenceID
	}
	if node.DockerSocket != "" {
		existing.DockerSocket = node.DockerSocket
	}
	if node.Location != "" {
		existing.Location = node.Location
	}
	existing.UpdatedAt = time.Now()

	if err := r.Store.UpdateNode(ctx, existing); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusOK,
		Res:  NodeFromDomain(existing),
	}, nil
}

// Delete removes a node by ID.
// DELETE /api/v1/nodes/{id}
// Auth: Only creator can delete their nodes
func (r NodeResource) Delete(id string, req api2go.Request) (api2go.Responder, error) {
	ctx := req.PlainRequest.Context()
	authCtx := auth.FromContext(ctx)

	// Check if node exists
	node, err := r.Store.GetNode(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("node not found"),
				"Node not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if user can delete this node
	if !auth.CanManageNode(authCtx, *node) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized to delete this node"),
			"Not authorized to delete this node",
			http.StatusForbidden,
		)
	}

	// TODO: Check for active deployments on this node before deleting
	// For now, just delete

	if err := r.Store.DeleteNode(ctx, id); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{Code: http.StatusNoContent}, nil
}

// =============================================================================
// Custom Actions
// =============================================================================

// SetMaintenance sets or clears maintenance mode on a node.
// POST /api/v1/nodes/{id}/maintenance
// Auth: Only creator can set maintenance mode
func (r NodeResource) SetMaintenance(id string, maintenance bool, req *http.Request) (api2go.Responder, error) {
	ctx := req.Context()
	authCtx := auth.FromContext(ctx)

	node, err := r.Store.GetNode(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return &Response{Code: http.StatusNotFound}, api2go.NewHTTPError(
				fmt.Errorf("node not found"),
				"Node not found",
				http.StatusNotFound,
			)
		}
		return &Response{Code: http.StatusInternalServerError}, err
	}

	// Check if user can manage this node
	if !auth.CanManageNode(authCtx, *node) {
		return &Response{Code: http.StatusForbidden}, api2go.NewHTTPError(
			fmt.Errorf("not authorized to manage this node"),
			"Not authorized to manage this node",
			http.StatusForbidden,
		)
	}

	if maintenance {
		node.Status = domain.NodeStatusMaintenance
	} else {
		// When exiting maintenance, set to offline until next health check
		node.Status = domain.NodeStatusOffline
	}
	node.UpdatedAt = time.Now()

	if err := r.Store.UpdateNode(ctx, node); err != nil {
		return &Response{Code: http.StatusInternalServerError}, err
	}

	return &Response{
		Code: http.StatusOK,
		Res:  NodeFromDomain(node),
	}, nil
}
