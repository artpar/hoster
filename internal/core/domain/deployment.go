package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Deployment Errors
// =============================================================================

var (
	ErrTemplateNotPublished = errors.New("template is not published")
	ErrMissingVariable      = errors.New("required variable is missing")
	ErrInvalidVariable      = errors.New("variable value is invalid")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrNodeRequired         = errors.New("node must be assigned before starting")
)

// =============================================================================
// Deployment Status
// =============================================================================

type DeploymentStatus string

const (
	StatusPending   DeploymentStatus = "pending"
	StatusScheduled DeploymentStatus = "scheduled"
	StatusStarting  DeploymentStatus = "starting"
	StatusRunning   DeploymentStatus = "running"
	StatusStopping  DeploymentStatus = "stopping"
	StatusStopped   DeploymentStatus = "stopped"
	StatusDeleting  DeploymentStatus = "deleting"
	StatusDeleted   DeploymentStatus = "deleted"
	StatusFailed    DeploymentStatus = "failed"
)

// =============================================================================
// Domain Types
// =============================================================================

type DomainType string

const (
	DomainTypeAuto   DomainType = "auto"
	DomainTypeCustom DomainType = "custom"
)

// DomainVerificationStatus represents the verification state of a custom domain.
type DomainVerificationStatus string

const (
	DomainVerificationNone     DomainVerificationStatus = ""         // Auto domains (no verification needed)
	DomainVerificationPending  DomainVerificationStatus = "pending"
	DomainVerificationVerified DomainVerificationStatus = "verified"
	DomainVerificationFailed   DomainVerificationStatus = "failed"
)

// DomainVerificationMethod describes how a domain is verified.
type DomainVerificationMethod string

const (
	DomainVerificationMethodNone  DomainVerificationMethod = ""
	DomainVerificationMethodCNAME DomainVerificationMethod = "cname"
	DomainVerificationMethodA     DomainVerificationMethod = "a_record"
)

// Domain represents a hostname assigned to a deployment.
type Domain struct {
	Hostname           string                   `json:"hostname"`
	Type               DomainType               `json:"type"`
	SSLEnabled         bool                     `json:"ssl_enabled"`
	SSLExpiresAt       *time.Time               `json:"ssl_expires_at,omitempty"`
	VerificationStatus DomainVerificationStatus `json:"verification_status,omitempty"`
	VerificationMethod DomainVerificationMethod `json:"verification_method,omitempty"`
	VerifiedAt         *time.Time               `json:"verified_at,omitempty"`
	LastCheckError     string                   `json:"last_check_error,omitempty"`
}

// =============================================================================
// Container Info
// =============================================================================

// PortMapping represents a port mapping.
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"` // tcp, udp
}

// ContainerInfo represents information about a running container.
type ContainerInfo struct {
	ID          string        `json:"id"`
	ServiceName string        `json:"service_name"`
	Image       string        `json:"image"`
	Status      string        `json:"status"`
	Ports       []PortMapping `json:"ports,omitempty"`
}

// =============================================================================
// Deployment
// =============================================================================

// Deployment represents a running instance of a template.
type Deployment struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	TemplateID      string            `json:"template_id"`
	TemplateVersion string            `json:"template_version"`
	CustomerID      string            `json:"customer_id"`
	NodeID          string            `json:"node_id,omitempty"`
	Status          DeploymentStatus  `json:"status"`
	Variables       map[string]string `json:"variables,omitempty"`
	Domains         []Domain          `json:"domains,omitempty"`
	Containers      []ContainerInfo   `json:"containers,omitempty"`
	Resources       Resources         `json:"resources"`
	ProxyPort       int               `json:"proxy_port,omitempty"` // Host port for App Proxy routing
	ErrorMessage    string            `json:"error_message,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	StoppedAt       *time.Time        `json:"stopped_at,omitempty"`
}

// NewDeployment creates a new deployment from a template.
func NewDeployment(template Template, customerID string, variables map[string]string) (*Deployment, error) {
	// Validate template is published
	if !template.Published {
		return nil, ErrTemplateNotPublished
	}

	// Validate variables
	errs := ValidateDeploymentVariables(template.Variables, variables)
	if len(errs) > 0 {
		return nil, errs[0] // Return first error
	}

	now := time.Now().UTC()
	return &Deployment{
		ID:              uuid.New().String(),
		Name:            GenerateDeploymentName(template.Slug),
		TemplateID:      template.ID,
		TemplateVersion: template.Version,
		CustomerID:      customerID,
		Status:          StatusPending,
		Variables:       variables,
		Resources:       template.ResourceRequirements,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Transition attempts to transition the deployment to a new status.
func (d *Deployment) Transition(to DeploymentStatus) error {
	// First validate the transition is allowed
	if err := ValidateTransition(d.Status, to); err != nil {
		return err
	}

	// Special check for starting - requires node
	if to == StatusStarting && d.NodeID == "" {
		return ErrNodeRequired
	}

	d.Status = to
	d.UpdatedAt = time.Now().UTC()

	// Clear error on retry
	if d.Status == StatusStarting {
		d.ErrorMessage = ""
	}

	// Set timestamps
	if to == StatusRunning {
		now := time.Now().UTC()
		d.StartedAt = &now
	}
	if to == StatusStopped {
		now := time.Now().UTC()
		d.StoppedAt = &now
	}

	return nil
}

// TransitionToFailed transitions to failed status with an error message.
func (d *Deployment) TransitionToFailed(errorMessage string) error {
	// Can fail from starting, running, or stopping
	switch d.Status {
	case StatusStarting, StatusRunning, StatusStopping:
		d.Status = StatusFailed
		d.ErrorMessage = errorMessage
		d.UpdatedAt = time.Now().UTC()
		return nil
	default:
		return ErrInvalidTransition
	}
}

// =============================================================================
// State Machine
// =============================================================================

// validTransitions defines the allowed state transitions.
var validTransitions = map[DeploymentStatus][]DeploymentStatus{
	StatusPending:   {StatusScheduled},
	StatusScheduled: {StatusStarting},
	StatusStarting:  {StatusRunning, StatusFailed},
	StatusRunning:   {StatusStopping, StatusFailed},
	StatusStopping:  {StatusStopped},
	StatusStopped:   {StatusStarting, StatusDeleting},
	StatusDeleting:  {StatusDeleted},
	StatusFailed:    {StatusStarting, StatusDeleting},
	StatusDeleted:   {}, // Terminal state
}

// ValidateTransition checks if a status transition is valid.
func ValidateTransition(from, to DeploymentStatus) error {
	allowed, exists := validTransitions[from]
	if !exists {
		return ErrInvalidTransition
	}

	for _, s := range allowed {
		if s == to {
			return nil
		}
	}

	return ErrInvalidTransition
}

// =============================================================================
// Name Generation
// =============================================================================

// GenerateDeploymentName generates a unique deployment name.
func GenerateDeploymentName(templateSlug string) string {
	suffix := make([]byte, 3)
	rand.Read(suffix)
	return fmt.Sprintf("%s-%s", templateSlug, hex.EncodeToString(suffix))
}

// =============================================================================
// Domain Generation
// =============================================================================

// GenerateDomain generates an auto domain for a deployment.
func GenerateDomain(deploymentName, baseDomain string) Domain {
	return Domain{
		Hostname:   fmt.Sprintf("%s.%s", Slugify(deploymentName), baseDomain),
		Type:       DomainTypeAuto,
		SSLEnabled: false, // Enabled after SSL provisioning
	}
}

// GenerateDomainForNode generates an auto domain using the node's base domain if available,
// falling back to the global base domain.
func GenerateDomainForNode(deploymentName, nodeBaseDomain, globalBaseDomain string) Domain {
	baseDomain := globalBaseDomain
	if nodeBaseDomain != "" {
		baseDomain = nodeBaseDomain
	}
	return GenerateDomain(deploymentName, baseDomain)
}

// NewCustomDomain creates a custom domain entry with pending verification.
func NewCustomDomain(hostname string) Domain {
	return Domain{
		Hostname:           hostname,
		Type:               DomainTypeCustom,
		SSLEnabled:         false,
		VerificationStatus: DomainVerificationPending,
		VerificationMethod: DomainVerificationMethodCNAME,
	}
}

// =============================================================================
// Variable Validation
// =============================================================================

// ValidateDeploymentVariables validates that all required variables are provided.
func ValidateDeploymentVariables(templateVars []Variable, providedVars map[string]string) []error {
	var errs []error

	for _, v := range templateVars {
		if v.Required {
			if _, exists := providedVars[v.Name]; !exists {
				errs = append(errs, fmt.Errorf("%w: %s", ErrMissingVariable, v.Name))
			}
		}
	}

	return errs
}
