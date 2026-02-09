// Package domain contains the core domain types and validation logic.
// This is part of the Functional Core - all functions are pure with no I/O.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Cloud Provider Errors
// =============================================================================

var (
	ErrCredentialNameRequired = errors.New("credential name is required")
	ErrCredentialNameTooShort = errors.New("credential name must be at least 3 characters")
	ErrCredentialNameTooLong  = errors.New("credential name must be at most 100 characters")
	ErrInvalidProviderType    = errors.New("invalid provider type: must be aws, digitalocean, or hetzner")
	ErrCredentialsRequired    = errors.New("credentials are required")

	ErrProvisionInstanceNameRequired = errors.New("instance name is required")
	ErrProvisionRegionRequired       = errors.New("region is required")
	ErrProvisionSizeRequired         = errors.New("size is required")
	ErrProvisionCredentialRequired   = errors.New("credential ID is required")
	ErrInvalidProvisionTransition    = errors.New("invalid provision status transition")
)

// =============================================================================
// Provider Types
// =============================================================================

// ProviderType represents a cloud infrastructure provider.
type ProviderType string

const (
	ProviderAWS          ProviderType = "aws"
	ProviderDigitalOcean ProviderType = "digitalocean"
	ProviderHetzner      ProviderType = "hetzner"
)

// IsValid checks if the provider type is supported.
func (p ProviderType) IsValid() bool {
	switch p {
	case ProviderAWS, ProviderDigitalOcean, ProviderHetzner:
		return true
	default:
		return false
	}
}

// DisplayName returns a human-readable name for the provider.
func (p ProviderType) DisplayName() string {
	switch p {
	case ProviderAWS:
		return "AWS"
	case ProviderDigitalOcean:
		return "DigitalOcean"
	case ProviderHetzner:
		return "Hetzner"
	default:
		return string(p)
	}
}

// =============================================================================
// Cloud Credential
// =============================================================================

// CloudCredential represents encrypted cloud provider credentials.
type CloudCredential struct {
	ID                   int          `json:"-"`
	ReferenceID          string       `json:"id"`
	CreatorID            int          `json:"-"`
	Name                 string       `json:"name"`
	Provider             ProviderType `json:"provider"`
	CredentialsEncrypted []byte       `json:"-"` // Never serialize
	DefaultRegion        string       `json:"default_region,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// GenerateCredentialID generates a new cloud credential ID.
func GenerateCredentialID() string {
	return "cred_" + uuid.New().String()[:8]
}

// NewCloudCredential creates a new cloud credential with validation.
func NewCloudCredential(creatorID int, name string, provider ProviderType, encryptedCreds []byte, defaultRegion string) (*CloudCredential, error) {
	if err := ValidateCredentialName(name); err != nil {
		return nil, err
	}
	if !provider.IsValid() {
		return nil, ErrInvalidProviderType
	}
	if len(encryptedCreds) == 0 {
		return nil, ErrCredentialsRequired
	}
	if creatorID == 0 {
		return nil, errors.New("creator ID is required")
	}

	now := time.Now()
	return &CloudCredential{
		ReferenceID:          GenerateCredentialID(),
		CreatorID:            creatorID,
		Name:                 name,
		Provider:             provider,
		CredentialsEncrypted: encryptedCreds,
		DefaultRegion:        defaultRegion,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

// =============================================================================
// Provision Status
// =============================================================================

// ProvisionStatus represents the status of a cloud provisioning job.
type ProvisionStatus string

const (
	ProvisionStatusPending     ProvisionStatus = "pending"
	ProvisionStatusCreating    ProvisionStatus = "creating"
	ProvisionStatusConfiguring ProvisionStatus = "configuring"
	ProvisionStatusReady       ProvisionStatus = "ready"
	ProvisionStatusFailed      ProvisionStatus = "failed"
	ProvisionStatusDestroying  ProvisionStatus = "destroying"
	ProvisionStatusDestroyed   ProvisionStatus = "destroyed"
)

// IsValid checks if the provision status is valid.
func (s ProvisionStatus) IsValid() bool {
	switch s {
	case ProvisionStatusPending, ProvisionStatusCreating, ProvisionStatusConfiguring,
		ProvisionStatusReady, ProvisionStatusFailed, ProvisionStatusDestroying, ProvisionStatusDestroyed:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if no further transitions are possible.
func (s ProvisionStatus) IsTerminal() bool {
	return s == ProvisionStatusDestroyed
}

// IsActive returns true if the provisioning is still in progress.
func (s ProvisionStatus) IsActive() bool {
	return s == ProvisionStatusPending || s == ProvisionStatusCreating || s == ProvisionStatusConfiguring
}

// validProvisionTransitions defines the allowed state transitions.
var validProvisionTransitions = map[ProvisionStatus][]ProvisionStatus{
	ProvisionStatusPending:     {ProvisionStatusCreating, ProvisionStatusFailed},
	ProvisionStatusCreating:    {ProvisionStatusConfiguring, ProvisionStatusFailed},
	ProvisionStatusConfiguring: {ProvisionStatusReady, ProvisionStatusFailed},
	ProvisionStatusReady:       {ProvisionStatusDestroying},
	ProvisionStatusFailed:      {ProvisionStatusPending, ProvisionStatusDestroying}, // retry or destroy
	ProvisionStatusDestroying:  {ProvisionStatusDestroyed, ProvisionStatusFailed},
	ProvisionStatusDestroyed:   {}, // terminal
}

// ValidateProvisionTransition checks if a provision status transition is valid.
func ValidateProvisionTransition(from, to ProvisionStatus) error {
	allowed, exists := validProvisionTransitions[from]
	if !exists {
		return ErrInvalidProvisionTransition
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return ErrInvalidProvisionTransition
}

// =============================================================================
// Cloud Provision
// =============================================================================

// CloudProvision represents an asynchronous cloud instance provisioning job.
type CloudProvision struct {
	ID                 int             `json:"-"`
	ReferenceID        string          `json:"id"`
	CreatorID          int             `json:"-"`
	CredentialID       int             `json:"-"`
	CredentialRefID    string          `json:"credential_id"`
	Provider           ProviderType    `json:"provider"`
	Status             ProvisionStatus `json:"status"`
	InstanceName       string          `json:"instance_name"`
	Region             string          `json:"region"`
	Size               string          `json:"size"`
	ProviderInstanceID string          `json:"provider_instance_id,omitempty"`
	PublicIP           string          `json:"public_ip,omitempty"`
	NodeID             string          `json:"node_id,omitempty"`
	SSHKeyID           string          `json:"ssh_key_id,omitempty"`
	CurrentStep        string          `json:"current_step,omitempty"`
	ErrorMessage       string          `json:"error_message,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
}

// GenerateProvisionID generates a new provision ID.
func GenerateProvisionID() string {
	return "prov_" + uuid.New().String()[:8]
}

// NewCloudProvision creates a new cloud provision with validation.
func NewCloudProvision(creatorID, credentialID int, provider ProviderType, instanceName, region, size string) (*CloudProvision, error) {
	if creatorID == 0 {
		return nil, errors.New("creator ID is required")
	}
	if credentialID == 0 {
		return nil, ErrProvisionCredentialRequired
	}
	if !provider.IsValid() {
		return nil, ErrInvalidProviderType
	}
	if instanceName == "" {
		return nil, ErrProvisionInstanceNameRequired
	}
	if region == "" {
		return nil, ErrProvisionRegionRequired
	}
	if size == "" {
		return nil, ErrProvisionSizeRequired
	}

	now := time.Now()
	return &CloudProvision{
		ReferenceID:  GenerateProvisionID(),
		CreatorID:    creatorID,
		CredentialID: credentialID,
		Provider:     provider,
		Status:       ProvisionStatusPending,
		InstanceName: instanceName,
		Region:       region,
		Size:         size,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// Transition attempts to transition the provision to a new status.
func (p *CloudProvision) Transition(to ProvisionStatus) error {
	if err := ValidateProvisionTransition(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	p.UpdatedAt = time.Now()

	if to == ProvisionStatusReady || to == ProvisionStatusDestroyed {
		now := time.Now()
		p.CompletedAt = &now
	}
	if to == ProvisionStatusPending {
		// Retry - clear error
		p.ErrorMessage = ""
		p.CurrentStep = ""
	}
	return nil
}

// TransitionToFailed sets failed status with error message.
func (p *CloudProvision) TransitionToFailed(errorMessage string) error {
	if err := ValidateProvisionTransition(p.Status, ProvisionStatusFailed); err != nil {
		return err
	}
	p.Status = ProvisionStatusFailed
	p.ErrorMessage = errorMessage
	p.UpdatedAt = time.Now()
	return nil
}

// SetStep updates the current step description.
func (p *CloudProvision) SetStep(step string) {
	p.CurrentStep = step
	p.UpdatedAt = time.Now()
}

// =============================================================================
// Validation Functions
// =============================================================================

// ValidateCredentialName validates a cloud credential name.
func ValidateCredentialName(name string) error {
	if name == "" {
		return ErrCredentialNameRequired
	}
	if len(name) < 3 {
		return ErrCredentialNameTooShort
	}
	if len(name) > 100 {
		return ErrCredentialNameTooLong
	}
	return nil
}
