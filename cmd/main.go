package cmd

import (
	"context"
	"fmt"
	"github.com/train360-corp/supago/pkg/services"
	"github.com/train360-corp/supago/pkg/supabase"
	"github.com/train360-corp/supago/pkg/utils"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"path/filepath"
)

func Execute() {

	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %v", err))
	}

	// setup custom logger
	if logger, err := utils.NewLogger(zapcore.InfoLevel, false); err != nil {
		panic(err)
	} else {
		utils.OverrideLogger(logger)
	}
	defer utils.Logger().Sync()

	// Create a root ctx that is canceled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// create config to run supabase
	config, err := supabase.GetRandomConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to generate supabase config: %v", err))
	}
	config.DatabaseDataDirectory = filepath.Join(cwd, "data", "postgres")
	config.StorageDirectory = filepath.Join(cwd, "data", "storage")

	// get supabase services from config
	svcs, err := supabase.GetServices(config)
	if err != nil {
		panic(err)
	}

	// run supabase services
	runner, err := services.NewRunner("supago-main-test-net")
	if err != nil {
		panic(err)
	}
	for _, service := range *svcs {
		if err := runner.Run(ctx, &service); err != nil {
			runner.Shutdown() // cancel running services
			utils.Logger().Fatal(err)
		}
	}
	utils.Logger().Infof("all services started")

	<-ctx.Done() // block until we get a stop signal

	utils.Logger().Warn("shutdown signal received")

	// shutdown each utility
	runner.Shutdown()

	utils.Logger().Debugf("main context done")
}
