/*
 * Use of this software is governed by the Business Source License
 * included in the LICENSE file. Production use is permitted, but
 * offering this software as a managed service requires a separate
 * commercial license.
 */

package postgres

import (
	_ "embed"
	"github.com/train360-corp/supago/pkg/types"
)

//go:embed realtime.sql
var RealtimeSQL []byte

//go:embed webhooks.sql
var WebhooksSQL []byte

//go:embed roles.sql
var RolesSQL []byte

//go:embed jwt.sql
var JwtSQL []byte

//go:embed _supabase.sql
var SupabaseSQL []byte

//go:embed logs.sql
var LogsSQL []byte

//go:embed pooler.sql
var PoolerSQL []byte

func GetEmbeddedFiles() []types.EmbeddedFile {
	return []types.EmbeddedFile{
		{
			Name: "realtime.sql",
			Data: RealtimeSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/99-realtime.sql",
		},
		{
			Name: "webhooks.sql",
			Data: WebhooksSQL,
			Path: "/docker-entrypoint-initdb.d/init-scripts/98-webhooks.sql",
		},
		{
			Name: "roles.sql",
			Data: RolesSQL,
			Path: "/docker-entrypoint-initdb.d/init-scripts/99-roles.sql",
		},
		{
			Name: "jwt.sql",
			Data: JwtSQL,
			Path: "/docker-entrypoint-initdb.d/init-scripts/99-jwt.sql",
		},
		{
			Name: "_supabase.sql",
			Data: SupabaseSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/97-_supabase.sql",
		},
		{
			Name: "logs.sql",
			Data: LogsSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/99-logs.sql",
		},
		{
			Name: "pooler.sql",
			Data: PoolerSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/99-pooler.sql",
		},
	}
}
