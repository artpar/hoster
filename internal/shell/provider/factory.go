package provider

import (
	"fmt"
	"log/slog"

	coreprovider "github.com/artpar/hoster/internal/core/provider"
)

// NewProvider creates a cloud provider client from decrypted credentials JSON.
func NewProvider(providerType string, credJSON []byte, logger *slog.Logger) (Provider, error) {
	switch providerType {
	case "aws":
		creds, err := coreprovider.ParseAWSCredentials(credJSON)
		if err != nil {
			return nil, fmt.Errorf("invalid AWS credentials: %w", err)
		}
		return NewAWSProvider(creds.AccessKeyID, creds.SecretAccessKey, logger), nil

	case "digitalocean":
		creds, err := coreprovider.ParseDigitalOceanCredentials(credJSON)
		if err != nil {
			return nil, fmt.Errorf("invalid DigitalOcean credentials: %w", err)
		}
		return NewDigitalOceanProvider(creds.APIToken, logger), nil

	case "hetzner":
		creds, err := coreprovider.ParseHetznerCredentials(credJSON)
		if err != nil {
			return nil, fmt.Errorf("invalid Hetzner credentials: %w", err)
		}
		return NewHetznerProvider(creds.APIToken, logger), nil

	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
