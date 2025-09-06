package meta

import (
	"fmt"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
)

const ContainerName = "supabase-meta"

func Service(databasePassword string) *types.Service {
	return &types.Service{
		Image:   "supabase/postgres-meta:v0.91.0",
		Name:    ContainerName,
		Aliases: []string{"meta"},
		Cmd:     nil,
		Env: []string{
			fmt.Sprintf("%s=%s", "PG_META_PORT", "8080"),
			fmt.Sprintf("%s=%s", "PG_META_DB_HOST", postgres.ContainerName),
			fmt.Sprintf("%s=%s", "PG_META_DB_PORT", "5432"),
			fmt.Sprintf("%s=%s", "PG_META_DB_NAME", "postgres"),
			fmt.Sprintf("%s=%s", "PG_META_DB_USER", "supabase_admin"),
			fmt.Sprintf("%s=%s", "PG_META_DB_PASSWORD", databasePassword),
		},
		Ports: []uint16{8080},
	}
}
