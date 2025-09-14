/*
 * Use of this software is governed by the Business Source License
 * included in the LICENSE file. Production use is permitted, but
 * offering this software as a managed service requires a separate
 * commercial license.
 */

package postgres

import (
	_ "embed"
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
