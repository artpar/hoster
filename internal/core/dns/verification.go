// Package dns contains pure functions for DNS verification logic.
// This is part of the Functional Core - all functions are pure with no I/O.
package dns

import (
	"errors"
	"net"
	"regexp"
	"strings"

	"github.com/artpar/hoster/internal/core/domain"
)

// =============================================================================
// Errors
// =============================================================================

var (
	ErrInvalidHostname     = errors.New("invalid hostname format")
	ErrHostnameTooLong     = errors.New("hostname must be under 253 characters")
	ErrDomainAlreadyExists = errors.New("custom domain already exists on this deployment")
	ErrMaxDomainsReached   = errors.New("maximum number of custom domains reached")
	ErrCannotRemoveAuto    = errors.New("cannot remove auto-generated domain")
)

const MaxCustomDomains = 5

// =============================================================================
// Validation
// =============================================================================

var hostnameRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// ValidateCustomDomain validates a hostname format for use as a custom domain.
func ValidateCustomDomain(hostname string) error {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	if hostname == "" {
		return ErrInvalidHostname
	}
	if len(hostname) > 253 {
		return ErrHostnameTooLong
	}
	if !hostnameRegex.MatchString(hostname) {
		return ErrInvalidHostname
	}
	return nil
}

// CanAddCustomDomain checks if a new custom domain can be added to existing domains.
func CanAddCustomDomain(existing []domain.Domain, hostname string) error {
	customCount := 0
	for _, d := range existing {
		if d.Type == domain.DomainTypeCustom {
			customCount++
			if strings.EqualFold(d.Hostname, hostname) {
				return ErrDomainAlreadyExists
			}
		}
	}
	if customCount >= MaxCustomDomains {
		return ErrMaxDomainsReached
	}
	return nil
}

// =============================================================================
// Verification
// =============================================================================

// VerificationInput contains DNS lookup results passed from the shell layer.
type VerificationInput struct {
	Hostname     string
	CNAMERecords []string
	ARecords     []net.IP
	LookupError  string
}

// VerificationResult is the pure output of verification logic.
type VerificationResult struct {
	Verified bool
	Method   domain.DomainVerificationMethod
	Error    string
}

// Verify performs pure verification logic on DNS lookup results.
// It checks if the DNS records point to the expected target.
func Verify(input VerificationInput, expectedAutoDomain string, expectedIPs []string) VerificationResult {
	if input.LookupError != "" {
		return VerificationResult{
			Verified: false,
			Error:    "DNS lookup failed: " + input.LookupError,
		}
	}

	// Check CNAME records first (preferred method)
	for _, cname := range input.CNAMERecords {
		// Remove trailing dot from CNAME
		cname = strings.TrimSuffix(cname, ".")
		if strings.EqualFold(cname, expectedAutoDomain) {
			return VerificationResult{
				Verified: true,
				Method:   domain.DomainVerificationMethodCNAME,
			}
		}
	}

	// Check A records
	for _, aRecord := range input.ARecords {
		for _, expectedIP := range expectedIPs {
			if aRecord.String() == expectedIP {
				return VerificationResult{
					Verified: true,
					Method:   domain.DomainVerificationMethodA,
				}
			}
		}
	}

	return VerificationResult{
		Verified: false,
		Error:    "DNS records do not point to the expected target",
	}
}

// =============================================================================
// DNS Instructions
// =============================================================================

// DNSInstruction represents a DNS record the user needs to create.
type DNSInstruction struct {
	Type     string `json:"type"`     // "CNAME" or "A"
	Name     string `json:"name"`     // The hostname to set
	Value    string `json:"value"`    // The target (auto domain or IP)
	Priority string `json:"priority"` // "recommended" or "alternative"
}

// GenerateInstructions returns DNS setup instructions for a custom domain.
func GenerateInstructions(customDomain, autoDomain, nodeIP string) []DNSInstruction {
	instructions := []DNSInstruction{
		{
			Type:     "CNAME",
			Name:     customDomain,
			Value:    autoDomain,
			Priority: "recommended",
		},
	}

	if nodeIP != "" {
		instructions = append(instructions, DNSInstruction{
			Type:     "A",
			Name:     customDomain,
			Value:    nodeIP,
			Priority: "alternative",
		})
	}

	return instructions
}

// FindAutoDomain returns the first auto-generated domain from a domain list.
func FindAutoDomain(domains []domain.Domain) string {
	for _, d := range domains {
		if d.Type == domain.DomainTypeAuto {
			return d.Hostname
		}
	}
	return ""
}
