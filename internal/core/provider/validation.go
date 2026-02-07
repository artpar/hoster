package provider

import (
	"encoding/json"
	"errors"
)

// =============================================================================
// Credential Validation (Pure - no I/O)
// =============================================================================

var (
	ErrAWSAccessKeyRequired    = errors.New("AWS access key ID is required")
	ErrAWSSecretKeyRequired    = errors.New("AWS secret access key is required")
	ErrDOTokenRequired         = errors.New("DigitalOcean API token is required")
	ErrHetznerTokenRequired    = errors.New("Hetzner API token is required")
	ErrUnknownProvider         = errors.New("unknown provider type")
)

// AWSCredentials represents AWS access credentials.
type AWSCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

// DigitalOceanCredentials represents DigitalOcean API credentials.
type DigitalOceanCredentials struct {
	APIToken string `json:"api_token"`
}

// HetznerCredentials represents Hetzner Cloud API credentials.
type HetznerCredentials struct {
	APIToken string `json:"api_token"`
}

// ValidateAWSCredentials validates AWS credential fields.
func ValidateAWSCredentials(creds AWSCredentials) error {
	if creds.AccessKeyID == "" {
		return ErrAWSAccessKeyRequired
	}
	if creds.SecretAccessKey == "" {
		return ErrAWSSecretKeyRequired
	}
	return nil
}

// ValidateDigitalOceanCredentials validates DigitalOcean credential fields.
func ValidateDigitalOceanCredentials(creds DigitalOceanCredentials) error {
	if creds.APIToken == "" {
		return ErrDOTokenRequired
	}
	return nil
}

// ValidateHetznerCredentials validates Hetzner credential fields.
func ValidateHetznerCredentials(creds HetznerCredentials) error {
	if creds.APIToken == "" {
		return ErrHetznerTokenRequired
	}
	return nil
}

// ValidateCredentialsJSON validates credential JSON for a given provider.
func ValidateCredentialsJSON(provider string, credJSON []byte) error {
	switch provider {
	case "aws":
		var creds AWSCredentials
		if err := json.Unmarshal(credJSON, &creds); err != nil {
			return errors.New("invalid AWS credentials JSON")
		}
		return ValidateAWSCredentials(creds)
	case "digitalocean":
		var creds DigitalOceanCredentials
		if err := json.Unmarshal(credJSON, &creds); err != nil {
			return errors.New("invalid DigitalOcean credentials JSON")
		}
		return ValidateDigitalOceanCredentials(creds)
	case "hetzner":
		var creds HetznerCredentials
		if err := json.Unmarshal(credJSON, &creds); err != nil {
			return errors.New("invalid Hetzner credentials JSON")
		}
		return ValidateHetznerCredentials(creds)
	default:
		return ErrUnknownProvider
	}
}

// ParseAWSCredentials parses AWS credentials from JSON.
func ParseAWSCredentials(data []byte) (AWSCredentials, error) {
	var creds AWSCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return creds, err
	}
	return creds, ValidateAWSCredentials(creds)
}

// ParseDigitalOceanCredentials parses DigitalOcean credentials from JSON.
func ParseDigitalOceanCredentials(data []byte) (DigitalOceanCredentials, error) {
	var creds DigitalOceanCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return creds, err
	}
	return creds, ValidateDigitalOceanCredentials(creds)
}

// ParseHetznerCredentials parses Hetzner credentials from JSON.
func ParseHetznerCredentials(data []byte) (HetznerCredentials, error) {
	var creds HetznerCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return creds, err
	}
	return creds, ValidateHetznerCredentials(creds)
}
