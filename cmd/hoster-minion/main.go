// Package main provides the hoster-minion binary that runs on remote nodes.
//
// The minion provides direct Docker SDK access on the node. The hoster backend
// communicates with the minion via SSH exec, exchanging JSON input/output.
//
// Usage:
//
//	hoster-minion <command> [args...]
//
// Commands:
//
//	version                           - Show minion version
//	ping                              - Test Docker connection
//	create-container                  - Create a container (JSON spec from stdin)
//	start-container <id>              - Start a container
//	stop-container <id> [timeout_ms]  - Stop a container
//	remove-container <id>             - Remove a container (JSON opts from stdin)
//	inspect-container <id>            - Inspect a container
//	list-containers                   - List containers (JSON opts from stdin)
//	container-logs <id>               - Get container logs (JSON opts from stdin)
//	container-stats <id>              - Get container resource stats
//	create-network                    - Create a network (JSON spec from stdin)
//	remove-network <id>               - Remove a network
//	connect-network <net> <container> - Connect container to network
//	disconnect-network <net> <container> [--force] - Disconnect container
//	create-volume                     - Create a volume (JSON spec from stdin)
//	remove-volume <name> [--force]    - Remove a volume
//	pull-image <image>                - Pull an image
//	image-exists <image>              - Check if image exists
package main

import (
	"encoding/json"
	"os"
	"runtime"

	"github.com/artpar/hoster/internal/core/minion"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		outputError("usage", minion.ErrCodeInvalidInput, "usage: hoster-minion <command> [args...]")
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	if err := dispatch(cmd, args); err != nil {
		// Error already written to stdout by command handler
		os.Exit(1)
	}
}

// outputSuccess writes a success response to stdout.
func outputSuccess(data interface{}) {
	resp, err := minion.NewSuccessResponse(data)
	if err != nil {
		outputError("internal", minion.ErrCodeInternal, err.Error())
		return
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

// outputError writes an error response to stdout.
func outputError(command, code, message string) {
	resp := minion.NewErrorResponse(command, code, message)
	json.NewEncoder(os.Stdout).Encode(resp)
}

// versionCmd handles the "version" command.
func versionCmd() error {
	info := minion.VersionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
	}
	outputSuccess(info)
	return nil
}
