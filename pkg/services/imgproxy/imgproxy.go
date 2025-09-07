package imgproxy

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/train360-corp/supago/pkg/types"
	"os"
	"time"
)

const ContainerName = "supago-imgproxy"

func Service(storageDir string) (*types.Service, error) {

	if info, err := os.Stat(storageDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(storageDir, 0o700); err != nil {
				return nil, fmt.Errorf("storage data directory \"%s\" does not exist and an error occurred while trying to create it: %v", storageDir, err)
			}
		} else {
			return nil, fmt.Errorf("error checking storage data directory \"%s\" exists: %v", storageDir, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("storage directory \"%s\" exists but is not a directory", storageDir)
	}

	return &types.Service{
		Name:    ContainerName,
		Image:   "darthsim/imgproxy:v3.8.0",
		Aliases: []string{"imgproxy"},
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD",
				"imgproxy",
				"health",
			},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: storageDir,
				Target: "/var/lib/storage",
			},
		},
		Env: []string{
			"IMGPROXY_BIND=:5001",
			"IMGPROXY_LOCAL_FILESYSTEM_ROOT=/",
			"IMGPROXY_USE_ETAG=true",
			"IMGPROXY_ENABLE_WEBP_DETECTION=true",
		},
	}, nil
}
