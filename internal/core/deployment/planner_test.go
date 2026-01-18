package deployment

import (
	"testing"

	"github.com/artpar/hoster/internal/core/domain"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// DetermineStartPath Tests
// =============================================================================

func TestDetermineStartPath_FromPending(t *testing.T) {
	path := DetermineStartPath(domain.StatusPending)

	assert.True(t, path.Valid)
	assert.Empty(t, path.ErrorReason)
	assert.Len(t, path.Transitions, 2)
	assert.Equal(t, domain.StatusScheduled, path.Transitions[0])
	assert.Equal(t, domain.StatusStarting, path.Transitions[1])
}

func TestDetermineStartPath_FromStopped(t *testing.T) {
	path := DetermineStartPath(domain.StatusStopped)

	assert.True(t, path.Valid)
	assert.Empty(t, path.ErrorReason)
	assert.Len(t, path.Transitions, 1)
	assert.Equal(t, domain.StatusStarting, path.Transitions[0])
}

func TestDetermineStartPath_FromFailed(t *testing.T) {
	path := DetermineStartPath(domain.StatusFailed)

	assert.True(t, path.Valid)
	assert.Empty(t, path.ErrorReason)
	assert.Len(t, path.Transitions, 1)
	assert.Equal(t, domain.StatusStarting, path.Transitions[0])
}

func TestDetermineStartPath_AlreadyRunning(t *testing.T) {
	path := DetermineStartPath(domain.StatusRunning)

	assert.False(t, path.Valid)
	assert.Equal(t, "deployment is already running", path.ErrorReason)
	assert.Empty(t, path.Transitions)
}

func TestDetermineStartPath_AlreadyStarting(t *testing.T) {
	path := DetermineStartPath(domain.StatusStarting)

	assert.False(t, path.Valid)
	assert.Equal(t, "deployment is already starting", path.ErrorReason)
	assert.Empty(t, path.Transitions)
}

func TestDetermineStartPath_CurrentlyStopping(t *testing.T) {
	path := DetermineStartPath(domain.StatusStopping)

	assert.False(t, path.Valid)
	assert.Equal(t, "deployment is currently stopping", path.ErrorReason)
	assert.Empty(t, path.Transitions)
}

func TestDetermineStartPath_BeingDeleted(t *testing.T) {
	path := DetermineStartPath(domain.StatusDeleting)

	assert.False(t, path.Valid)
	assert.Equal(t, "deployment is being deleted", path.ErrorReason)
	assert.Empty(t, path.Transitions)
}

func TestDetermineStartPath_AlreadyDeleted(t *testing.T) {
	path := DetermineStartPath(domain.StatusDeleted)

	assert.False(t, path.Valid)
	assert.Equal(t, "cannot start deleted deployment", path.ErrorReason)
	assert.Empty(t, path.Transitions)
}

func TestDetermineStartPath_AlreadyScheduled(t *testing.T) {
	path := DetermineStartPath(domain.StatusScheduled)

	assert.False(t, path.Valid)
	assert.Equal(t, "deployment is already scheduled", path.ErrorReason)
	assert.Empty(t, path.Transitions)
}

// =============================================================================
// CanStopDeployment Tests
// =============================================================================

func TestCanStopDeployment_WhenRunning(t *testing.T) {
	allowed, reason := CanStopDeployment(domain.StatusRunning)

	assert.True(t, allowed)
	assert.Empty(t, reason)
}

func TestCanStopDeployment_WhenNotRunning(t *testing.T) {
	statuses := []domain.DeploymentStatus{
		domain.StatusPending,
		domain.StatusScheduled,
		domain.StatusStarting,
		domain.StatusStopping,
		domain.StatusStopped,
		domain.StatusFailed,
		domain.StatusDeleting,
		domain.StatusDeleted,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			allowed, reason := CanStopDeployment(status)

			assert.False(t, allowed)
			assert.Equal(t, "deployment is not running", reason)
		})
	}
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestDetermineStartPath_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		status         domain.DeploymentStatus
		wantValid      bool
		wantTransCount int
		wantError      string
	}{
		{
			name:           "pending starts with two transitions",
			status:         domain.StatusPending,
			wantValid:      true,
			wantTransCount: 2,
			wantError:      "",
		},
		{
			name:           "stopped restarts with one transition",
			status:         domain.StatusStopped,
			wantValid:      true,
			wantTransCount: 1,
			wantError:      "",
		},
		{
			name:           "failed retries with one transition",
			status:         domain.StatusFailed,
			wantValid:      true,
			wantTransCount: 1,
			wantError:      "",
		},
		{
			name:           "running cannot start",
			status:         domain.StatusRunning,
			wantValid:      false,
			wantTransCount: 0,
			wantError:      "deployment is already running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := DetermineStartPath(tt.status)

			assert.Equal(t, tt.wantValid, path.Valid)
			assert.Len(t, path.Transitions, tt.wantTransCount)
			assert.Equal(t, tt.wantError, path.ErrorReason)
		})
	}
}
