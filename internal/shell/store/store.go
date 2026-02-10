package store

import (
	"context"
	"time"

	"github.com/artpar/hoster/internal/core/domain"
)

// =============================================================================
// Store Interface
// =============================================================================

// Store defines the persistence interface for Hoster entities.
type Store interface {
	// User resolution (upsert user from APIGate reference ID)
	ResolveUser(ctx context.Context, referenceID, email, name, planID string) (int, error)

	// Template operations
	CreateTemplate(ctx context.Context, template *domain.Template) error
	GetTemplate(ctx context.Context, id string) (*domain.Template, error)
	GetTemplateBySlug(ctx context.Context, slug string) (*domain.Template, error)
	UpdateTemplate(ctx context.Context, template *domain.Template) error
	DeleteTemplate(ctx context.Context, id string) error
	ListTemplates(ctx context.Context, opts ListOptions) ([]domain.Template, error)

	// Deployment operations
	CreateDeployment(ctx context.Context, deployment *domain.Deployment) error
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
	UpdateDeployment(ctx context.Context, deployment *domain.Deployment) error
	DeleteDeployment(ctx context.Context, id string) error
	ListDeployments(ctx context.Context, opts ListOptions) ([]domain.Deployment, error)
	ListDeploymentsByTemplate(ctx context.Context, templateID string, opts ListOptions) ([]domain.Deployment, error)
	ListDeploymentsByCustomer(ctx context.Context, customerID int, opts ListOptions) ([]domain.Deployment, error)

	// Proxy-related deployment operations
	GetDeploymentByDomain(ctx context.Context, hostname string) (*domain.Deployment, error)
	GetUsedProxyPorts(ctx context.Context, nodeID string) ([]int, error)
	CountRoutableDeployments(ctx context.Context) (int, error)

	// Usage event operations (F009: Billing Integration)
	CreateUsageEvent(ctx context.Context, event *domain.MeterEvent) error
	GetUnreportedEvents(ctx context.Context, limit int) ([]domain.MeterEvent, error)
	MarkEventsReported(ctx context.Context, ids []string, reportedAt time.Time) error

	// Container event operations (F010: Monitoring)
	CreateContainerEvent(ctx context.Context, event *domain.ContainerEvent) error
	GetContainerEvents(ctx context.Context, deploymentID string, limit int, eventType *string) ([]domain.ContainerEvent, error)

	// Node operations (Creator Worker Nodes)
	CreateNode(ctx context.Context, node *domain.Node) error
	GetNode(ctx context.Context, id string) (*domain.Node, error)
	GetNodeByCreatorAndName(ctx context.Context, creatorID int, name string) (*domain.Node, error)
	UpdateNode(ctx context.Context, node *domain.Node) error
	DeleteNode(ctx context.Context, id string) error
	ListNodesByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.Node, error)
	ListOnlineNodes(ctx context.Context) ([]domain.Node, error)
	ListCheckableNodes(ctx context.Context) ([]domain.Node, error) // Returns nodes not in maintenance mode

	// SSH Key operations
	CreateSSHKey(ctx context.Context, key *domain.SSHKey) error
	GetSSHKey(ctx context.Context, id string) (*domain.SSHKey, error)
	DeleteSSHKey(ctx context.Context, id string) error
	ListSSHKeysByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.SSHKey, error)

	// Cloud Credential operations
	CreateCloudCredential(ctx context.Context, cred *domain.CloudCredential) error
	GetCloudCredential(ctx context.Context, id string) (*domain.CloudCredential, error)
	DeleteCloudCredential(ctx context.Context, id string) error
	ListCloudCredentialsByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.CloudCredential, error)

	// Cloud Provision operations
	CreateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error
	GetCloudProvision(ctx context.Context, id string) (*domain.CloudProvision, error)
	UpdateCloudProvision(ctx context.Context, prov *domain.CloudProvision) error
	ListCloudProvisionsByCreator(ctx context.Context, creatorID int, opts ListOptions) ([]domain.CloudProvision, error)
	ListActiveProvisions(ctx context.Context) ([]domain.CloudProvision, error)
	ListCloudProvisionsByCredential(ctx context.Context, credentialID int) ([]domain.CloudProvision, error)

	// Dependency lookups (for safe deletion checks)
	ListDeploymentsByNode(ctx context.Context, nodeRefID string) ([]domain.Deployment, error)
	ListNodesBySSHKey(ctx context.Context, sshKeyID int) ([]domain.Node, error)

	// Transaction support
	WithTx(ctx context.Context, fn func(Store) error) error

	// Lifecycle
	Close() error
}

// =============================================================================
// Options
// =============================================================================

// ListOptions defines pagination and filtering options.
type ListOptions struct {
	Limit  int
	Offset int
}

// DefaultListOptions returns default list options.
func DefaultListOptions() ListOptions {
	return ListOptions{
		Limit:  100,
		Offset: 0,
	}
}

// Normalize ensures list options have valid values.
func (o ListOptions) Normalize() ListOptions {
	if o.Limit <= 0 {
		o.Limit = 100
	}
	if o.Limit > 1000 {
		o.Limit = 1000
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
	return o
}
