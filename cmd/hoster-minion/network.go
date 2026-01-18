package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/artpar/hoster/internal/core/minion"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// createNetworkCmd handles the "create-network" command.
// Reads NetworkSpec JSON from stdin.
func createNetworkCmd() error {
	ctx := context.Background()

	// Read spec from stdin
	var spec minion.NetworkSpec
	if err := json.NewDecoder(os.Stdin).Decode(&spec); err != nil {
		outputError("create-network", minion.ErrCodeInvalidInput, "invalid JSON input: "+err.Error())
		return err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("create-network", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	driver := spec.Driver
	if driver == "" {
		driver = "bridge"
	}

	opts := network.CreateOptions{
		Driver: driver,
		Labels: spec.Labels,
	}

	resp, err := cli.NetworkCreate(ctx, spec.Name, opts)
	if err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "already exists") {
			code = minion.ErrCodeAlreadyExists
		}
		outputError("create-network", code, err.Error())
		return err
	}

	outputSuccess(minion.CreateResult{ID: resp.ID})
	return nil
}

// removeNetworkCmd handles the "remove-network <id>" command.
func removeNetworkCmd(args []string) error {
	if len(args) < 1 {
		outputError("remove-network", minion.ErrCodeInvalidInput, "usage: remove-network <network_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	networkID := args[0]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("remove-network", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	if err := cli.NetworkRemove(ctx, networkID); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = minion.ErrCodeNotFound
		} else if strings.Contains(err.Error(), "has active endpoints") {
			code = minion.ErrCodeInUse
		}
		outputError("remove-network", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}

// connectNetworkCmd handles the "connect-network <network_id> <container_id>" command.
func connectNetworkCmd(args []string) error {
	if len(args) < 2 {
		outputError("connect-network", minion.ErrCodeInvalidInput, "usage: connect-network <network_id> <container_id>")
		return errInvalidArgs
	}

	ctx := context.Background()
	networkID := args[0]
	containerID := args[1]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("connect-network", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	if err := cli.NetworkConnect(ctx, networkID, containerID, nil); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = minion.ErrCodeNotFound
		}
		outputError("connect-network", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}

// disconnectNetworkCmd handles the "disconnect-network <network_id> <container_id> [--force]" command.
func disconnectNetworkCmd(args []string) error {
	if len(args) < 2 {
		outputError("disconnect-network", minion.ErrCodeInvalidInput, "usage: disconnect-network <network_id> <container_id> [--force]")
		return errInvalidArgs
	}

	ctx := context.Background()
	networkID := args[0]
	containerID := args[1]

	force := false
	if len(args) > 2 && args[2] == "--force" {
		force = true
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("disconnect-network", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	if err := cli.NetworkDisconnect(ctx, networkID, containerID, force); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = minion.ErrCodeNotFound
		}
		outputError("disconnect-network", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}
