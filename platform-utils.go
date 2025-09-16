package supago

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/train360-corp/supago/internal/utils"
	"io"
	"regexp"
	"strconv"
	"time"
)

func (sg *SupaGo) setupDocker() error {
	if c, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	); err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	} else {
		sg.docker = c
	}
	sg.logger.Debugf("Docker Host: %v", sg.docker.DaemonHost())

	// fast, cheap liveness probe
	ping, err := sg.docker.Ping(context.Background())
	if err != nil {
		return fmt.Errorf("docker daemon not reachable: %v", err)
	} else {
		if ping.APIVersion != "" {
			sg.logger.Debugf("Docker API version: %v", ping.APIVersion)
		}
		if ping.OSType != "" {
			sg.logger.Debugf("Docker OS: %v", ping.OSType)
		}
		if ping.BuilderVersion != "" {
			sg.logger.Debugf("Docker Builder: %v", ping.BuilderVersion)
		}
		sg.logger.Debugf("Docker Experimental: %v", ping.Experimental)
	}
	return nil
}

func (sg *SupaGo) ensureNetwork(ctx context.Context) error {

	if sg.network != nil {
		sg.logger.Debug("skipping network initialization (exists)")
		return nil
	}

	sg.logger.Debug("ensuring network exists")
	args := filters.NewArgs(filters.Arg("name", sg.config.Global.PlatformName))

	var queryNets func(triedCreate bool) error
	queryNets = func(triedCreate bool) error {
		if nets, err := sg.docker.NetworkList(ctx, network.ListOptions{Filters: args}); err != nil {
			sg.logger.Debugf("failed to list networks: %v", err)
			return fmt.Errorf("failed to list networks: %v", err)
		} else if len(nets) == 0 {
			if triedCreate { // guard for infinite recursion
				sg.logger.Errorf("tried to create network but still not found")
				return fmt.Errorf("tried to create network but still not found")
			}
			sg.logger.Debugf("no existing network found; attempting to create a new network")
			if _, err := sg.docker.NetworkCreate(ctx, sg.config.Global.PlatformName, network.CreateOptions{
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
				sg.logger.Errorf("failed to create network: %v", err)
				return fmt.Errorf("failed to create network: %v", err)
			} else { // retry creation, subject to guard-clause above
				return queryNets(true)
			}
		} else if len(nets) == 1 {
			sg.logger.Debugf("found existing network: %s", utils.ShortStr(nets[0].ID))
			sg.network = &nets[0]
			return nil
		} else if len(nets) > 1 {
			sg.logger.Debugf("found multiple networks: %v", nets)
			return fmt.Errorf("found multiple networks: %v", nets)
		} else {
			sg.logger.Errorf("an unexpected edge-case occurred while ensuring network")
			return fmt.Errorf("an unexpected edge-case occurred while ensuring network")
		}
	}

	return queryNets(false)
}

func ports(svc *Service) (nat.PortSet, nat.PortMap) {
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

func (sg *SupaGo) removeContainerByName(ctx context.Context, containerName string) {
	sg.logger.Debugf("removing container: %v", containerName)
	err := sg.docker.ContainerRemove(ctx, containerName, container.RemoveOptions{
		RemoveVolumes: true, // also remove anonymous volumes
		Force:         true, // stop and remove even if running
	})
	if err != nil {
		sg.logger.Debugf("failed to remove container \"%s\": %v", containerName, err)
	}
}

func (sg *SupaGo) pullImage(ctx context.Context, svc *Service) error {
	sg.logger.Debugf("pulling image for %v", svc)
	pull, err := sg.docker.ImagePull(ctx, svc.Image, image.PullOptions{})
	if err != nil {
		sg.logger.Errorf("failed to pull image for %v: %v", svc, err)
		return fmt.Errorf("failed to pull image for %v: %v", svc, err)
	}
	if pull != nil {
		_, _ = io.Copy(io.Discard, pull)
		_ = pull.Close()
	}
	sg.logger.Debugf("pulled image for %v", svc)
	return nil
}

func (sg *SupaGo) createContainer(ctx context.Context, svc *Service) (*container.CreateResponse, error) {
	sg.logger.Debugf("creating container for %v", svc)

	if sg.network == nil {
		sg.logger.Errorf("network not initialized")
		return nil, fmt.Errorf("network not initialized")
	}

	if svc.Labels == nil {
		svc.Labels = map[string]string{}
	}
	if IsValidPlatformName(sg.config.Global.PlatformName) {
		svc.Labels["com.docker.compose.project"] = sg.config.Global.PlatformName
	} else {
		svc.Labels["com.docker.compose.project"] = "supago"
	}

	exposedPorts, portBindings := ports(svc)
	if resp, err := sg.docker.ContainerCreate(ctx,
		&container.Config{
			Image:        svc.Image,
			Entrypoint:   svc.Entrypoint,
			Cmd:          svc.Cmd,
			Env:          svc.Env,
			OpenStdin:    false,
			StdinOnce:    false,
			Tty:          false,
			ExposedPorts: exposedPorts,
			Labels:       svc.Labels,
		},
		&container.HostConfig{
			AutoRemove:    false,
			RestartPolicy: container.RestartPolicy{Name: "no"},
			NetworkMode:   container.NetworkMode(sg.network.Name),
			Mounts:        svc.Mounts,
			PortBindings:  portBindings,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				sg.network.Name: {
					NetworkID: sg.network.ID,
					Aliases:   svc.Aliases,
				},
			},
		},
		nil,
		svc.Name,
	); err != nil {
		sg.logger.Errorf("failed to create container for %v: %v", svc, err)
		return nil, fmt.Errorf("failed to create container for %v: %v", svc, err)
	} else {
		sg.logger.Debugf("created container for %v: %v", svc, utils.ShortStr(resp.ID))
		return &resp, nil
	}
}

func (sg *SupaGo) startContainer(ctx context.Context, svc *Service) error {
	sg.logger.Debugf("starting %v conainter %s", svc, utils.ShortStr(svc.container.ID))
	return sg.docker.ContainerStart(ctx, svc.container.ID, container.StartOptions{})
}

func (sg *SupaGo) healthcheckContainer(ctx context.Context, svc *Service, retry int) error {

	sg.logger.Debugf("checking %v container %s health (retry=%d)", svc, utils.ShortStr(svc.container.ID), retry)
	inspected, err := sg.docker.ContainerInspect(ctx, svc.container.ID)
	if err != nil {
		sg.logger.Errorf("unable to check %v container %s health: %v", svc, utils.ShortStr(svc.container.ID), err)
		return fmt.Errorf("unable to check %v container %s health: %v", svc, utils.ShortStr(svc.container.ID), err)
	}

	if inspected.State != nil && inspected.State.Health != nil {
		switch inspected.State.Health.Status {
		case container.NoHealthcheck:
			sg.logger.Warnf("%v container %s does not have a health-check (continuing)", svc, utils.ShortStr(svc.container.ID))
			return nil
		case container.Healthy:
			sg.logger.Debugf("%v container %s is healthy (continuing)", svc, utils.ShortStr(svc.container.ID))
			return nil
		case container.Starting:
			sg.logger.Debugf("%v container %s is still starting (retrying in 5 seconds...)", svc, utils.ShortStr(svc.container.ID))
			time.Sleep(5 * time.Second)
			return sg.healthcheckContainer(ctx, svc, retry+1)
		case container.Unhealthy:
			sg.logger.Errorf("%v container %s is unhealthy", svc, utils.ShortStr(svc.container.ID))
			return fmt.Errorf("%v container %s is unhealthy", svc, utils.ShortStr(svc.container.ID))
		default:
			sg.logger.Errorf("%v container %s unhandled status: %v", svc, utils.ShortStr(svc.container.ID), inspected.State.Health.Status)
			return fmt.Errorf("%v container %s unhandled status: %v", svc, utils.ShortStr(svc.container.ID), inspected.State.Health.Status)
		}
	}

	return nil
}

func (sg *SupaGo) stopContainer(service *Service) {
	stopOptions := container.StopOptions{
		Timeout: utils.Pointer(15),
	}
	if service.StopTimeout != nil {
		stopOptions.Timeout = utils.Pointer(int(service.StopTimeout.Seconds()))
	}
	if service.StopSignal != nil {
		stopOptions.Signal = *service.StopSignal
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(*stopOptions.Timeout)*time.Second)
	defer cancel()
	if service.container == nil {
		sg.logger.Debugf("container shutdown skipped for %s (no container initialized)", service)
	} else {
		sg.logger.Debugf("stopping %v container %s", service, utils.ShortStr(service.container.ID))
		if err := sg.docker.ContainerStop(shutdownCtx, service.container.ID, stopOptions); err != nil {
			sg.logger.Errorf("failed to stop %v container %s: %v", service, utils.ShortStr(service.container.ID), err)
		} else {
			sg.logger.Warnf("%v stopped", service)
		}
	}

}

func (sg *SupaGo) removeContainer(service *Service) {

	timeout := 15 * time.Second
	if service.StopTimeout != nil {
		timeout = *service.StopTimeout
	}

	removeCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if service.container == nil {
		sg.logger.Debugf("container removal skipped for %s (no container initialized)", service)
	} else {
		sg.logger.Debugf("removing %v container %s ", service, utils.ShortStr(service.container.ID))
		if err := sg.docker.ContainerRemove(removeCtx, service.container.ID, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false, // causing bugs when true
			Force:         true,
		}); err != nil {
			isAlreadyShuttingDownError := regexp.MustCompile(`^Error response from daemon: removal of container [a-f0-9]+ is already in progress$`)
			if isAlreadyShuttingDownError.MatchString(err.Error()) {
				sg.logger.Debugf("confirmed removal of %v container %s in progress", service, utils.ShortStr(service.container.ID))
			} else {
				sg.logger.Errorf("failed to remove %v container %s: %v", service, utils.ShortStr(service.container.ID), err)
			}
		} else {
			sg.logger.Debugf("removed %v container", service)
		}
	}
}
