package main

import (
	"context"
	"runtime"

	"github.com/artpar/hoster/internal/core/minion"
	"github.com/docker/docker/client"
)

// pingCmd handles the "ping" command.
// It tests the connection to Docker and returns version info.
func pingCmd() error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("ping", minion.ErrCodeConnectionFailed, "failed to create docker client: "+err.Error())
		return err
	}
	defer cli.Close()

	// Get Docker version info
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		outputError("ping", minion.ErrCodeConnectionFailed, "failed to connect to docker: "+err.Error())
		return err
	}

	info := minion.PingInfo{
		DockerVersion: version.Version,
		APIVersion:    version.APIVersion,
		OS:            version.Os,
		Arch:          runtime.GOARCH,
	}
	outputSuccess(info)
	return nil
}
