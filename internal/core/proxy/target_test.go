package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyTarget_CanRoute(t *testing.T) {
	tests := []struct {
		name   string
		target ProxyTarget
		want   bool
	}{
		{
			name:   "running with port",
			target: ProxyTarget{Status: "running", Port: 30001},
			want:   true,
		},
		{
			name:   "stopped with port",
			target: ProxyTarget{Status: "stopped", Port: 30001},
			want:   false,
		},
		{
			name:   "running no port",
			target: ProxyTarget{Status: "running", Port: 0},
			want:   false,
		},
		{
			name:   "pending with port",
			target: ProxyTarget{Status: "pending", Port: 30001},
			want:   false,
		},
		{
			name:   "starting with port",
			target: ProxyTarget{Status: "starting", Port: 30001},
			want:   false,
		},
		{
			name:   "failed with port",
			target: ProxyTarget{Status: "failed", Port: 30001},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.target.CanRoute())
		})
	}
}

func TestProxyTarget_IsLocal(t *testing.T) {
	tests := []struct {
		name   string
		nodeID string
		want   bool
	}{
		{"empty node ID", "", true},
		{"local", "local", true},
		{"remote node", "node_123", false},
		{"another remote", "remote-worker", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := ProxyTarget{NodeID: tt.nodeID}
			assert.Equal(t, tt.want, target.IsLocal())
		})
	}
}

func TestProxyTarget_LocalAddress(t *testing.T) {
	tests := []struct {
		name string
		port int
		want string
	}{
		{"port 30001", 30001, "127.0.0.1:30001"},
		{"port 30999", 30999, "127.0.0.1:30999"},
		{"port 80", 80, "127.0.0.1:80"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := ProxyTarget{Port: tt.port}
			assert.Equal(t, tt.want, target.LocalAddress())
		})
	}
}

func TestProxyTarget_RemoteAddress(t *testing.T) {
	tests := []struct {
		name   string
		nodeIP string
		port   int
		want   string
	}{
		{"standard remote", "24.199.126.77", 30001, "24.199.126.77:30001"},
		{"different port", "10.0.0.5", 8080, "10.0.0.5:8080"},
		{"hostname", "worker.example.com", 30002, "worker.example.com:30002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := ProxyTarget{NodeIP: tt.nodeIP, Port: tt.port}
			assert.Equal(t, tt.want, target.RemoteAddress())
		})
	}
}
