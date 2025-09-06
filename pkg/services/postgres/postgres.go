package postgres

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	postgres "github.com/train360-corp/supago/pkg/services/postgres/embeds"
	"github.com/train360-corp/supago/pkg/types"
	"github.com/train360-corp/supago/pkg/utils"
	"os"
	"path/filepath"
	"time"
)

const ContainerName = "supago-db"

// Service create the postgres service object
func Service(dataDir string, password string, jwtSecret string) (*types.Service, error) {

	if info, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dataDir, 0o700); err != nil {
				return nil, fmt.Errorf("postgres data directory \"%s\" does not exist and an error occurred while trying to create it: %v", dataDir, err)
			}
		} else {
			return nil, fmt.Errorf("error checking postgres data directory \"%s\" exists: %v", dataDir, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("postgres data directory \"%s\" exists but is not a directory", dataDir)
	}

	dbDataDir := filepath.Join(dataDir, "postgres")
	if info, err := os.Stat(dbDataDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dbDataDir, 0o700); err != nil {
				return nil, fmt.Errorf("postgres data directory \"%s\" does not exist and an error occurred while trying to create it: %v", dbDataDir, err)
			}
		} else {
			return nil, fmt.Errorf("error checking postgres data directory \"%s\" exists: %v", dbDataDir, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("postgres data directory \"%s\" exists but is not a directory", dbDataDir)
	}

	files := postgres.GetEmbeddedFiles()
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "db-config",
			Target: "/etc/postgresql-custom",
		},
		{
			Type:   mount.TypeBind,
			Source: dbDataDir,
			Target: "/var/lib/postgresql/data",
		},
	}
	for _, file := range files {
		mnt, err := file.Mount()
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, *mnt)
	}

	return &types.Service{
		Image:       "supabase/postgres:17.4.1.055",
		Name:        ContainerName,
		Mounts:      mounts,
		Ports:       make([]uint16, 0),
		StopTimeout: utils.Pointer(10 * time.Second),
		Aliases:     []string{"db"},
		Labels: map[string]string{
			"projconf.service": "postgres",
		},
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD", "pg_isready", "-U", "postgres", "-h", "localhost"},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  10,
		},
		Cmd: []string{
			"postgres",
			"-c", "config_file=/etc/postgresql/postgresql.conf",
			"-c", "log_min_messages=error",
			"-c", "wal_level=minimal",
			"-c", "max_wal_senders=0",
		},
		Env: []string{
			"POSTGRES_HOST=/var/run/postgresql",
			"PGPORT=5432",
			"POSTGRES_PORT=5432",
			fmt.Sprintf("PGPASSWORD=%s", password),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
			"PGDATABASE=postgres",
			"POSTGRES_DB=postgres",
			fmt.Sprintf("JWT_SECRET=%s", jwtSecret),
			"JWT_EXP=3600",
		},
	}, nil
}
