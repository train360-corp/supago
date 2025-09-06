package services

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/train360-corp/supago/pkg/types"
	"github.com/train360-corp/supago/pkg/utils"
	"io"
	"strconv"
)

func (runner *Runner) ensureNetwork(ctx context.Context) error {

	if runner.network != nil {
		utils.Logger().Debug("skipping network initialization (exists)")
		return nil
	}

	utils.Logger().Debug("ensuring network exists")
	args := filters.NewArgs(filters.Arg("name", runner.networkName))

	var queryNets func(triedCreate bool) error
	queryNets = func(triedCreate bool) error {
		if nets, err := runner.docker.NetworkList(ctx, network.ListOptions{Filters: args}); err != nil {
			utils.Logger().Debugf("failed to list networks: %v", err)
			return fmt.Errorf("failed to list networks: %v", err)
		} else if len(nets) == 0 {
			if triedCreate { // guard for infinite recursion
				utils.Logger().Errorf("tried to create network but still not found")
				return fmt.Errorf("tried to create network but still not found")
			}
			utils.Logger().Debugf("no existing network found; attempting to create a new network")
			if _, err := runner.docker.NetworkCreate(ctx, runner.networkName, network.CreateOptions{
				Driver: "bridge",
				Scope:  "local",
				IPAM: &network.IPAM{
					Driver: "default",
					Config: []network.IPAMConfig{{Subnet: "172.30.0.0/16"}},
				},
				EnableIPv4: utils.Pointer(true),
				EnableIPv6: utils.Pointer(true),
				Internal:   false, // true = no external connectivity (usually keep false)
				Attachable: true,  // allow standalone containers to attach/detach
			}); err != nil {
				utils.Logger().Errorf("failed to create network: %v", err)
				return fmt.Errorf("failed to create network: %v", err)
			} else { // retry creation, subject to guard-clause above
				return queryNets(true)
			}
		} else if len(nets) == 1 {
			utils.Logger().Debugf("found existing network: %s", utils.ShortStr(nets[0].ID))
			runner.network = &nets[0]
			return nil
		} else if len(nets) > 1 {
			utils.Logger().Debugf("found multiple networks: %v", nets)
			return fmt.Errorf("found multiple networks: %v", nets)
		} else {
			utils.Logger().Errorf("an unexpected edge-case occurred while ensuring network")
			return fmt.Errorf("an unexpected edge-case occurred while ensuring network")
		}
	}

	return queryNets(false)
}

func ports(svc *types.Service) (nat.PortSet, nat.PortMap) {
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range svc.Ports {
		port := nat.Port(fmt.Sprintf("%d/tcp", p))
		exposedPorts[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{
			{
				HostIP:   "127.0.0.1",
				HostPort: strconv.Itoa(int(p)),
			},
		}
	}
	return exposedPorts, portBindings
}

func (runner *Runner) pull(ctx context.Context, svc *types.Service) error {
	utils.Logger().Debugf("pulling image for %v", svc)
	pull, err := runner.docker.ImagePull(ctx, svc.Image, image.PullOptions{})
	if err != nil {
		utils.Logger().Errorf("failed to pull image for %v: %v", svc, err)
		return fmt.Errorf("failed to pull image for %v: %v", svc, err)
	}
	if pull != nil {
		_, _ = io.Copy(io.Discard, pull)
		_ = pull.Close()
	}
	utils.Logger().Debugf("pulled image for %v", svc)
	return nil
}

func (runner *Runner) create(ctx context.Context, svc *types.Service) (*container.CreateResponse, error) {
	utils.Logger().Debugf("creating container for %v", svc)

	if runner.network == nil {
		utils.Logger().Errorf("network not initialized")
		return nil, fmt.Errorf("network not initialized")
	}

	exposedPorts, portBindings := ports(svc)
	if resp, err := runner.docker.ContainerCreate(ctx,
		&container.Config{
			Image:        svc.Image,
			Cmd:          svc.Cmd,
			Env:          svc.Env,
			OpenStdin:    true,  // keep a stdin pipe open from our process
			StdinOnce:    true,  // when our stdin attach disconnects, close container's STDIN
			Tty:          false, // keep streams multiplexed for stdcopy
			ExposedPorts: exposedPorts,
			Labels:       svc.Labels,
		},
		&container.HostConfig{
			AutoRemove:    true, // like --rm
			RestartPolicy: container.RestartPolicy{Name: "no"},
			NetworkMode:   container.NetworkMode(runner.network.Name),
			Mounts:        svc.Mounts,
			PortBindings:  portBindings,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				runner.network.Name: {
					NetworkID: runner.network.ID,
					Aliases:   svc.Aliases,
				},
			},
		},
		nil,
		svc.Name,
	); err != nil {
		utils.Logger().Errorf("failed to create container for %v: %v", svc, err)
		return nil, fmt.Errorf("failed to create container for %v: %v", svc, err)
	} else {
		utils.Logger().Debugf("created container for %v: %v", svc, utils.ShortStr(resp.ID))
		return &resp, nil
	}
}
