package supago

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/train360-corp/supago/internal/utils"
	"go.uber.org/zap"
	"regexp"
	"sync"
)

func IsValidPlatformName(platform string) bool {
	var platformNameRegex = regexp.MustCompile(`^[A-Za-z](?:[A-Za-z-]*[A-Za-z])?$`)
	return platformNameRegex.MatchString(platform)
}

type SupaGo struct {
	services []*Service
	logger   *zap.SugaredLogger
	config   Config
	mu       sync.Mutex
	docker   *client.Client
	network  *network.Summary
}

func New(config Config) *SupaGo {
	return &SupaGo{
		config:   config,
		logger:   zap.NewNop().Sugar(),
		services: []*Service{},
	}
}

func (sg *SupaGo) SetLogger(logger *zap.SugaredLogger) *SupaGo {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	sg.logger = logger
	return sg
}

func (sg *SupaGo) AddServices(services AllServices) *SupaGo {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	if sg.services == nil {
		sg.services = []*Service{}
	}
	for _, service := range *services(sg.config) {
		sg.services = append(sg.services, service)
	}
	return sg
}

func (sg *SupaGo) AddService(service *Service, services ...*Service) *SupaGo {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if sg.services == nil {
		sg.services = []*Service{}
	}
	sg.services = append(
		append(sg.services, service),
		services...,
	)
	return sg
}

// Run start and serve all services attached to the SupaGo instance
func (sg *SupaGo) Run(ctx context.Context) error {
	return sg.run(ctx, false)
}

// RunForcefully like Run, but will remove any conflicting containers destructively
func (sg *SupaGo) RunForcefully(ctx context.Context) error {
	return sg.run(ctx, true)
}

func (sg *SupaGo) Stop() {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	sg.logger.Warn("stop sequence initiated")

	// stop each service
	for i := range sg.services {
		func(service *Service) {
			if service.closeConn != nil {
				service.closeConn()
			}
			sg.stopContainer(service)
			if !sg.config.Global.DebugMode {
				sg.removeContainer(service)
			}
		}(sg.services[len(sg.services)-i-1]) // in reverse (for dependencies)
	}
}

func (sg *SupaGo) run(ctx context.Context, forcefully bool) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	// connect to docker
	if err := sg.setupDocker(); err != nil {
		e := fmt.Sprintf("failed to setup docker connection: %v", err)
		err := fmt.Errorf(e)
		sg.logger.Error(e)
		return err
	}

	// ensure network exists
	if err := sg.ensureNetwork(ctx); err != nil {
		e := fmt.Sprintf("failed to setup docker network: %v", err)
		err := fmt.Errorf(e)
		sg.logger.Error(e)
		return err
	}

	// start each service
	for _, service := range sg.services {

		// pull image
		if err := sg.pullImage(ctx, service); err != nil {
			e := fmt.Sprintf("failed to pull image: %v", err)
			err := fmt.Errorf(e)
			sg.logger.Error(e)
			return err
		}

		if forcefully { // always pre-attempt to remove container when forceful
			sg.removeContainerByName(ctx, service.Name)
		}

		// create container
		if ctr, err := sg.createContainer(ctx, service); err != nil {
			return err
		} else if ctr == nil {
			e := fmt.Sprintf("failed to create container: %v", "container unexpectedly nil")
			err := fmt.Errorf(e)
			sg.logger.Error(e)
			return err
		} else {
			service.container = ctr
		}

		// start container
		if err := sg.startContainer(ctx, service); err != nil {
			e := fmt.Sprintf("failed to start container for %v: %v", service, err)
			err := fmt.Errorf(e)
			sg.logger.Error(e)
			return err
		}

		// listen to container status
		sg.logger.Debugf("listening to %v container %s status", service, utils.ShortStr(service.container.ID))
		statusCh, errCh := sg.docker.ContainerWait(context.Background(), service.container.ID, container.WaitConditionNotRunning)
		go func() {
			select {
			case err := <-errCh:
				if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					sg.logger.Errorf("wait error for %v container %s: %v", service, utils.ShortStr(service.container.ID), err)
				} else {
					sg.logger.Debugf("wait exited (context-cancelled) for %v container %s", service, utils.ShortStr(service.container.ID))
				}
			case st := <-statusCh: // container exited (possibly immediately)
				sg.logger.Warnf("%v container %s exited with status: %v", service, utils.ShortStr(service.container.ID), st.StatusCode)
			}
		}()

		// healthcheck
		if err := sg.healthcheckContainer(ctx, service, 0); err != nil {
			e := fmt.Sprintf("failed to healthcheck container for %v: %v", service, err)
			err := fmt.Errorf(e)
			sg.logger.Error(e)
			return err
		}

		// AfterStart
		if service.AfterStart != nil {
			sg.logger.Debugf("running AfterStart for %v", service)
			if err := service.AfterStart(ctx, sg.docker, service.container.ID); err != nil {
				e := fmt.Sprintf("AfterStart failed for %v: %v", service, err)
				err := fmt.Errorf(e)
				sg.logger.Error(e)
				return err
			}
		}

		// attach to container (keep this connection open while app is alive)
		if att, err := sg.docker.ContainerAttach(context.Background(), service.container.ID, container.AttachOptions{
			Stdin:  true,
			Stream: true,
			Stdout: sg.config.Global.DebugMode,
			Stderr: sg.config.Global.DebugMode,
		}); err != nil {
			sg.logger.Errorf("attach failed for %v container %s: %v", service, utils.ShortStr(service.container.ID), err)
		} else {
			service.closeConn = att.Close
			sg.logger.Infof("%v started (container %s)", service, utils.ShortStr(service.container.ID))
		}
	}

	sg.logger.Info("all services started")

	return nil
}
