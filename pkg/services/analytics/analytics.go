package analytics

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
	"time"
)

const ContainerName = "supago-analytics"

func Service(postgresPassword string, publicKey string, privateKey string) *types.Service {
	return &types.Service{
		Image:   "supabase/logflare:1.14.2",
		Name:    ContainerName,
		Aliases: []string{"analytics"},
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD",
				"curl",
				"http://localhost:4000/health",
			},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  10,
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "LOGFLARE_NODE_HOST", "127.0.0.1"),
			fmt.Sprintf("%s=%s", "DB_USERNAME", "supabase_admin"),
			fmt.Sprintf("%s=%s", "DB_DATABASE", "_supabase"),
			fmt.Sprintf("%s=%s", "DB_HOSTNAME", postgres.ContainerName),
			fmt.Sprintf("%s=%s", "DB_PORT", "5432"),
			fmt.Sprintf("%s=%s", "DB_PASSWORD", postgresPassword),
			fmt.Sprintf("%s=%s", "DB_SCHEMA", "_analytics"),
			fmt.Sprintf("%s=%s", "LOGFLARE_PUBLIC_ACCESS_TOKEN", publicKey),
			fmt.Sprintf("%s=%s", "LOGFLARE_PRIVATE_ACCESS_TOKEN", privateKey),
			fmt.Sprintf("%s=%s", "LOGFLARE_SINGLE_TENANT", "true"),
			fmt.Sprintf("%s=%s", "LOGFLARE_SUPABASE_MODE", "true"),
			fmt.Sprintf("%s=%s", "LOGFLARE_MIN_CLUSTER_SIZE", "1"),
			fmt.Sprintf("%s=%s", "POSTGRES_BACKEND_URL",
				fmt.Sprintf("postgresql://supabase_admin:%s@%s:5432/_supabase", postgresPassword, postgres.ContainerName)),
			fmt.Sprintf("%s=%s", "POSTGRES_BACKEND_SCHEMA", "_analytics"),
			fmt.Sprintf("%s=%s", "LOGFLARE_FEATURE_FLAG_OVERRIDE", "multibackend=true"),
		},
	}
}
