package main

import (
	"github.com/artpar/hoster/internal/core/minion"
)

// dispatch routes the command to the appropriate handler.
func dispatch(cmd string, args []string) error {
	switch cmd {
	// Health commands
	case "version":
		return versionCmd()
	case "ping":
		return pingCmd()

	// Container commands
	case "create-container":
		return createContainerCmd()
	case "start-container":
		return startContainerCmd(args)
	case "stop-container":
		return stopContainerCmd(args)
	case "remove-container":
		return removeContainerCmd(args)
	case "inspect-container":
		return inspectContainerCmd(args)
	case "list-containers":
		return listContainersCmd()
	case "container-logs":
		return containerLogsCmd(args)
	case "container-stats":
		return containerStatsCmd(args)

	// Network commands
	case "create-network":
		return createNetworkCmd()
	case "remove-network":
		return removeNetworkCmd(args)
	case "connect-network":
		return connectNetworkCmd(args)
	case "disconnect-network":
		return disconnectNetworkCmd(args)

	// Volume commands
	case "create-volume":
		return createVolumeCmd()
	case "remove-volume":
		return removeVolumeCmd(args)

	// Image commands
	case "pull-image":
		return pullImageCmd(args)
	case "image-exists":
		return imageExistsCmd(args)

	default:
		outputError(cmd, minion.ErrCodeInvalidInput, "unknown command: "+cmd)
		return errUnknownCommand
	}
}

// errUnknownCommand is returned for unknown commands.
var errUnknownCommand = &commandError{msg: "unknown command"}

// commandError represents a command error.
type commandError struct {
	msg string
}

func (e *commandError) Error() string {
	return e.msg
}
