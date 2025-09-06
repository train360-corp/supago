package studio

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/train360-corp/supago/pkg/types"
	"time"
)

const ContainerName = "supago-studio"

type Keys struct {
	Public  string
	Private string
	Secret  string
}

type Database struct {
	Password string
}

type LogFlare struct {
	Password string
}

type Props struct {
	Keys     Keys
	Database Database
	LogFlare LogFlare
}

func Service(props Props) *types.Service {
	return &types.Service{
		Image:   "supabase/studio:2025.06.30-sha-6f5982d",
		Name:    ContainerName,
		Aliases: []string{"studio"},
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD",
				"node",
				"-e",
				"fetch('http://localhost:3000/api/platform/profile').then((r) => {if (r.status !== 200) throw new Error(r.status)})",
			},
			Interval: 5 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		},
		Env: []string{

			"HOSTNAME=0.0.0.0",

			fmt.Sprintf("%s=%s", "STUDIO_PG_META_URL", "http://meta:8080"),
			fmt.Sprintf("%s=%s", "POSTGRES_PASSWORD", props.Database.Password),

			fmt.Sprintf("%s=%s", "DEFAULT_ORGANIZATION_NAME", "Supago"),
			fmt.Sprintf("%s=%s", "DEFAULT_PROJECT_NAME", "Supago"),
			//fmt.Sprintf("%s=%s", "OPENAI_API_KEY", ""),

			fmt.Sprintf("%s=%s", "SUPABASE_URL", "http://kong:8000"),
			fmt.Sprintf("%s=%s", "SUPABASE_PUBLIC_URL", "http://127.0.0.1:8000"),
			fmt.Sprintf("%s=%s", "SUPABASE_ANON_KEY", props.Keys.Public),
			fmt.Sprintf("%s=%s", "SUPABASE_SERVICE_KEY", props.Keys.Private),
			fmt.Sprintf("%s=%s", "AUTH_JWT_SECRET", props.Keys.Secret),

			fmt.Sprintf("%s=%s", "LOGFLARE_PRIVATE_ACCESS_TOKEN", props.LogFlare.Password),
			fmt.Sprintf("%s=%s", "LOGFLARE_URL", "http://analytics:4000"),
			fmt.Sprintf("%s=%s", "NEXT_PUBLIC_ENABLE_LOGS", "true"),
			fmt.Sprintf("%s=%s", "NEXT_ANALYTICS_BACKEND_PROVIDER", "postgres"),
		},
	}
}
