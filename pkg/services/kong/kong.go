package kong

import (
	_ "embed"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/train360-corp/supago/pkg/types"
)

const ContainerName = "supago-kong"

//go:embed kong.yml
var ConfigFile []byte

type Keys struct {
	Public  string
	Private string
}

type Dashboard struct {
	Username string
	Password string
}

type Props struct {
	Keys      Keys
	Dashboard Dashboard
}

func Service(props Props) (*types.Service, error) {

	config := types.EmbeddedFile{
		Name: "kong.yml",
		Data: ConfigFile,
		Path: "/usr/local/kong-template.yml",
	}

	var mounts []mount.Mount
	if mnt, err := config.Mount(); err != nil {
		return nil, err
	} else {
		mounts = append(mounts, *mnt)
	}

	return &types.Service{
		Image:   "kong:2.8.1",
		Name:    ContainerName,
		Aliases: []string{"kong"},
		Labels: map[string]string{
			"supago.service": ContainerName,
		},
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
			fmt.Sprintf("%s=%s", "SUPABASE_ANON_KEY", props.Keys.Public),
			fmt.Sprintf("%s=%s", "SUPABASE_SERVICE_KEY", props.Keys.Private),
			fmt.Sprintf("%s=%s", "DASHBOARD_USERNAME", props.Dashboard.Username),
			fmt.Sprintf("%s=%s", "DASHBOARD_PASSWORD", props.Dashboard.Password),
		},
	}, nil
}
