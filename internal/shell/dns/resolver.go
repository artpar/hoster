// Package dns provides DNS resolution for domain verification.
// This is part of the Imperative Shell - handles I/O (DNS lookups).
package dns

import (
	"context"
	"net"

	coredns "github.com/artpar/hoster/internal/core/dns"
)

// Resolver performs DNS lookups for domain verification.
type Resolver struct {
	resolver *net.Resolver
}

// NewResolver creates a new DNS resolver.
func NewResolver() *Resolver {
	return &Resolver{
		resolver: net.DefaultResolver,
	}
}

// Resolve performs DNS lookups for the given hostname and returns a VerificationInput
// that can be passed to the pure verification function.
func (r *Resolver) Resolve(ctx context.Context, hostname string) coredns.VerificationInput {
	input := coredns.VerificationInput{
		Hostname: hostname,
	}

	// Look up CNAME records
	cname, err := r.resolver.LookupCNAME(ctx, hostname)
	if err == nil && cname != "" {
		input.CNAMERecords = []string{cname}
	}

	// Look up A records
	ips, err := r.resolver.LookupIPAddr(ctx, hostname)
	if err == nil {
		for _, ip := range ips {
			input.ARecords = append(input.ARecords, ip.IP)
		}
	}

	// If both lookups failed, record the error
	if len(input.CNAMERecords) == 0 && len(input.ARecords) == 0 {
		input.LookupError = "no DNS records found for " + hostname
	}

	return input
}
