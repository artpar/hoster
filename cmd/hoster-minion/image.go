package main

import (
	"context"
	"io"
	"strings"

	"github.com/artpar/hoster/internal/core/minion"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// pullImageCmd handles the "pull-image <image>" command.
func pullImageCmd(args []string) error {
	if len(args) < 1 {
		outputError("pull-image", minion.ErrCodeInvalidInput, "usage: pull-image <image>")
		return errInvalidArgs
	}

	ctx := context.Background()
	imageName := args[0]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("pull-image", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	pullOpts := image.PullOptions{}

	// Check for platform option in args
	if len(args) > 1 {
		pullOpts.Platform = args[1]
	}

	reader, err := cli.ImagePull(ctx, imageName, pullOpts)
	if err != nil {
		code := minion.ErrCodePullFailed
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "manifest unknown") {
			code = minion.ErrCodeNotFound
		}
		outputError("pull-image", code, err.Error())
		return err
	}
	defer reader.Close()

	// Drain the reader to complete the pull
	_, _ = io.Copy(io.Discard, reader)

	outputSuccess(nil)
	return nil
}

// imageExistsCmd handles the "image-exists <image>" command.
func imageExistsCmd(args []string) error {
	if len(args) < 1 {
		outputError("image-exists", minion.ErrCodeInvalidInput, "usage: image-exists <image>")
		return errInvalidArgs
	}

	ctx := context.Background()
	imageName := args[0]

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		outputError("image-exists", minion.ErrCodeConnectionFailed, err.Error())
		return err
	}
	defer cli.Close()

	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if strings.Contains(err.Error(), "No such image") {
			outputSuccess(minion.ImageExistsResult{Exists: false})
			return nil
		}
		outputError("image-exists", minion.ErrCodeInternal, err.Error())
		return err
	}

	outputSuccess(minion.ImageExistsResult{Exists: true})
	return nil
}
