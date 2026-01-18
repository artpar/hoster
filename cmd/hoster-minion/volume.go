package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/artpar/hoster/internal/core/minion"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// createVolumeCmd handles the "create-volume" command.
// Reads VolumeSpec JSON from stdin.
func createVolumeCmd() error {
	ctx := context.Background()

	// Read spec from stdin
	var spec minion.VolumeSpec
	if err := json.NewDecoder(os.Stdin).Decode(&spec); err != nil {
		outputError("create-volume", minion.ErrCodeInvalidInput, "invalid JSON input: "+err.Error())
		return err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("create-volume", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	driver := spec.Driver
	if driver == "" {
		driver = "local"
	}

	opts := volume.CreateOptions{
		Name:   spec.Name,
		Driver: driver,
		Labels: spec.Labels,
	}

	resp, err := cli.VolumeCreate(ctx, opts)
	if err != nil {
		outputError("create-volume", minion.ErrCodeInternal, err.Error())
		return err
	}

	outputSuccess(minion.VolumeCreateResult{Name: resp.Name})
	return nil
}

// removeVolumeCmd handles the "remove-volume <name> [--force]" command.
func removeVolumeCmd(args []string) error {
	if len(args) < 1 {
		outputError("remove-volume", minion.ErrCodeInvalidInput, "usage: remove-volume <volume_name> [--force]")
		return errInvalidArgs
	}

	ctx := context.Background()
	volumeName := args[0]

	force := false
	if len(args) > 1 && args[1] == "--force" {
		force = true
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("remove-volume", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	if err := cli.VolumeRemove(ctx, volumeName, force); err != nil {
		code := minion.ErrCodeInternal
		if strings.Contains(err.Error(), "not found") {
			code = minion.ErrCodeNotFound
		} else if strings.Contains(err.Error(), "in use") {
			code = minion.ErrCodeInUse
		}
		outputError("remove-volume", code, err.Error())
		return err
	}

	outputSuccess(nil)
	return nil
}
