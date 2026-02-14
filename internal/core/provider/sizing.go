// Package provider contains pure functions for cloud provider logic.
// This is part of the Functional Core - all functions are pure with no I/O.
package provider

// Region represents a cloud provider region.
type Region struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Available   bool   `json:"available"`
}

// InstanceSize represents an instance type/size option.
type InstanceSize struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	CPUCores    float64 `json:"cpu_cores"`
	MemoryMB    int64   `json:"memory_mb"`
	DiskGB      int     `json:"disk_gb"`
	PriceHourly float64 `json:"price_hourly"`
}

// =============================================================================
// AWS EC2 Catalog
// =============================================================================

// AWSRegions returns the commonly used AWS regions.
func AWSRegions() []Region {
	return []Region{
		{ID: "us-east-1", Name: "US East (N. Virginia)", Available: true},
		{ID: "us-east-2", Name: "US East (Ohio)", Available: true},
		{ID: "us-west-1", Name: "US West (N. California)", Available: true},
		{ID: "us-west-2", Name: "US West (Oregon)", Available: true},
		{ID: "eu-west-1", Name: "EU (Ireland)", Available: true},
		{ID: "eu-west-2", Name: "EU (London)", Available: true},
		{ID: "eu-central-1", Name: "EU (Frankfurt)", Available: true},
		{ID: "ap-southeast-1", Name: "Asia Pacific (Singapore)", Available: true},
		{ID: "ap-northeast-1", Name: "Asia Pacific (Tokyo)", Available: true},
	}
}

// AWSSizes returns common EC2 instance types for hosting.
func AWSSizes() []InstanceSize {
	return []InstanceSize{
		{ID: "t3.micro", Name: "t3.micro (1 vCPU, 1 GB)", CPUCores: 1, MemoryMB: 1024, DiskGB: 8, PriceHourly: 0.0104},
		{ID: "t3.small", Name: "t3.small (2 vCPU, 2 GB)", CPUCores: 2, MemoryMB: 2048, DiskGB: 20, PriceHourly: 0.0208},
		{ID: "t3.medium", Name: "t3.medium (2 vCPU, 4 GB)", CPUCores: 2, MemoryMB: 4096, DiskGB: 40, PriceHourly: 0.0416},
		{ID: "t3.large", Name: "t3.large (2 vCPU, 8 GB)", CPUCores: 2, MemoryMB: 8192, DiskGB: 80, PriceHourly: 0.0832},
		{ID: "t3.xlarge", Name: "t3.xlarge (4 vCPU, 16 GB)", CPUCores: 4, MemoryMB: 16384, DiskGB: 160, PriceHourly: 0.1664},
	}
}

// =============================================================================
// DigitalOcean Catalog
// =============================================================================

// DigitalOceanRegions returns common DO regions.
func DigitalOceanRegions() []Region {
	return []Region{
		{ID: "nyc1", Name: "New York 1", Available: true},
		{ID: "nyc3", Name: "New York 3", Available: true},
		{ID: "sfo3", Name: "San Francisco 3", Available: true},
		{ID: "ams3", Name: "Amsterdam 3", Available: true},
		{ID: "lon1", Name: "London 1", Available: true},
		{ID: "fra1", Name: "Frankfurt 1", Available: true},
		{ID: "sgp1", Name: "Singapore 1", Available: true},
		{ID: "blr1", Name: "Bangalore 1", Available: true},
	}
}

// DigitalOceanSizes returns common DO droplet sizes.
func DigitalOceanSizes() []InstanceSize {
	return []InstanceSize{
		{ID: "s-1vcpu-1gb", Name: "Basic (1 vCPU, 1 GB)", CPUCores: 1, MemoryMB: 1024, DiskGB: 25, PriceHourly: 0.00893},
		{ID: "s-1vcpu-2gb", Name: "Basic (1 vCPU, 2 GB)", CPUCores: 1, MemoryMB: 2048, DiskGB: 50, PriceHourly: 0.01786},
		{ID: "s-2vcpu-2gb", Name: "Basic (2 vCPU, 2 GB)", CPUCores: 2, MemoryMB: 2048, DiskGB: 60, PriceHourly: 0.02679},
		{ID: "s-2vcpu-4gb", Name: "Basic (2 vCPU, 4 GB)", CPUCores: 2, MemoryMB: 4096, DiskGB: 80, PriceHourly: 0.03571},
		{ID: "s-4vcpu-8gb", Name: "Basic (4 vCPU, 8 GB)", CPUCores: 4, MemoryMB: 8192, DiskGB: 160, PriceHourly: 0.07143},
	}
}

// =============================================================================
// Hetzner Catalog
// =============================================================================

// HetznerRegions returns common Hetzner Cloud regions.
func HetznerRegions() []Region {
	return []Region{
		{ID: "nbg1", Name: "Nuremberg", Available: true},
		{ID: "fsn1", Name: "Falkenstein", Available: true},
		{ID: "hel1", Name: "Helsinki", Available: true},
		{ID: "ash", Name: "Ashburn, VA", Available: true},
		{ID: "hil", Name: "Hillsboro, OR", Available: true},
	}
}

// HetznerSizes returns common Hetzner server types.
func HetznerSizes() []InstanceSize {
	return []InstanceSize{
		{ID: "cx22", Name: "CX22 (2 vCPU, 4 GB)", CPUCores: 2, MemoryMB: 4096, DiskGB: 40, PriceHourly: 0.0065},
		{ID: "cx32", Name: "CX32 (4 vCPU, 8 GB)", CPUCores: 4, MemoryMB: 8192, DiskGB: 80, PriceHourly: 0.0119},
		{ID: "cx42", Name: "CX42 (8 vCPU, 16 GB)", CPUCores: 8, MemoryMB: 16384, DiskGB: 160, PriceHourly: 0.0229},
		{ID: "cx52", Name: "CX52 (16 vCPU, 32 GB)", CPUCores: 16, MemoryMB: 32768, DiskGB: 320, PriceHourly: 0.0449},
	}
}

// =============================================================================
// Catalog Lookup
// =============================================================================

// StaticRegions returns the static region catalog for a provider.
func StaticRegions(provider string) []Region {
	switch provider {
	case "aws":
		return AWSRegions()
	case "digitalocean":
		return DigitalOceanRegions()
	case "hetzner":
		return HetznerRegions()
	default:
		return nil
	}
}

// StaticSizes returns the static size catalog for a provider.
func StaticSizes(provider string) []InstanceSize {
	switch provider {
	case "aws":
		return AWSSizes()
	case "digitalocean":
		return DigitalOceanSizes()
	case "hetzner":
		return HetznerSizes()
	default:
		return nil
	}
}

// LookupSize returns the InstanceSize for a given provider and size ID, or nil if not found.
func LookupSize(provider, sizeID string) *InstanceSize {
	for _, s := range StaticSizes(provider) {
		if s.ID == sizeID {
			return &s
		}
	}
	return nil
}
