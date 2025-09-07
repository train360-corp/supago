package storage

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
	"os"
	"time"
)

const ContainerName = "supago-storage"

type Keys struct {
	Public  string
	Private string
	Secret  string
}

type Database struct {
	Password string
}

type Storage struct {
	Dir string
}

type Props struct {
	Keys     Keys
	Database Database
	Storage  Storage
}

func Service(props Props) (*types.Service, error) {
	if info, err := os.Stat(props.Storage.Dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(props.Storage.Dir, 0o700); err != nil {
				return nil, fmt.Errorf("storage data directory \"%s\" does not exist and an error occurred while trying to create it: %v", props.Storage.Dir, err)
			}
		} else {
			return nil, fmt.Errorf("error checking storage data directory \"%s\" exists: %v", props.Storage.Dir, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("storage directory \"%s\" exists but is not a directory", props.Storage.Dir)
	}

	return &types.Service{
		Name:    ContainerName,
		Image:   "supabase/storage-api:v1.25.7",
		Aliases: []string{"storage"},
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD",
				"wget",
				"--no-verbose",
				"--tries=1",
				"--spider",
				"http://storage:5000/status",
			},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: props.Storage.Dir,
				Target: "/var/lib/storage",
			},
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "ANON_KEY", props.Keys.Public),
			fmt.Sprintf("%s=%s", "SERVICE_KEY", props.Keys.Private),
			fmt.Sprintf("%s=%s", "POSTGREST_URL", "http://rest:3000"),
			fmt.Sprintf("%s=%s", "PGRST_JWT_SECRET", props.Keys.Secret),
			fmt.Sprintf("%s=%s", "DATABASE_URL",
				fmt.Sprintf("postgres://supabase_storage_admin:%s@%s:5432/postgres", props.Database.Password, postgres.ContainerName)),
			fmt.Sprintf("%s=%s", "FILE_SIZE_LIMIT", "52428800"),
			fmt.Sprintf("%s=%s", "STORAGE_BACKEND", "file"),
			fmt.Sprintf("%s=%s", "FILE_STORAGE_BACKEND_PATH", "/var/lib/storage"),
			fmt.Sprintf("%s=%s", "TENANT_ID", "stub"),
			fmt.Sprintf("%s=%s", "REGION", "stub"),
			fmt.Sprintf("%s=%s", "GLOBAL_S3_BUCKET", "stub"),
			fmt.Sprintf("%s=%s", "ENABLE_IMAGE_TRANSFORMATION", "true"),
			fmt.Sprintf("%s=%s", "IMGPROXY_URL", "http://imgproxy:5001"),
		},
	}, nil
}
