// Package provider implements cloud infrastructure provider clients.
// This is part of the Imperative Shell - handles I/O with cloud APIs.
package provider

import (
	"context"

	coreprovider "github.com/artpar/hoster/internal/core/provider"
)

// ProvisionRequest contains parameters for creating a cloud instance.
type ProvisionRequest struct {
	InstanceName string
	Region       string
	Size         string
	SSHPublicKey string // Public key to install on the instance
}

// ProvisionResult contains the result of creating a cloud instance.
type ProvisionResult struct {
	ProviderInstanceID string
	PublicIP           string
}

// DestroyRequest contains parameters for destroying a cloud instance.
type DestroyRequest struct {
	ProviderInstanceID string
	InstanceName       string // derives SSH key name: "hoster-{InstanceName}"
	Region             string // AWS needs this to target correct region
}

// Provider defines the interface for cloud infrastructure providers.
type Provider interface {
	// CreateInstance provisions a new cloud instance.
	CreateInstance(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)

	// DestroyInstance terminates a cloud instance and cleans up associated resources.
	DestroyInstance(ctx context.Context, req DestroyRequest) error

	// ListRegions returns available regions (live from API).
	ListRegions(ctx context.Context) ([]coreprovider.Region, error)

	// ListSizes returns available instance sizes for a region (live from API).
	ListSizes(ctx context.Context, region string) ([]coreprovider.InstanceSize, error)
}
