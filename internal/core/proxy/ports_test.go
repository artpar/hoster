package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPortRange(t *testing.T) {
	pr := DefaultPortRange()
	assert.Equal(t, 30000, pr.Start)
	assert.Equal(t, 39999, pr.End)
}

func TestAllocatePort(t *testing.T) {
	tests := []struct {
		name      string
		usedPorts []int
		portRange PortRange
		wantPort  int
		wantErr   bool
	}{
		{
			name:      "empty used ports returns first port",
			usedPorts: nil,
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  30000,
			wantErr:   false,
		},
		{
			name:      "first port used returns second",
			usedPorts: []int{30000},
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  30001,
			wantErr:   false,
		},
		{
			name:      "some ports used returns first available",
			usedPorts: []int{30000, 30001},
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  30002,
			wantErr:   false,
		},
		{
			name:      "all ports used returns error",
			usedPorts: []int{30000, 30001, 30002, 30003, 30004, 30005},
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  0,
			wantErr:   true,
		},
		{
			name:      "gaps in used ports fills first gap",
			usedPorts: []int{30000, 30002},
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  30001,
			wantErr:   false,
		},
		{
			name:      "large gaps finds first available",
			usedPorts: []int{30000, 30005},
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  30001,
			wantErr:   false,
		},
		{
			name:      "unsorted used ports works correctly",
			usedPorts: []int{30002, 30000, 30001},
			portRange: PortRange{Start: 30000, End: 30005},
			wantPort:  30003,
			wantErr:   false,
		},
		{
			name:      "single port range all used",
			usedPorts: []int{30000},
			portRange: PortRange{Start: 30000, End: 30000},
			wantPort:  0,
			wantErr:   true,
		},
		{
			name:      "single port range empty",
			usedPorts: nil,
			portRange: PortRange{Start: 30000, End: 30000},
			wantPort:  30000,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := AllocatePort(tt.usedPorts, tt.portRange)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, 0, port)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPort, port)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	portRange := PortRange{Start: 30000, End: 39999}

	tests := []struct {
		name  string
		port  int
		valid bool
	}{
		{"start of range", 30000, true},
		{"end of range", 39999, true},
		{"middle of range", 35000, true},
		{"below range", 29999, false},
		{"above range", 40000, false},
		{"zero", 0, false},
		{"negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, ValidatePort(tt.port, portRange))
		})
	}
}
