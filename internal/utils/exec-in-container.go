package utils

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ExecInContainer runs "cmd" inside container cid and streams output to stdout/stderr
func ExecInContainer(ctx context.Context, docker *client.Client, cid string, cmd []string) (string, error) {
	// create ExecInContainer instance
	execResp, err := docker.ContainerExecCreate(ctx, cid, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	})
	if err != nil {
		return "", fmt.Errorf("ExecInContainer create failed: %w", err)
	}

	// attach
	att, err := docker.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: false})
	if err != nil {
		return "", fmt.Errorf("ExecInContainer attach failed: %w", err)
	}
	defer att.Close()

	// copy output to local stdout/stderr
	var buf bytes.Buffer
	_, _ = stdcopy.StdCopy(&buf, &buf, att.Reader)
	output := buf.String()

	// check exit code
	inspect, err := docker.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return output, fmt.Errorf("ExecInContainer inspect failed: %w", err)
	}
	if inspect.ExitCode != 0 {
		return output, fmt.Errorf("ExecInContainer command exited with code %d", inspect.ExitCode)
	}

	return output, nil
}
