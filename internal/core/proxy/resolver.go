package proxy

import "strings"

// HostnameParser extracts deployment info from hostname.
// Pure function - no I/O.
type HostnameParser struct {
	BaseDomain string // e.g., "apps.hoster.io"
}

// Parse extracts the deployment slug from a hostname.
// "my-blog.apps.hoster.io" → "my-blog"
// "my-blog.apps.hoster.io:8080" → "my-blog"
// Returns empty string and false if hostname doesn't match the base domain.
func (p HostnameParser) Parse(hostname string) (slug string, ok bool) {
	if hostname == "" {
		return "", false
	}

	// Strip port if present (find last colon, check if it's followed by digits)
	host := hostname
	if idx := strings.LastIndex(hostname, ":"); idx != -1 {
		// Check if everything after colon looks like a port
		potentialPort := hostname[idx+1:]
		isPort := len(potentialPort) > 0
		for _, c := range potentialPort {
			if c < '0' || c > '9' {
				isPort = false
				break
			}
		}
		if isPort {
			host = hostname[:idx]
		}
	}

	// Check if hostname ends with base domain
	suffix := "." + p.BaseDomain
	if !strings.HasSuffix(host, suffix) {
		return "", false
	}

	// Extract slug (everything before the suffix)
	slug = strings.TrimSuffix(host, suffix)
	if slug == "" {
		return "", false
	}

	return slug, true
}
