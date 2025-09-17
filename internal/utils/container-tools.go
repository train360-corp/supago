package utils

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"path"
	"strings"
	"time"
)

// CopyToContainer copies a single file's contents into a docker container at file.Path.
// It creates any missing parent directories with mode 0755 and writes the file as 0644.
// Ownership will be the container default (usually root:root).
func CopyToContainer(
	ctx context.Context,
	docker *client.Client,
	cid string,
	file struct {
		Data []byte
		Path string // absolute path inside container, e.g. "/etc/postgresql-custom/postgresql.custom.conf"
	},
) error {
	if file.Path == "" || !strings.HasPrefix(file.Path, "/") {
		return fmt.Errorf("container path must be absolute, got %q", file.Path)
	}

	// BuildE a tar that places the file at its absolute path by extracting under "/"
	rel := strings.TrimPrefix(file.Path, "/")
	if rel == "" || rel == "." || strings.Contains(rel, "..") {
		return fmt.Errorf("invalid container path %q", file.Path)
	}
	parent := path.Dir(rel) // e.g. "etc/postgresql-custom"
	base := path.Base(rel)  // e.g. "postgresql.custom.conf"

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	now := time.Now()

	// Emit mkdir -p style directory headers (0755)
	if parent != "." && parent != "/" {
		parts := strings.Split(parent, "/")
		cur := ""
		for _, seg := range parts {
			if seg == "" {
				continue
			}
			if cur == "" {
				cur = seg
			} else {
				cur = cur + "/" + seg
			}
			hdr := &tar.Header{
				Name:     cur + "/", // relative dir entry
				Typeflag: tar.TypeDir,
				Mode:     0o755, // drwxr-xr-x
				ModTime:  now,
			}
			if err := tw.WriteHeader(hdr); err != nil {
				_ = tw.Close()
				return fmt.Errorf("write dir header %q: %w", cur, err)
			}
		}
	}

	// File header at full relative path (parent/base) with 0644
	fhdr := &tar.Header{
		Name:     path.Join(parent, base),
		Typeflag: tar.TypeReg,
		Mode:     0o644, // -rw-r--r--
		Size:     int64(len(file.Data)),
		ModTime:  now,
	}
	if err := tw.WriteHeader(fhdr); err != nil {
		_ = tw.Close()
		return fmt.Errorf("write file header %q: %w", file.Path, err)
	}
	if _, err := tw.Write(file.Data); err != nil {
		_ = tw.Close()
		return fmt.Errorf("write file data %q: %w", file.Path, err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}

	// Extract under "/" so the tar's relative paths land at absolute locations
	return docker.CopyToContainer(
		ctx,
		cid,
		"/",
		bytes.NewReader(buf.Bytes()),
		container.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
			CopyUIDGID:                false, // do NOT preserve uid/gid from headers; use container defaults
		},
	)
}

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
