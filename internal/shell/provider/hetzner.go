package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	coreprovider "github.com/artpar/hoster/internal/core/provider"
)

// HetznerProvider implements Provider for Hetzner Cloud.
type HetznerProvider struct {
	client *hcloud.Client
	logger *slog.Logger
}

// NewHetznerProvider creates a new Hetzner Cloud provider.
func NewHetznerProvider(apiToken string, logger *slog.Logger) *HetznerProvider {
	return &HetznerProvider{
		client: hcloud.NewClient(hcloud.WithToken(apiToken)),
		logger: logger.With("provider", "hetzner"),
	}
}

// CreateInstance provisions a Hetzner Cloud server.
func (p *HetznerProvider) CreateInstance(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	// Upload SSH key (idempotent: delete existing key first if present)
	keyName := fmt.Sprintf("hoster-%s", req.InstanceName)
	if existing, _, _ := p.client.SSHKey.GetByName(ctx, keyName); existing != nil {
		p.client.SSHKey.Delete(ctx, existing)
	}
	key, _, err := p.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      keyName,
		PublicKey: req.SSHPublicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload SSH key: %w", err)
	}

	// Get server type
	serverType, _, err := p.client.ServerType.GetByName(ctx, req.Size)
	if err != nil || serverType == nil {
		return nil, fmt.Errorf("invalid server type %s: %w", req.Size, err)
	}

	// Get location
	location, _, err := p.client.Location.GetByName(ctx, req.Region)
	if err != nil || location == nil {
		return nil, fmt.Errorf("invalid location %s: %w", req.Region, err)
	}

	// Get Ubuntu image
	image, _, err := p.client.Image.GetByNameAndArchitecture(ctx, "ubuntu-22.04", hcloud.ArchitectureX86)
	if err != nil || image == nil {
		return nil, fmt.Errorf("failed to find Ubuntu image: %w", err)
	}

	// Create server with Docker user data
	result, _, err := p.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       req.InstanceName,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		SSHKeys:    []*hcloud.SSHKey{key},
		UserData:   dockerInstallScript(),
		Labels: map[string]string{
			"managed-by": "hoster",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	p.logger.Info("Hetzner server created", "server_id", result.Server.ID, "location", req.Region)

	// Wait for running state
	publicIP, err := p.waitForPublicIP(ctx, result.Server.ID)
	if err != nil {
		return nil, fmt.Errorf("failed waiting for public IP: %w", err)
	}

	return &ProvisionResult{
		ProviderInstanceID: strconv.FormatInt(result.Server.ID, 10),
		PublicIP:           publicIP,
	}, nil
}

func (p *HetznerProvider) waitForPublicIP(ctx context.Context, serverID int64) (string, error) {
	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
		}

		server, _, err := p.client.Server.GetByID(ctx, serverID)
		if err != nil || server == nil {
			continue
		}

		if server.Status == hcloud.ServerStatusRunning && !server.PublicNet.IPv4.IP.IsUnspecified() {
			return server.PublicNet.IPv4.IP.String(), nil
		}
	}
	return "", errors.New("timed out waiting for server public IP")
}

// DestroyInstance deletes a Hetzner Cloud server and cleans up SSH key.
func (p *HetznerProvider) DestroyInstance(ctx context.Context, req DestroyRequest) error {
	serverID, err := strconv.ParseInt(req.ProviderInstanceID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid server ID: %w", err)
	}

	server, _, err := p.client.Server.GetByID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}
	if server == nil {
		p.logger.Info("Hetzner server already deleted", "server_id", serverID)
	} else {
		_, _, err = p.client.Server.DeleteWithResult(ctx, server)
		if err != nil {
			return fmt.Errorf("failed to delete server: %w", err)
		}
		p.logger.Info("Hetzner server deleted", "server_id", serverID)
	}

	// Best-effort cleanup of SSH key
	keyName := fmt.Sprintf("hoster-%s", req.InstanceName)
	if existing, _, _ := p.client.SSHKey.GetByName(ctx, keyName); existing != nil {
		if _, err := p.client.SSHKey.Delete(ctx, existing); err != nil {
			p.logger.Warn("failed to delete SSH key during destroy", "key_name", keyName, "error", err)
		}
	}

	return nil
}

// ListRegions returns available Hetzner locations.
func (p *HetznerProvider) ListRegions(ctx context.Context) ([]coreprovider.Region, error) {
	locations, _, err := p.client.Location.List(ctx, hcloud.LocationListOpts{})
	if err != nil {
		return coreprovider.HetznerRegions(), nil
	}

	regions := make([]coreprovider.Region, 0, len(locations))
	for _, loc := range locations {
		regions = append(regions, coreprovider.Region{
			ID:        loc.Name,
			Name:      fmt.Sprintf("%s (%s)", loc.City, loc.Country),
			Available: true,
		})
	}
	return regions, nil
}

// ListSizes returns available Hetzner server types.
func (p *HetznerProvider) ListSizes(ctx context.Context, region string) ([]coreprovider.InstanceSize, error) {
	serverTypes, _, err := p.client.ServerType.List(ctx, hcloud.ServerTypeListOpts{})
	if err != nil {
		return coreprovider.HetznerSizes(), nil
	}

	sizes := make([]coreprovider.InstanceSize, 0)
	for _, st := range serverTypes {
		// Filter shared CPU types only (cx/cax lines) for cost efficiency
		if st.CPUType != hcloud.CPUTypeShared {
			continue
		}

		price := 0.0
		for _, p := range st.Pricings {
			if p.Location.Name == region || region == "" {
				hourly, _ := strconv.ParseFloat(p.Hourly.Gross, 64)
				price = hourly
				break
			}
		}

		sizes = append(sizes, coreprovider.InstanceSize{
			ID:          st.Name,
			Name:        fmt.Sprintf("%s (%d vCPU, %.0f GB)", st.Name, st.Cores, st.Memory),
			CPUCores:    float64(st.Cores),
			MemoryMB:    int64(st.Memory * 1024),
			DiskGB:      st.Disk,
			PriceHourly: price,
		})
	}
	return sizes, nil
}

// dockerInstallScript returns a cloud-init script for installing Docker.
func dockerInstallScript() string {
	return `#!/bin/bash
set -e
apt-get update -y
apt-get install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable docker
systemctl start docker
`
}
