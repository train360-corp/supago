package types

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/google/uuid"
	"github.com/train360-corp/supago/pkg/utils"
	"os"
	"path/filepath"
	"time"
)

type Service struct {
	Image       string
	Name        string
	Aliases     []string // network aliases
	Cmd         []string
	Env         []string
	Labels      map[string]string
	Mounts      []mount.Mount
	Ports       []uint16
	Healthcheck *container.HealthConfig
	StopSignal  *string
	StopTimeout *time.Duration
	stop        func()
}

func (s Service) Stop() {
	if s.stop != nil {
		s.stop()
	} else {
		utils.Logger().Debugf("%v attempted to stop a service without a stop function", s)
	}
}

func (s Service) String() string {
	return fmt.Sprintf("Service[%s]", s.Name)
}

func (f EmbeddedFile) Mount() (*mount.Mount, error) {

	uniqueFileName := fmt.Sprintf("%s-%s", uuid.New().String(), f.Name)
	localPath := filepath.Join(utils.GetTempDir(), uniqueFileName)

	if err := os.WriteFile(localPath, f.Data, 0o444); err != nil {
		return nil, fmt.Errorf("failed to write temp-file \"%s\": %v", f.Name, err)
	}

	return &mount.Mount{
		Type:     mount.TypeBind,
		Source:   localPath,
		Target:   f.Path,
		ReadOnly: true,
	}, nil
}
