package deployment

import (
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// ConvertPorts Tests
// =============================================================================

func TestConvertPorts_Empty(t *testing.T) {
	result := ConvertPorts(nil)
	assert.Empty(t, result)
	assert.NotNil(t, result) // Should be empty slice, not nil
}

func TestConvertPorts_EmptySlice(t *testing.T) {
	result := ConvertPorts([]PortBinding{})
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

func TestConvertPorts_SinglePort(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 80, HostPort: 8080, Protocol: "tcp"},
	}
	result := ConvertPorts(ports)

	assert.Len(t, result, 1)
	assert.Equal(t, 80, result[0].ContainerPort)
	assert.Equal(t, 8080, result[0].HostPort)
	assert.Equal(t, "tcp", result[0].Protocol)
}

func TestConvertPorts_DefaultProtocol(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 80, HostPort: 8080, Protocol: ""},
	}
	result := ConvertPorts(ports)

	assert.Equal(t, "tcp", result[0].Protocol)
}

func TestConvertPorts_UDP(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 53, HostPort: 53, Protocol: "udp"},
	}
	result := ConvertPorts(ports)

	assert.Equal(t, "udp", result[0].Protocol)
}

func TestConvertPorts_Multiple(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 80, HostPort: 80, Protocol: "tcp"},
		{ContainerPort: 443, HostPort: 443, Protocol: "tcp"},
		{ContainerPort: 53, HostPort: 53, Protocol: "udp"},
	}
	result := ConvertPorts(ports)

	assert.Len(t, result, 3)
	assert.Equal(t, 80, result[0].ContainerPort)
	assert.Equal(t, 443, result[1].ContainerPort)
	assert.Equal(t, 53, result[2].ContainerPort)
	assert.Equal(t, "udp", result[2].Protocol)
}

func TestConvertPorts_HighPorts(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 3000, HostPort: 3000, Protocol: "tcp"},
		{ContainerPort: 8080, HostPort: 8080, Protocol: "tcp"},
	}
	result := ConvertPorts(ports)

	assert.Len(t, result, 2)
	assert.Equal(t, 3000, result[0].ContainerPort)
	assert.Equal(t, 8080, result[1].ContainerPort)
}

func TestConvertPorts_DifferentHostAndContainerPorts(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 80, HostPort: 8080, Protocol: "tcp"},
		{ContainerPort: 3000, HostPort: 80, Protocol: "tcp"},
	}
	result := ConvertPorts(ports)

	assert.Len(t, result, 2)
	assert.Equal(t, 80, result[0].ContainerPort)
	assert.Equal(t, 8080, result[0].HostPort)
	assert.Equal(t, 3000, result[1].ContainerPort)
	assert.Equal(t, 80, result[1].HostPort)
}

func TestConvertPorts_ZeroPorts(t *testing.T) {
	ports := []PortBinding{
		{ContainerPort: 0, HostPort: 0, Protocol: "tcp"},
	}
	result := ConvertPorts(ports)

	// Should pass through zeros (validation is at caller level)
	assert.Len(t, result, 1)
	assert.Equal(t, 0, result[0].ContainerPort)
	assert.Equal(t, 0, result[0].HostPort)
}

func TestConvertPorts_HostIPIgnored(t *testing.T) {
	// HostIP is in PortBinding but not in domain.PortMapping
	ports := []PortBinding{
		{ContainerPort: 80, HostPort: 80, Protocol: "tcp", HostIP: "127.0.0.1"},
	}
	result := ConvertPorts(ports)

	// HostIP is not part of domain.PortMapping, but conversion should still work
	assert.Len(t, result, 1)
	assert.Equal(t, 80, result[0].ContainerPort)
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestConvertPorts_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		ports []PortBinding
		want  []domain.PortMapping
	}{
		{
			name:  "nil input",
			ports: nil,
			want:  []domain.PortMapping{},
		},
		{
			name:  "empty slice",
			ports: []PortBinding{},
			want:  []domain.PortMapping{},
		},
		{
			name:  "single tcp port",
			ports: []PortBinding{{ContainerPort: 80, HostPort: 80, Protocol: "tcp"}},
			want:  []domain.PortMapping{{ContainerPort: 80, HostPort: 80, Protocol: "tcp"}},
		},
		{
			name:  "default protocol",
			ports: []PortBinding{{ContainerPort: 80, HostPort: 80, Protocol: ""}},
			want:  []domain.PortMapping{{ContainerPort: 80, HostPort: 80, Protocol: "tcp"}},
		},
		{
			name:  "udp port",
			ports: []PortBinding{{ContainerPort: 53, HostPort: 53, Protocol: "udp"}},
			want:  []domain.PortMapping{{ContainerPort: 53, HostPort: 53, Protocol: "udp"}},
		},
		{
			name: "mixed protocols",
			ports: []PortBinding{
				{ContainerPort: 80, HostPort: 80, Protocol: "tcp"},
				{ContainerPort: 53, HostPort: 53, Protocol: "udp"},
			},
			want: []domain.PortMapping{
				{ContainerPort: 80, HostPort: 80, Protocol: "tcp"},
				{ContainerPort: 53, HostPort: 53, Protocol: "udp"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertPorts(tt.ports)
			assert.Equal(t, tt.want, got)
		})
	}
}
