package postgrest

import (
	"fmt"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
)

const ContainerName = "supago-rest"

func Service(postgresPassword string, jwtSecret string) *types.Service {
	return &types.Service{
		Image:   "postgrest/postgrest:v12.2.12",
		Name:    ContainerName,
		Aliases: []string{"rest"},
		Cmd:     []string{"postgrest"},
		Labels: map[string]string{
			"supago.service": ContainerName,
		},
		Env: []string{
			fmt.Sprintf("PGRST_DB_URI=postgres://authenticator:%s@%s:5432/postgres", postgresPassword, postgres.ContainerName),
			"PGRST_DB_SCHEMAS=public",
			"PGRST_DB_ANON_ROLE=anon",
			fmt.Sprintf("PGRST_JWT_SECRET=%s", jwtSecret),
			"PGRST_DB_USE_LEGACY_GUCS=false",
			fmt.Sprintf("PGRST_APP_SETTINGS_JWT_SECRET=%s", jwtSecret),
			"PGRST_APP_SETTINGS_JWT_EXP=3600",
			"PGRST_ADMIN_SERVER_PORT=3001",
		},
	}
}
