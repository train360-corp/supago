package supago

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"time"
)

type Service struct {
	Image       string
	Name        string
	Aliases     []string
	Entrypoint  []string
	Cmd         []string
	Env         []string
	Labels      map[string]string
	Mounts      []mount.Mount
	Ports       []uint16
	Healthcheck *container.HealthConfig
	StopSignal  *string
	StopTimeout *time.Duration
	AfterStart  func(ctx context.Context, docker *client.Client, containerID string) error
	container   *container.CreateResponse
	closeConn   func()
}

func (s Service) String() string {
	return fmt.Sprintf("Service[%s]", s.Name)
}
