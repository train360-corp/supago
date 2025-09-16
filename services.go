package supago

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/train360-corp/supago/internal/services/kong"
	postgres "github.com/train360-corp/supago/internal/services/postgres/embeds"
	"github.com/train360-corp/supago/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AllServices func(Config) *[]*Service

type IServices interface {
	Analytics(Config) *Service
	Auth(Config) *Service
	ImgProxy(Config) *Service
	Kong(Config) *Service
	Meta(Config) *Service
	Postgres(Config) *Service
	Postgrest(Config) *Service
	Realtime(Config) *Service
	Storage(Config) *Service
	Studio(Config) *Service
	All() AllServices
}

type TServices struct{}

var Services IServices = TServices{}

const dbContainerName = "supago-db"

func containerName(config Config, name string) string {
	if !IsValidPlatformName(config.Global.PlatformName) {
		return name
	}
	return fmt.Sprintf("%s-%s", config.Global.PlatformName, name)
}

func (T TServices) Analytics(config Config) *Service {
	return &Service{
		Image:   "supabase/logflare:1.14.2",
		Name:    containerName(config, "supago-analytics"),
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
			fmt.Sprintf("%s=%s", "DB_HOSTNAME", containerName(config, dbContainerName)),
			fmt.Sprintf("%s=%s", "DB_PORT", "5432"),
			fmt.Sprintf("%s=%s", "DB_PASSWORD", config.Database.Password),
			fmt.Sprintf("%s=%s", "DB_SCHEMA", "_analytics"),
			fmt.Sprintf("%s=%s", "LOGFLARE_PUBLIC_ACCESS_TOKEN", config.LogFlare.PublicKey),
			fmt.Sprintf("%s=%s", "LOGFLARE_PRIVATE_ACCESS_TOKEN", config.LogFlare.PrivateKey),
			fmt.Sprintf("%s=%s", "LOGFLARE_SINGLE_TENANT", "true"),
			fmt.Sprintf("%s=%s", "LOGFLARE_SUPABASE_MODE", "true"),
			fmt.Sprintf("%s=%s", "LOGFLARE_MIN_CLUSTER_SIZE", "1"),
			fmt.Sprintf("%s=%s", "POSTGRES_BACKEND_URL",
				fmt.Sprintf("postgresql://supabase_admin:%s@%s:5432/_supabase", config.Database.Password, containerName(config, dbContainerName))),
			fmt.Sprintf("%s=%s", "POSTGRES_BACKEND_SCHEMA", "_analytics"),
			fmt.Sprintf("%s=%s", "LOGFLARE_FEATURE_FLAG_OVERRIDE", "multibackend=true"),
		},
	}
}

func (T TServices) Auth(config Config) *Service {
	return &Service{
		Name:    containerName(config, "supago-auth"),
		Aliases: []string{"auth", "gotrue"},
		Image:   "supabase/gotrue:v2.177.0",
		Healthcheck: &container.HealthConfig{
			Test: []string{
				"CMD",
				"wget",
				"--no-verbose",
				"--tries=1",
				"--spider",
				"http://localhost:9999/health",
			},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "GOTRUE_API_HOST", "0.0.0.0"),
			fmt.Sprintf("%s=%s", "GOTRUE_API_PORT", "9999"),
			fmt.Sprintf("%s=%s", "API_EXTERNAL_URL", config.Kong.URLs.Kong),

			fmt.Sprintf("%s=%s", "GOTRUE_DB_DRIVER", "postgres"),
			fmt.Sprintf("%s=%s", "GOTRUE_DB_DATABASE_URL",
				fmt.Sprintf("postgres://supabase_auth_admin:%s@%s:5432/postgres", config.Database.Password, containerName(config, dbContainerName))),

			fmt.Sprintf("%s=%s", "GOTRUE_SITE_URL", config.Kong.URLs.Site),
			fmt.Sprintf("%s=%s", "GOTRUE_URI_ALLOW_LIST", ""),
			fmt.Sprintf("%s=%s", "GOTRUE_DISABLE_SIGNUP", "false"),

			fmt.Sprintf("%s=%s", "GOTRUE_JWT_ADMIN_ROLES", "service_role"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_AUD", "authenticated"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_DEFAULT_GROUP_NAME", "authenticated"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_EXP", "3600"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_SECRET", config.Keys.JwtSecret),

			fmt.Sprintf("%s=%s", "GOTRUE_EXTERNAL_EMAIL_ENABLED", "true"),
			fmt.Sprintf("%s=%s", "GOTRUE_EXTERNAL_ANONYMOUS_USERS_ENABLED", "false"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_AUTOCONFIRM", "false"),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_ADMIN_EMAIL", config.Kong.SMTP.From.Email),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_HOST", config.Kong.SMTP.Host),
			fmt.Sprintf("%s=%d", "GOTRUE_SMTP_PORT", config.Kong.SMTP.Port),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_USER", config.Kong.SMTP.User),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_PASS", config.Kong.SMTP.Pass),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_SENDER_NAME", config.Kong.SMTP.From.Name),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_INVITE", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_CONFIRMATION", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_RECOVERY", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_EMAIL_CHANGE", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_EXTERNAL_PHONE_ENABLED", "false"),
			fmt.Sprintf("%s=%s", "GOTRUE_SMS_AUTOCONFIRM", "false"),
		},
	}
}

func (T TServices) ImgProxy(config Config) *Service {
	if info, err := os.Stat(config.Storage.DataDirectory); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(config.Storage.DataDirectory, 0o700); err != nil {
				panic(fmt.Sprintf("storage data directory \"%s\" does not exist and an error occurred while trying to create it: %v", config.Storage.DataDirectory, err))
			}
		} else {
			panic(fmt.Sprintf("error checking storage data directory \"%s\" exists: %v", config.Storage.DataDirectory, err))
		}
	} else if !info.IsDir() {
		panic(fmt.Sprintf("storage directory \"%s\" exists but is not a directory", config.Storage.DataDirectory))
	}

	return &Service{
		Name:    containerName(config, "supago-imgproxy"),
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
				Source: config.Storage.DataDirectory,
				Target: "/var/lib/storage",
			},
		},
		Env: []string{
			"IMGPROXY_BIND=:5001",
			"IMGPROXY_LOCAL_FILESYSTEM_ROOT=/",
			"IMGPROXY_USE_ETAG=true",
			"IMGPROXY_ENABLE_WEBP_DETECTION=true",
		},
	}
}

func (T TServices) Kong(config Config) *Service {
	cfg := EmbeddedFile{
		Name: "kong.yml",
		Data: kong.ConfigFile,
		Path: "/usr/local/kong-template.yml",
	}

	var mounts []mount.Mount
	if mnt, err := cfg.Mount(); err != nil {
		panic(err)
	} else {
		mounts = append(mounts, *mnt)
	}

	return &Service{
		Image:   "kong:2.8.1",
		Name:    containerName(config, kong.ContainerName),
		Aliases: []string{"kong"},
		Ports: []uint16{
			8000,
		},
		Mounts: mounts,
		Entrypoint: []string{
			"bash", "-c",
			`set -euo pipefail
eval "echo \"$(cat /usr/local/kong-template.yml)\"" > "$HOME/kong.yml"
exec /docker-entrypoint.sh kong docker-start`,
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "KONG_DATABASE", "off"),
			fmt.Sprintf("%s=%s", "KONG_DECLARATIVE_CONFIG", "/home/kong/kong.yml"),
			fmt.Sprintf("%s=%s", "KONG_DNS_ORDER", "LAST,A,CNAME"),
			fmt.Sprintf("%s=%s", "KONG_PLUGINS", "request-transformer,cors,key-auth,acl,basic-auth"),
			fmt.Sprintf("%s=%s", "KONG_NGINX_PROXY_PROXY_BUFFER_SIZE", "160k"),
			fmt.Sprintf("%s=%s", "KONG_NGINX_PROXY_PROXY_BUFFERS", "64 160k"),
			fmt.Sprintf("%s=%s", "SUPABASE_ANON_KEY", config.Keys.PublicJwt),
			fmt.Sprintf("%s=%s", "SUPABASE_SERVICE_KEY", config.Keys.PrivateJwt),
			fmt.Sprintf("%s=%s", "DASHBOARD_USERNAME", config.Dashboard.Username),
			fmt.Sprintf("%s=%s", "DASHBOARD_PASSWORD", config.Dashboard.Password),
		},
	}
}

func (T TServices) Meta(config Config) *Service {
	return &Service{
		Image:   "supabase/postgres-meta:v0.91.0",
		Name:    containerName(config, "supago-meta"),
		Aliases: []string{"meta"},
		Env: []string{
			fmt.Sprintf("%s=%s", "PG_META_PORT", "8080"),
			fmt.Sprintf("%s=%s", "PG_META_DB_HOST", containerName(config, dbContainerName)),
			fmt.Sprintf("%s=%s", "PG_META_DB_PORT", "5432"),
			fmt.Sprintf("%s=%s", "PG_META_DB_NAME", "postgres"),
			fmt.Sprintf("%s=%s", "PG_META_DB_USER", "supabase_admin"),
			fmt.Sprintf("%s=%s", "PG_META_DB_PASSWORD", config.Database.Password),
		},
	}
}

func (T TServices) Postgres(config Config) *Service {

	// folder for storing database data
	if info, err := os.Stat(config.Database.DataDirectory); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(config.Database.DataDirectory, 0o700); err != nil {
				panic(fmt.Sprintf("postgres data directory \"%s\" does not exist and an error occurred while trying to create it: %v", config.Database.DataDirectory, err))
			}
		} else {
			panic(fmt.Sprintf("error checking postgres data directory \"%s\" exists: %v", config.Database.DataDirectory, err))
		}
	} else if !info.IsDir() {
		panic(fmt.Sprintf("postgres data directory \"%s\" exists but is not a directory", config.Database.DataDirectory))
	}

	// folder for storing custom db files
	if info, err := os.Stat(config.Database.ConfigDirectory); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(config.Database.ConfigDirectory, 0o700); err != nil {
				panic(fmt.Sprintf("postgres config directory \"%s\" does not exist and an error occurred while trying to create it: %v", config.Database.ConfigDirectory, err))
			}
		} else {
			panic(fmt.Sprintf("error checking postgres config directory \"%s\" exists: %v", config.Database.ConfigDirectory, err))
		}
	} else if !info.IsDir() {
		panic(fmt.Sprintf("postgres config directory \"%s\" exists but is not a directory", config.Database.ConfigDirectory))
	}

	// pg sodium key file
	pgSodiumRootKeyFile := filepath.Join(config.Database.ConfigDirectory, "pgsodium_root.key")
	if info, err := os.Stat(pgSodiumRootKeyFile); err != nil {
		if os.IsNotExist(err) {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				panic(fmt.Sprintf("postgres pgsodium key file \"%s\" does not exist and an error occurred while trying to create it: %v", pgSodiumRootKeyFile, err))
			}
			if err := os.WriteFile(pgSodiumRootKeyFile, []byte(hex.EncodeToString(b)), 0o700); err != nil {
				panic(fmt.Sprintf("postgres pgsodium key file \"%s\" does not exist and an error occurred while trying to create it: %v", pgSodiumRootKeyFile, err))
			}
		} else {
			panic(fmt.Sprintf("error checking postgres pgsodium key file \"%s\" exists: %v", pgSodiumRootKeyFile, err))
		}
	} else if info.IsDir() {
		panic(fmt.Sprintf("postgres pgsodium key file \"%s\" exists but is not a file", pgSodiumRootKeyFile))
	}

	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   pgSodiumRootKeyFile,
			Target:   "/etc/postgresql-custom/pgsodium_root.key",
			ReadOnly: true,
		},
		{
			Type:   mount.TypeBind,
			Source: config.Database.DataDirectory,
			Target: "/var/lib/postgresql/data",
		},
	}

	for _, file := range []EmbeddedFile{
		{
			Name: "realtime.sql",
			Data: postgres.RealtimeSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/99-realtime.sql",
		},
		{
			Name: "webhooks.sql",
			Data: postgres.WebhooksSQL,
			Path: "/docker-entrypoint-initdb.d/init-scripts/98-webhooks.sql",
		},
		{
			Name: "roles.sql",
			Data: postgres.RolesSQL,
			Path: "/docker-entrypoint-initdb.d/init-scripts/99-roles.sql",
		},
		{
			Name: "jwt.sql",
			Data: postgres.JwtSQL,
			Path: "/docker-entrypoint-initdb.d/init-scripts/99-jwt.sql",
		},
		{
			Name: "_supabase.sql",
			Data: postgres.SupabaseSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/97-_supabase.sql",
		},
		{
			Name: "logs.sql",
			Data: postgres.LogsSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/99-logs.sql",
		},
		{
			Name: "pooler.sql",
			Data: postgres.PoolerSQL,
			Path: "/docker-entrypoint-initdb.d/migrations/99-pooler.sql",
		},
	} {
		mnt, err := file.Mount()
		if err != nil {
			panic(err)
		}
		mounts = append(mounts, *mnt)
	}

	return &Service{
		Image:       "supabase/postgres:17.4.1.055",
		Name:        containerName(config, dbContainerName),
		Mounts:      mounts,
		Ports:       make([]uint16, 0),
		StopTimeout: utils.Pointer(10 * time.Second),
		Aliases:     []string{"db"},
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
			"-c", "archive_mode=off",
		},
		Env: []string{
			"POSTGRES_HOST=/var/run/postgresql",
			"PGPORT=5432",
			"POSTGRES_PORT=5432",
			fmt.Sprintf("PGPASSWORD=%s", config.Database.Password),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Database.Password),
			"PGDATABASE=postgres",
			"POSTGRES_DB=postgres",
			fmt.Sprintf("JWT_SECRET=%s", config.Keys.JwtSecret),
			"JWT_EXP=3600",
		},
		AfterStart: func(ctx context.Context, docker *client.Client, cid string) error {
			// patch postgres password
			p := config.Database.Password
			output, err := utils.ExecInContainer(ctx, docker, cid, []string{
				"psql",
				"-h", "127.0.0.1",
				"-U", "supabase_admin",
				"-d", "postgres",
				"-v", "ON_ERROR_STOP=1",
				"-c",
				fmt.Sprintf(`
ALTER USER anon                    WITH PASSWORD '%s';
ALTER USER authenticated           WITH PASSWORD '%s';
ALTER USER authenticator           WITH PASSWORD '%s';
ALTER USER dashboard_user          WITH PASSWORD '%s';
ALTER USER pgbouncer               WITH PASSWORD '%s';
ALTER USER postgres                WITH PASSWORD '%s';
ALTER USER service_role            WITH PASSWORD '%s';
ALTER USER supabase_admin          WITH PASSWORD '%s';
ALTER USER supabase_auth_admin     WITH PASSWORD '%s';
ALTER USER supabase_read_only_user WITH PASSWORD '%s';
ALTER USER supabase_replication_admin WITH PASSWORD '%s';
ALTER USER supabase_storage_admin  WITH PASSWORD '%s';
`, p, p, p, p, p, p, p, p, p, p, p, p),
			})
			if err != nil {
				return fmt.Errorf("failed to patch postgres password: %v (%s)", err, strings.ReplaceAll(strings.TrimSpace(output), "\n", "\\n"))
			}
			return nil
		},
	}
}

func (T TServices) Postgrest(config Config) *Service {
	return &Service{
		Image:   "postgrest/postgrest:v12.2.12",
		Name:    containerName(config, "supago-rest"),
		Aliases: []string{"rest"},
		Cmd:     []string{"postgrest"},
		Env: []string{
			fmt.Sprintf("PGRST_DB_URI=postgres://authenticator:%s@%s:5432/postgres", config.Database.Password, containerName(config, dbContainerName)),
			"PGRST_DB_SCHEMAS=public",
			"PGRST_DB_ANON_ROLE=anon",
			fmt.Sprintf("PGRST_JWT_SECRET=%s", config.Keys.JwtSecret),
			"PGRST_DB_USE_LEGACY_GUCS=false",
			fmt.Sprintf("PGRST_APP_SETTINGS_JWT_SECRET=%s", config.Keys.JwtSecret),
			"PGRST_APP_SETTINGS_JWT_EXP=3600",
			"PGRST_ADMIN_SERVER_PORT=3001",
		},
	}
}

func (T TServices) Realtime(config Config) *Service {
	return &Service{
		Name:  "realtime-dev.supabase-realtime",
		Image: "supabase/realtime:v2.34.47",
		Aliases: []string{
			"supago-realtime",
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
				fmt.Sprintf("Authorization: Bearer %s", config.Keys.PublicJwt),
				"http://localhost:4000/api/tenants/realtime-dev/health",
			},
			Interval: 5 * time.Second,
			Timeout:  5 * time.Second,
			Retries:  3,
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "PORT", "4000"),
			fmt.Sprintf("%s=%s", "DB_HOST", containerName(config, dbContainerName)),
			fmt.Sprintf("%s=%s", "DB_PORT", "5432"),
			fmt.Sprintf("%s=%s", "DB_USER", "supabase_admin"),
			fmt.Sprintf("%s=%s", "DB_PASSWORD", config.Database.Password),
			fmt.Sprintf("%s=%s", "DB_NAME", "postgres"),
			fmt.Sprintf("%s=%s", "DB_AFTER_CONNECT_QUERY", "SET search_path TO _realtime"),
			fmt.Sprintf("%s=%s", "DB_ENC_KEY", "supabaserealtime"),
			fmt.Sprintf("%s=%s", "API_JWT_SECRET", config.Keys.JwtSecret),
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

func (T TServices) Storage(config Config) *Service {
	if info, err := os.Stat(config.Storage.DataDirectory); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(config.Storage.DataDirectory, 0o700); err != nil {
				panic(fmt.Sprintf("storage data directory \"%s\" does not exist and an error occurred while trying to create it: %v", config.Storage.DataDirectory, err))
			}
		} else {
			panic(fmt.Sprintf("error checking storage data directory \"%s\" exists: %v", config.Storage.DataDirectory, err))
		}
	} else if !info.IsDir() {
		panic(fmt.Sprintf("storage directory \"%s\" exists but is not a directory", config.Storage.DataDirectory))
	}

	return &Service{
		Name:    containerName(config, "supabase-storage"),
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
				Source: config.Storage.DataDirectory,
				Target: "/var/lib/storage",
			},
		},
		Env: []string{
			fmt.Sprintf("%s=%s", "ANON_KEY", config.Keys.PublicJwt),
			fmt.Sprintf("%s=%s", "SERVICE_KEY", config.Keys.PrivateJwt),
			fmt.Sprintf("%s=%s", "POSTGREST_URL", "http://rest:3000"),
			fmt.Sprintf("%s=%s", "PGRST_JWT_SECRET", config.Keys.JwtSecret),
			fmt.Sprintf("%s=%s", "DATABASE_URL",
				fmt.Sprintf("postgres://supabase_storage_admin:%s@%s:5432/postgres", config.Database.Password, containerName(config, dbContainerName))),
			fmt.Sprintf("%s=%s", "FILE_SIZE_LIMIT", "52428800"),
			fmt.Sprintf("%s=%s", "STORAGE_BACKEND", "file"),
			fmt.Sprintf("%s=%s", "FILE_STORAGE_BACKEND_PATH", "/var/lib/storage"),
			fmt.Sprintf("%s=%s", "TENANT_ID", "stub"),
			fmt.Sprintf("%s=%s", "REGION", "stub"),
			fmt.Sprintf("%s=%s", "GLOBAL_S3_BUCKET", "stub"),
			fmt.Sprintf("%s=%s", "ENABLE_IMAGE_TRANSFORMATION", "true"),
			fmt.Sprintf("%s=%s", "IMGPROXY_URL", "http://imgproxy:5001"),
		},
	}
}

func (T TServices) Studio(config Config) *Service {
	return &Service{
		Image:   "supabase/studio:2025.06.30-sha-6f5982d",
		Name:    containerName(config, "supabase-studio"),
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
			fmt.Sprintf("%s=%s", "POSTGRES_PASSWORD", config.Database.Password),

			fmt.Sprintf("%s=%s", "DEFAULT_ORGANIZATION_NAME", "Supago"),
			fmt.Sprintf("%s=%s", "DEFAULT_PROJECT_NAME", "Supago"),
			//fmt.Sprintf("%s=%s", "OPENAI_API_KEY", ""),

			fmt.Sprintf("%s=%s", "SUPABASE_URL", "http://kong:8000"),
			fmt.Sprintf("%s=%s", "SUPABASE_PUBLIC_URL", "http://127.0.0.1:8000"),
			fmt.Sprintf("%s=%s", "SUPABASE_ANON_KEY", config.Keys.PublicJwt),
			fmt.Sprintf("%s=%s", "SUPABASE_SERVICE_KEY", config.Keys.PrivateJwt),
			fmt.Sprintf("%s=%s", "AUTH_JWT_SECRET", config.Keys.JwtSecret),

			fmt.Sprintf("%s=%s", "LOGFLARE_PRIVATE_ACCESS_TOKEN", config.LogFlare.PrivateKey),
			fmt.Sprintf("%s=%s", "LOGFLARE_URL", "http://analytics:4000"),
			fmt.Sprintf("%s=%s", "NEXT_PUBLIC_ENABLE_LOGS", "true"),
			fmt.Sprintf("%s=%s", "NEXT_ANALYTICS_BACKEND_PROVIDER", "postgres"),
		},
	}
}

// All get all services supported by SupaGo
func (T TServices) All() AllServices {
	return func(config Config) *[]*Service {
		return &[]*Service{
			T.Postgres(config),
			T.ImgProxy(config),
			T.Kong(config),
			T.Analytics(config),
			T.Auth(config),
			T.Meta(config),
			T.Postgrest(config),
			T.Storage(config),
			T.Realtime(config),
			T.Studio(config),
		}
	}
}
