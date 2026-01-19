package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostnameParser_Parse(t *testing.T) {
	parser := HostnameParser{BaseDomain: "apps.hoster.io"}

	tests := []struct {
		name     string
		hostname string
		wantSlug string
		wantOK   bool
	}{
		{
			name:     "valid simple hostname",
			hostname: "my-blog.apps.hoster.io",
			wantSlug: "my-blog",
			wantOK:   true,
		},
		{
			name:     "valid with port",
			hostname: "my-blog.apps.hoster.io:8080",
			wantSlug: "my-blog",
			wantOK:   true,
		},
		{
			name:     "nested subdomain",
			hostname: "api.my-blog.apps.hoster.io",
			wantSlug: "api.my-blog",
			wantOK:   true,
		},
		{
			name:     "deep nested subdomain",
			hostname: "v1.api.my-blog.apps.hoster.io",
			wantSlug: "v1.api.my-blog",
			wantOK:   true,
		},
		{
			name:     "wrong domain",
			hostname: "my-blog.other.io",
			wantSlug: "",
			wantOK:   false,
		},
		{
			name:     "base domain only",
			hostname: "apps.hoster.io",
			wantSlug: "",
			wantOK:   false,
		},
		{
			name:     "empty hostname",
			hostname: "",
			wantSlug: "",
			wantOK:   false,
		},
		{
			name:     "just port",
			hostname: ":8080",
			wantSlug: "",
			wantOK:   false,
		},
		{
			name:     "partial domain match",
			hostname: "my-blog.notapps.hoster.io",
			wantSlug: "",
			wantOK:   false,
		},
		{
			name:     "slug with numbers",
			hostname: "blog123.apps.hoster.io",
			wantSlug: "blog123",
			wantOK:   true,
		},
		{
			name:     "slug with hyphens",
			hostname: "my-awesome-blog.apps.hoster.io",
			wantSlug: "my-awesome-blog",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, ok := parser.Parse(tt.hostname)
			assert.Equal(t, tt.wantSlug, slug, "slug mismatch")
			assert.Equal(t, tt.wantOK, ok, "ok mismatch")
		})
	}
}

func TestHostnameParser_Parse_DifferentBaseDomains(t *testing.T) {
	tests := []struct {
		name       string
		baseDomain string
		hostname   string
		wantSlug   string
		wantOK     bool
	}{
		{
			name:       "localhost domain",
			baseDomain: "apps.localhost",
			hostname:   "my-app.apps.localhost:9091",
			wantSlug:   "my-app",
			wantOK:     true,
		},
		{
			name:       "custom domain",
			baseDomain: "deployed.example.com",
			hostname:   "shop.deployed.example.com",
			wantSlug:   "shop",
			wantOK:     true,
		},
		{
			name:       "IP-based won't work",
			baseDomain: "192.168.1.1",
			hostname:   "app.192.168.1.1",
			wantSlug:   "app",
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := HostnameParser{BaseDomain: tt.baseDomain}
			slug, ok := parser.Parse(tt.hostname)
			assert.Equal(t, tt.wantSlug, slug)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
