package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/digitalocean/godo"

	coreprovider "github.com/artpar/hoster/internal/core/provider"
)

// DigitalOceanProvider implements Provider for DigitalOcean.
type DigitalOceanProvider struct {
	client *godo.Client
	logger *slog.Logger
}

// NewDigitalOceanProvider creates a new DigitalOcean provider.
func NewDigitalOceanProvider(apiToken string, logger *slog.Logger) *DigitalOceanProvider {
	return &DigitalOceanProvider{
		client: godo.NewFromToken(apiToken),
		logger: logger.With("provider", "digitalocean"),
	}
}

// CreateInstance provisions a DigitalOcean Droplet.
func (p *DigitalOceanProvider) CreateInstance(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	// Upload SSH key
	key, _, err := p.client.Keys.Create(ctx, &godo.KeyCreateRequest{
		Name:      fmt.Sprintf("hoster-%s", req.InstanceName),
		PublicKey: req.SSHPublicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload SSH key: %w", err)
	}

	// Create droplet
	droplet, _, err := p.client.Droplets.Create(ctx, &godo.DropletCreateRequest{
		Name:   req.InstanceName,
		Region: req.Region,
		Size:   req.Size,
		Image: godo.DropletCreateImage{
			Slug: "docker-20-04", // DigitalOcean Docker marketplace image
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			{ID: key.ID},
		},
		Tags: []string{"hoster", "managed"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create droplet: %w", err)
	}

	p.logger.Info("droplet created", "droplet_id", droplet.ID, "region", req.Region)

	// Wait for active status and public IP
	publicIP, err := p.waitForPublicIP(ctx, droplet.ID)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for public IP: %w", err)
	}

	return &ProvisionResult{
		ProviderInstanceID: fmt.Sprintf("%d", droplet.ID),
		PublicIP:           publicIP,
	}, nil
}

func (p *DigitalOceanProvider) waitForPublicIP(ctx context.Context, dropletID int) (string, error) {
	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}

		droplet, _, err := p.client.Droplets.Get(ctx, dropletID)
		if err != nil {
			continue
		}

		if droplet.Status == "active" {
			ip, err := droplet.PublicIPv4()
			if err == nil && ip != "" {
				return ip, nil
			}
		}
	}
	return "", errors.New("timed out waiting for droplet public IP")
}

// DestroyInstance deletes a DigitalOcean Droplet.
func (p *DigitalOceanProvider) DestroyInstance(ctx context.Context, providerInstanceID string) error {
	var dropletID int
	if _, err := fmt.Sscanf(providerInstanceID, "%d", &dropletID); err != nil {
		return fmt.Errorf("invalid droplet ID: %w", err)
	}

	_, err := p.client.Droplets.Delete(ctx, dropletID)
	if err != nil {
		return fmt.Errorf("failed to delete droplet: %w", err)
	}

	p.logger.Info("droplet deleted", "droplet_id", dropletID)
	return nil
}

// ListRegions returns available DigitalOcean regions.
func (p *DigitalOceanProvider) ListRegions(ctx context.Context) ([]coreprovider.Region, error) {
	doRegions, _, err := p.client.Regions.List(ctx, &godo.ListOptions{PerPage: 100})
	if err != nil {
		return coreprovider.DigitalOceanRegions(), nil
	}

	regions := make([]coreprovider.Region, 0, len(doRegions))
	for _, r := range doRegions {
		regions = append(regions, coreprovider.Region{
			ID:        r.Slug,
			Name:      r.Name,
			Available: r.Available,
		})
	}
	return regions, nil
}

// ListSizes returns available DigitalOcean droplet sizes.
func (p *DigitalOceanProvider) ListSizes(ctx context.Context, region string) ([]coreprovider.InstanceSize, error) {
	doSizes, _, err := p.client.Sizes.List(ctx, &godo.ListOptions{PerPage: 100})
	if err != nil {
		return coreprovider.DigitalOceanSizes(), nil
	}

	sizes := make([]coreprovider.InstanceSize, 0)
	for _, s := range doSizes {
		// Filter by region availability
		available := false
		for _, r := range s.Regions {
			if r == region {
				available = true
				break
			}
		}
		if !available && region != "" {
			continue
		}

		sizes = append(sizes, coreprovider.InstanceSize{
			ID:          s.Slug,
			Name:        fmt.Sprintf("%s (%d vCPU, %d MB)", s.Slug, s.Vcpus, s.Memory),
			CPUCores:    float64(s.Vcpus),
			MemoryMB:    int64(s.Memory),
			DiskGB:      s.Disk,
			PriceHourly: s.PriceHourly,
		})
	}
	return sizes, nil
}
