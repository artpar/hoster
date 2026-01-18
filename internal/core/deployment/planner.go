package deployment

import "github.com/artpar/hoster/internal/core/domain"

// =============================================================================
// Deployment State Transition Planning
// =============================================================================

// StartPath represents the result of planning a deployment start operation.
// It contains the sequence of state transitions needed to start a deployment.
type StartPath struct {
	// Valid indicates whether the start operation can proceed.
	Valid bool

	// Transitions is the sequence of states to transition through.
	// Empty if Valid is false.
	Transitions []domain.DeploymentStatus

	// ErrorReason contains the reason why the start is not allowed.
	// Empty if Valid is true.
	ErrorReason string
}

// DetermineStartPath determines the sequence of state transitions needed
// to start a deployment from its current status.
//
// This is a pure function that encapsulates the state machine logic for
// starting deployments, following ADR-002 "Values as Boundaries".
//
// Valid start paths:
//   - pending → scheduled → starting
//   - stopped → starting (restart)
//   - failed → starting (retry)
//
// Invalid states for starting:
//   - running: already running
//   - starting/stopping/deleting: operation in progress
//   - deleted: cannot restart deleted deployment
//
// Example:
//
//	path := DetermineStartPath(deployment.Status)
//	if !path.Valid {
//	    return errors.New(path.ErrorReason)
//	}
//	for _, status := range path.Transitions {
//	    deployment.Transition(status)
//	}
func DetermineStartPath(currentStatus domain.DeploymentStatus) StartPath {
	switch currentStatus {
	case domain.StatusPending:
		// First-time start: needs to go through scheduled
		return StartPath{
			Valid:       true,
			Transitions: []domain.DeploymentStatus{domain.StatusScheduled, domain.StatusStarting},
		}

	case domain.StatusStopped, domain.StatusFailed:
		// Restart or retry: direct to starting
		return StartPath{
			Valid:       true,
			Transitions: []domain.DeploymentStatus{domain.StatusStarting},
		}

	case domain.StatusRunning:
		return StartPath{
			Valid:       false,
			ErrorReason: "deployment is already running",
		}

	case domain.StatusStarting:
		return StartPath{
			Valid:       false,
			ErrorReason: "deployment is already starting",
		}

	case domain.StatusStopping:
		return StartPath{
			Valid:       false,
			ErrorReason: "deployment is currently stopping",
		}

	case domain.StatusDeleting:
		return StartPath{
			Valid:       false,
			ErrorReason: "deployment is being deleted",
		}

	case domain.StatusDeleted:
		return StartPath{
			Valid:       false,
			ErrorReason: "cannot start deleted deployment",
		}

	case domain.StatusScheduled:
		return StartPath{
			Valid:       false,
			ErrorReason: "deployment is already scheduled",
		}

	default:
		return StartPath{
			Valid:       false,
			ErrorReason: "cannot start deployment in current state",
		}
	}
}

// CanStopDeployment checks if a deployment can be stopped from its current status.
// Only running deployments can be stopped.
//
// Returns whether the stop is allowed and an optional reason if not.
//
// Example:
//
//	allowed, reason := CanStopDeployment(deployment.Status)
//	if !allowed {
//	    return errors.New(reason)
//	}
func CanStopDeployment(currentStatus domain.DeploymentStatus) (bool, string) {
	if currentStatus != domain.StatusRunning {
		return false, "deployment is not running"
	}
	return true, ""
}
