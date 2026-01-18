// Package e2e provides end-to-end testing utilities for Hoster.
package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/artpar/hoster/internal/shell/docker"
)

// =============================================================================
// Log Capture
// =============================================================================

// LogCapture streams and stores container logs.
// CRITICAL: Logs are our eyes into what's happening. Without logs, we are blind.
type LogCapture struct {
	mu       sync.Mutex
	logs     map[string]*bytes.Buffer // containerID -> logs
	docker   docker.Client
	testName string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewLogCapture creates a new log capture for a test.
func NewLogCapture(d docker.Client, testName string) *LogCapture {
	return &LogCapture{
		logs:     make(map[string]*bytes.Buffer),
		docker:   d,
		testName: testName,
	}
}

// StartCapturing begins streaming logs for a container.
// This runs in the background until the context is cancelled.
func (lc *LogCapture) StartCapturing(ctx context.Context, containerID string) {
	lc.mu.Lock()
	if _, exists := lc.logs[containerID]; exists {
		lc.mu.Unlock()
		return // Already capturing
	}
	buf := &bytes.Buffer{}
	lc.logs[containerID] = buf
	lc.mu.Unlock()

	lc.wg.Add(1)
	go func() {
		defer lc.wg.Done()

		reader, err := lc.docker.ContainerLogs(containerID, docker.LogOptions{
			Follow:     true,
			Tail:       "all",
			Timestamps: true,
		})
		if err != nil {
			lc.mu.Lock()
			buf.WriteString(fmt.Sprintf("ERROR: Failed to get logs: %v\n", err))
			lc.mu.Unlock()
			return
		}
		defer reader.Close()

		// Read logs until context is cancelled
		readBuf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := reader.Read(readBuf)
			if n > 0 {
				lc.mu.Lock()
				// Docker logs have an 8-byte header per line, skip it for cleaner output
				data := readBuf[:n]
				if len(data) > 8 {
					// Strip Docker log headers (8 bytes per frame)
					buf.Write(stripDockerLogHeaders(data))
				} else {
					buf.Write(data)
				}
				lc.mu.Unlock()
			}
			if err != nil {
				if err == io.EOF {
					return
				}
				return
			}
		}
	}()
}

// stripDockerLogHeaders removes the 8-byte Docker log frame headers.
func stripDockerLogHeaders(data []byte) []byte {
	var result bytes.Buffer
	for len(data) >= 8 {
		// Docker log frame: [stream_type (1), 0, 0, 0, size (4 bytes big-endian)]
		frameSize := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		if frameSize <= 0 || len(data) < 8+frameSize {
			result.Write(data)
			break
		}
		result.Write(data[8 : 8+frameSize])
		data = data[8+frameSize:]
	}
	if len(data) > 0 && len(data) < 8 {
		result.Write(data)
	}
	return result.Bytes()
}

// GetLogs returns captured logs for a container.
func (lc *LogCapture) GetLogs(containerID string) string {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if buf, exists := lc.logs[containerID]; exists {
		return buf.String()
	}
	return ""
}

// DumpAllLogs writes all captured logs to the test output on failure.
func (lc *LogCapture) DumpAllLogs(t *testing.T) {
	if !t.Failed() {
		return // Only dump on failure
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	for containerID, buf := range lc.logs {
		t.Logf("=== LOGS FOR CONTAINER %s ===\n%s\n=== END LOGS ===",
			containerID[:12], buf.String())
	}

	// Also save to disk
	if err := lc.dumpToFile(); err != nil {
		t.Logf("Failed to dump logs to file: %v", err)
	}
}

// DumpToFile saves logs to tests/e2e/logs/{testName}/{containerID}.log.
func (lc *LogCapture) DumpToFile() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.dumpToFile()
}

func (lc *LogCapture) dumpToFile() error {
	if len(lc.logs) == 0 {
		return nil
	}

	// Create logs directory
	logDir := filepath.Join("tests", "e2e", "logs", lc.testName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log dir: %w", err)
	}

	for containerID, buf := range lc.logs {
		logFile := filepath.Join(logDir, containerID[:12]+".log")
		if err := os.WriteFile(logFile, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write log file: %w", err)
		}
	}

	return nil
}

// Stop stops all log capturing.
func (lc *LogCapture) Stop() {
	if lc.cancel != nil {
		lc.cancel()
	}
	lc.wg.Wait()
}

// =============================================================================
// Health Check Waiting
// =============================================================================

// WaitForHealthy polls a container until it becomes healthy or timeout.
// Checks every 5 seconds as per CLAUDE.md requirements.
func WaitForHealthy(ctx context.Context, t *testing.T, d docker.Client, containerID string,
	timeout time.Duration, logCapture *LogCapture) error {

	// Check every 5 seconds as per CLAUDE.md
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	// First check immediately
	info, err := d.InspectContainer(containerID)
	if err != nil {
		return fmt.Errorf("inspect failed: %w", err)
	}
	t.Logf("Container %s initial state: status=%s health=%s", containerID[:12], info.State, info.Health)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := d.InspectContainer(containerID)
			if err != nil {
				return fmt.Errorf("inspect failed: %w", err)
			}

			// Log current state for visibility
			t.Logf("Container %s: status=%s health=%s", containerID[:12], info.State, info.Health)

			switch info.Health {
			case "healthy":
				t.Logf("Container %s is healthy!", containerID[:12])
				return nil
			case "unhealthy":
				// CRITICAL: Dump logs before failing
				logs := logCapture.GetLogs(containerID)
				return fmt.Errorf("container unhealthy after health check, logs:\n%s", logs)
			case "":
				// No health check defined - check if running
				if info.State == "running" {
					t.Logf("Container %s is running (no healthcheck defined)", containerID[:12])
					return nil
				}
			}

			if time.Now().After(deadline) {
				logs := logCapture.GetLogs(containerID)
				return fmt.Errorf("timeout waiting for container to become healthy (current: %s), logs:\n%s",
					info.Health, logs)
			}
		}
	}
}

// WaitForRunning polls a container until it's in running state.
func WaitForRunning(ctx context.Context, t *testing.T, d docker.Client, containerID string,
	timeout time.Duration) error {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			info, err := d.InspectContainer(containerID)
			if err != nil {
				return fmt.Errorf("inspect failed: %w", err)
			}

			t.Logf("Container %s state: %s", containerID[:12], info.State)

			if info.State == "running" {
				return nil
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for container to start (current state: %s)", info.State)
			}
		}
	}
}

// =============================================================================
// Eventually Helper
// =============================================================================

// Eventually retries a condition function until it returns true or timeout.
func Eventually(t *testing.T, timeout, interval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// =============================================================================
// Cleanup Utilities
// =============================================================================

// CleanupDeployment removes all Docker resources for a deployment.
// Order: containers → networks → volumes (networks fail if containers attached).
func CleanupDeployment(ctx context.Context, t *testing.T, d docker.Client,
	deploymentID string, logCapture *LogCapture) error {

	t.Logf("Cleaning up deployment: %s", deploymentID)

	// 1. List containers by label
	containers, err := d.ListContainers(docker.ListOptions{
		All: true,
		Filters: map[string]string{
			"label": docker.LabelDeployment + "=" + deploymentID,
		},
	})
	if err != nil {
		t.Logf("WARN: failed to list containers: %v", err)
		containers = nil
	}

	// 2. Stop and remove containers (capture logs first)
	for _, c := range containers {
		t.Logf("Stopping container: %s (%s)", c.Name, c.ID[:12])

		// Capture final logs
		if logCapture != nil {
			logCapture.StartCapturing(ctx, c.ID)
			time.Sleep(100 * time.Millisecond) // Brief pause to capture logs
		}

		timeout := 10 * time.Second
		if err := d.StopContainer(c.ID, &timeout); err != nil {
			t.Logf("WARN: stop failed for %s: %v", c.ID[:12], err)
		}

		if err := d.RemoveContainer(c.ID, docker.RemoveOptions{Force: true, RemoveVolumes: true}); err != nil {
			t.Logf("WARN: remove failed for %s: %v", c.ID[:12], err)
		}
	}

	// 3. Remove networks with deployment label
	// Note: Network removal is best-effort since they might be shared
	// Docker client doesn't have ListNetworks, so we skip this for now
	// Networks are cleaned up when containers are removed if they're deployment-specific

	t.Logf("Cleanup complete for deployment: %s", deploymentID)
	return nil
}

// CleanupAllTestResources removes all Hoster-managed containers.
// Use this in TestMain cleanup.
func CleanupAllTestResources(ctx context.Context, d docker.Client) error {
	// List all Hoster-managed containers
	containers, err := d.ListContainers(docker.ListOptions{
		All: true,
		Filters: map[string]string{
			"label": docker.LabelManaged + "=true",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Stop and remove all
	for _, c := range containers {
		timeout := 5 * time.Second
		_ = d.StopContainer(c.ID, &timeout)
		_ = d.RemoveContainer(c.ID, docker.RemoveOptions{Force: true, RemoveVolumes: true})
	}

	return nil
}

// =============================================================================
// Fixture Loading
// =============================================================================

// LoadFixture loads a compose fixture from the fixtures directory.
func LoadFixture(t *testing.T, name string) string {
	t.Helper()

	fixturePath := filepath.Join("fixtures", name)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Failed to load fixture %s: %v", name, err)
	}

	return string(content)
}

// =============================================================================
// Container Info Helpers
// =============================================================================

// GetContainersByDeployment returns all containers for a deployment.
func GetContainersByDeployment(ctx context.Context, d docker.Client, deploymentID string) ([]docker.ContainerInfo, error) {
	return d.ListContainers(docker.ListOptions{
		All: true,
		Filters: map[string]string{
			"label": docker.LabelDeployment + "=" + deploymentID,
		},
	})
}
