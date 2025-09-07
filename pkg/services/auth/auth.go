package auth

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/train360-corp/supago/pkg/services/postgres"
	"github.com/train360-corp/supago/pkg/types"
	"time"
)

type SMTPFrom struct {
	Email string
	Name  string
}

type SMTP struct {
	Host string
	Port uint16
	User string
	Pass string
	From SMTPFrom
}

type Keys struct {
	Secret string
}

type Database struct {
	Password string
}

type URLs struct {
	Site string // where the frontend Site is publicly accessible
	Kong string // where Kong is publicly accessible
}

type Props struct {
	URLs URLs
	SMTP SMTP
	DB   Database
	Keys Keys
}

const ContainerName = "supago-auth"

func Service(props Props) *types.Service {
	return &types.Service{
		Name:    ContainerName,
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
			fmt.Sprintf("%s=%s", "API_EXTERNAL_URL", props.URLs.Kong),

			fmt.Sprintf("%s=%s", "GOTRUE_DB_DRIVER", "postgres"),
			fmt.Sprintf("%s=%s", "GOTRUE_DB_DATABASE_URL",
				fmt.Sprintf("postgres://supabase_auth_admin:%s@%s:5432/postgres", props.DB.Password, postgres.ContainerName)),

			fmt.Sprintf("%s=%s", "GOTRUE_SITE_URL", props.URLs.Site),
			fmt.Sprintf("%s=%s", "GOTRUE_URI_ALLOW_LIST", ""),
			fmt.Sprintf("%s=%s", "GOTRUE_DISABLE_SIGNUP", "false"),

			fmt.Sprintf("%s=%s", "GOTRUE_JWT_ADMIN_ROLES", "service_role"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_AUD", "authenticated"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_DEFAULT_GROUP_NAME", "authenticated"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_EXP", "3600"),
			fmt.Sprintf("%s=%s", "GOTRUE_JWT_SECRET", props.Keys.Secret),

			fmt.Sprintf("%s=%s", "GOTRUE_EXTERNAL_EMAIL_ENABLED", "true"),
			fmt.Sprintf("%s=%s", "GOTRUE_EXTERNAL_ANONYMOUS_USERS_ENABLED", "false"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_AUTOCONFIRM", "false"),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_ADMIN_EMAIL", props.SMTP.From.Email),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_HOST", props.SMTP.Host),
			fmt.Sprintf("%s=%d", "GOTRUE_SMTP_PORT", props.SMTP.Port),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_USER", props.SMTP.User),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_PASS", props.SMTP.Pass),
			fmt.Sprintf("%s=%s", "GOTRUE_SMTP_SENDER_NAME", props.SMTP.From.Name),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_INVITE", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_CONFIRMATION", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_RECOVERY", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_MAILER_URLPATHS_EMAIL_CHANGE", "/auth/v1/verify"),
			fmt.Sprintf("%s=%s", "GOTRUE_EXTERNAL_PHONE_ENABLED", "false"),
			fmt.Sprintf("%s=%s", "GOTRUE_SMS_AUTOCONFIRM", "false"),
		},
	}
}
