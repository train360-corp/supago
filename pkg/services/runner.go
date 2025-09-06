package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/train360-corp/supago/pkg/types"
	"github.com/train360-corp/supago/pkg/utils"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	docker                *client.Client
	network               *network.Summary
	networkName           string
	isProperlyInitialized bool
}

func NewRunner(networkName string) (*Runner, error) {
	var runner Runner
	runner.networkName = strings.TrimSpace(networkName)
	if runner.networkName == "" {
		return nil, errors.New("runner's NetworkName is required but empty")
	}

	if c, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	); err != nil {
		return nil, fmt.Errorf("failed to create docker client: %v", err)
	} else {
		runner.docker = c
	}
	utils.Logger().Infof("Docker Host: %v", runner.docker.DaemonHost())

	// fast, cheap liveness probe
	ping, err := runner.docker.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("docker daemon not reachable: %v", err)
	} else {
		if ping.APIVersion != "" {
			utils.Logger().Debugf("Docker API version: %v", ping.APIVersion)
		}
		if ping.OSType != "" {
			utils.Logger().Debugf("Docker OS: %v", ping.OSType)
		}
		if ping.BuilderVersion != "" {
			utils.Logger().Debugf("Docker Builder: %v", ping.BuilderVersion)
		}
		utils.Logger().Debugf("Docker Experimental: %v", ping.Experimental)
	}

	runner.isProperlyInitialized = true
	return &runner, nil
}

func (runner *Runner) Run(ctx context.Context, svc *types.Service) (func(context.Context), error) {

	if !runner.isProperlyInitialized {
		utils.Logger().Debugf("detected improperly initialized Runner")
		return nil, fmt.Errorf("runner not properly initialized; Runner should not be directly instantiated (call the services.NewRunner(...) func instead)")
	}

	utils.Logger().Infof("starting %v", svc)

	// ensure network exists
	if err := runner.ensureNetwork(ctx); err != nil {
		return nil, err
	}

	// pull image
	if err := runner.pull(ctx, svc); err != nil {
		return nil, err
	}

	// create container
	if ctr, err := runner.create(ctx, svc); err != nil {
		return nil, err
	} else {
		var once sync.Once
		stopContainer := func(ctx context.Context) {
			once.Do(func() {
				utils.Logger().Debugf("stopping %v container %s", svc, utils.ShortStr(ctr.ID))
				stopOptions := container.StopOptions{}
				if svc.StopTimeout != nil {
					stopOptions.Timeout = utils.Pointer(int(svc.StopTimeout.Seconds()))
				}
				if svc.StopSignal != nil {
					stopOptions.Signal = *svc.StopSignal
				}
				if err := runner.docker.ContainerStop(ctx, ctr.ID, stopOptions); err != nil {
					utils.Logger().Errorf("failed to stop %v container %s: %v", svc, utils.ShortStr(ctr.ID), err)
				} else {
					utils.Logger().Warnf("stopped %v container", svc)
				}
			})
		}

		removeContainer := func(ctx context.Context) {
			utils.Logger().Debugf("removing %v container %s ", svc, utils.ShortStr(ctr.ID))
			if err := runner.docker.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{
				RemoveVolumes: true,
				RemoveLinks:   true,
				Force:         true,
			}); err != nil {
				isAlreadyShuttingDownError := regexp.MustCompile(`^Error response from daemon: removal of container [a-f0-9]+ is already in progress$`)
				if isAlreadyShuttingDownError.MatchString(err.Error()) {
					utils.Logger().Debugf("confirmed removal of %v container %s in progress", svc, utils.ShortStr(ctr.ID))
				} else {
					utils.Logger().Errorf("failed to remove %v container %s: %v", svc, utils.ShortStr(ctr.ID), err)
				}
			} else {
				utils.Logger().Debugf("removed %v container", svc)
			}
		}

		var healthcheck func(retry int) error
		healthcheck = func(retry int) error {
			utils.Logger().Debugf("checking %v container %s health (retry=%d)", svc, utils.ShortStr(ctr.ID), retry)
			inspected, err := runner.docker.ContainerInspect(ctx, ctr.ID)
			if err != nil {
				utils.Logger().Errorf("unable to check %v container %s health: %v", svc, utils.ShortStr(ctr.ID), err)
				return fmt.Errorf("unable to check %v container %s health: %v", svc, utils.ShortStr(ctr.ID), err)
			}
			switch inspected.State.Health.Status {
			case container.NoHealthcheck:
				utils.Logger().Warnf("%v container %s does not have a health-check (continuing)", svc, utils.ShortStr(ctr.ID))
				return nil
			case container.Healthy:
				utils.Logger().Debugf("%v container %s is healthy (continuing)", svc, utils.ShortStr(ctr.ID))
				return nil
			case container.Starting:
				utils.Logger().Debugf("%v container %s is still starting (retrying in 5 seconds...)", svc, utils.ShortStr(ctr.ID))
				time.Sleep(5 * time.Second)
				return healthcheck(retry + 1)
			case container.Unhealthy:
				utils.Logger().Errorf("%v container %s is unhealthy", svc, utils.ShortStr(ctr.ID))
				return fmt.Errorf("%v container %s is unhealthy", svc, utils.ShortStr(ctr.ID))
			default:
				utils.Logger().Errorf("%v container %s unhandled status: %v", svc, utils.ShortStr(ctr.ID), inspected.State.Health.Status)
				return fmt.Errorf("%v container %s unhandled status: %v", svc, utils.ShortStr(ctr.ID), inspected.State.Health.Status)
			}
		}

		// start container
		utils.Logger().Debugf("starting %v conainter %s", svc, utils.ShortStr(ctr.ID))
		if err := runner.docker.ContainerStart(ctx, ctr.ID, container.StartOptions{}); err != nil {
			utils.Logger().Errorf("failed to start %v container: %v", svc, err)
			stopContainer(ctx)
			removeContainer(ctx)
			return nil, fmt.Errorf("failed to start %v container: %v", svc, err)
		}

		// listen to status
		utils.Logger().Debugf("listening to %v container %s status", svc, utils.ShortStr(ctr.ID))
		statusCh, errCh := runner.docker.ContainerWait(ctx, ctr.ID, container.WaitConditionNotRunning)
		go func() {
			select {
			case err := <-errCh:
				if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					utils.Logger().Errorf("wait error for %v container %s: %v", svc, utils.ShortStr(ctr.ID), err)
				} else {
					utils.Logger().Debugf("wait exited (context-cancelled) for %v container %s", svc, utils.ShortStr(ctr.ID))
				}
				stopContainer(context.Background())
				removeContainer(context.Background())
			case st := <-statusCh: // container exited (possibly immediately)
				utils.Logger().Debugf("%v container %s exited with status: %v", svc, utils.ShortStr(ctr.ID), st)
				stopContainer(ctx)
				removeContainer(ctx)
			}
		}()

		// perform healthcheck
		if err := healthcheck(0); err != nil {
			stopContainer(ctx)
			removeContainer(ctx)
			return nil, err
		}

		// after start
		if svc.AfterStart != nil {
			if err := svc.AfterStart(ctx, runner.docker, ctr.ID); err != nil {
				stopContainer(ctx)
				removeContainer(ctx)
				return nil, fmt.Errorf("AfterStart failed for %v: %v", svc, err)
			}
		}

		// attach to container (keep this connection open while app is alive)
		if att, err := runner.docker.ContainerAttach(ctx, ctr.ID, container.AttachOptions{
			Stdin:  true,
			Stream: true,
			Stdout: false,
			Stderr: false,
		}); err != nil {
			utils.Logger().Errorf("attach failed for %v container %s: %v", svc, utils.ShortStr(ctr.ID), err)
			stopContainer(ctx)
			removeContainer(ctx)
			return nil, fmt.Errorf("attach failed for %v container %s: %v", svc, utils.ShortStr(ctr.ID), err)
		} else {
			detach := func(ctx context.Context) {
				utils.Logger().Debugf("stop and detach sequence triggered for %v container %s", svc, utils.ShortStr(ctr.ID))

				if err := att.CloseWrite(); err != nil {
					utils.Logger().Errorf("failed to close stdin for %v container %s: %v", svc, utils.ShortStr(ctr.ID), err)
				}
				att.Close()

				stopContainer(ctx)
				removeContainer(ctx)

				// give a brief moment for AutoRemove
				time.Sleep(200 * time.Millisecond)

				utils.Logger().Debugf("completed stop and detach sequence for %v container %s", svc, utils.ShortStr(ctr.ID))
			}

			utils.Logger().Infof("started %v (container %s)", svc, utils.ShortStr(ctr.ID))
			return detach, nil
		}
	}
}
