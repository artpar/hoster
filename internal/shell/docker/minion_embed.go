package docker

import (
	"embed"
	"fmt"
	"runtime"
)

// Embedded minion binaries for Linux platforms.
// These are compiled separately and embedded at build time.
//
// To build the minion binaries:
//   make build-minion
//
// This will create:
//   internal/shell/docker/binaries/minion-linux-amd64
//   internal/shell/docker/binaries/minion-linux-arm64

//go:embed binaries/*
var minionBinaries embed.FS

// GetMinionBinary returns the minion binary for the specified OS and architecture.
// Currently only Linux amd64 and arm64 are supported.
func GetMinionBinary(goos, goarch string) ([]byte, error) {
	if goos != "linux" {
		return nil, fmt.Errorf("unsupported OS: %s (only linux is supported)", goos)
	}

	var filename string
	switch goarch {
	case "amd64":
		filename = "binaries/minion-linux-amd64"
	case "arm64":
		filename = "binaries/minion-linux-arm64"
	default:
		return nil, fmt.Errorf("unsupported architecture: %s (only amd64 and arm64 are supported)", goarch)
	}

	data, err := minionBinaries.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("minion binary not found for %s/%s: %w (run 'make build-minion' first)", goos, goarch, err)
	}

	return data, nil
}

// GetMinionBinaryForCurrentPlatform returns the minion binary for the current platform.
// Useful for testing on the local machine.
func GetMinionBinaryForCurrentPlatform() ([]byte, error) {
	return GetMinionBinary(runtime.GOOS, runtime.GOARCH)
}

// MinionVersion is the version of the embedded minion binaries.
// This should match the version in cmd/hoster-minion/main.go.
var MinionVersion = "1.1.0"
