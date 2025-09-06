package realtime

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
	"github.com/train360-corp/supago/pkg/utils"
	"time"
)

const ContainerName = "supago-realtime"

func Service(postgresPassword string, anonKey string, JwtSecret string) *types.Service {
	return &types.Service{
		Name:  "realtime-dev.supabase-realtime",
		Image: "supabase/realtime:v2.34.47",
		Aliases: []string{
			ContainerName,
			"realtime-dev.supabase-realtime",
			"realtime",
		},
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD",
				"curl",
				"-sSfL",
				"--head",
				"-o",
				"/dev/null",
				"-H",
				fmt.Sprintf("Authorization: Bearer %s", anonKey),
				"http://localhost:4000/api/tenants/realtime-dev/health",
			},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "PORT", "4000"),
			fmt.Sprintf("%s=%s", "DB_HOST", postgres.ContainerName),
			fmt.Sprintf("%s=%s", "DB_PORT", "5432"),
			fmt.Sprintf("%s=%s", "DB_USER", "supabase_admin"),
			fmt.Sprintf("%s=%s", "DB_PASSWORD", postgresPassword),
			fmt.Sprintf("%s=%s", "DB_NAME", "postgres"),
			fmt.Sprintf("%s=%s", "DB_AFTER_CONNECT_QUERY", "SET search_path TO _realtime"),
			fmt.Sprintf("%s=%s", "DB_ENC_KEY", "supabaserealtime"),
			fmt.Sprintf("%s=%s", "API_JWT_SECRET", JwtSecret),
			fmt.Sprintf("%s=%s", "SECRET_KEY_BASE", utils.RandomString(64)),
			fmt.Sprintf("%s=%s", "ERL_AFLAGS", "-proto_dist inet_tcp"),
			fmt.Sprintf("%s=%s", "DNS_NODES", "''"),
			fmt.Sprintf("%s=%s", "RLIMIT_NOFILE", "10000"),
			fmt.Sprintf("%s=%s", "APP_NAME", "realtime"),
			fmt.Sprintf("%s=%s", "SEED_SELF_HOST", "true"),
			fmt.Sprintf("%s=%s", "RUN_JANITOR", "true"),
		},
	}
}
