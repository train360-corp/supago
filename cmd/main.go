package cmd

import (
	"context"
	"fmt"
	"github.com/train360-corp/supago/pkg/services"
	"github.com/train360-corp/supago/pkg/services/supabase"
	"github.com/train360-corp/supago/pkg/utils"
	"go.uber.org/zap/zapcore"
	"os"
	"os/signal"
	"path/filepath"
	"time"
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

	// Create a root ctx that is canceled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if config, err := supabase.GetRandomConfig(); err != nil {
		panic(fmt.Sprintf("failed to generate supabase config: %v", err))
	} else {
		config.DatabaseDataDirectory = filepath.Join(filepath.Dir(cwd), "data", "postgres")
		if svcs, err := supabase.GetServices(config); err != nil {
			panic(err)
		} else {
			runner, err := services.NewRunner("supago-main-test-net")
			if err != nil {
				panic(err)
			}

			var stops []func(context.Context)
			for _, service := range *svcs {
				stop, err := runner.Run(context.Background(), &service)
				if err != nil {
					utils.Logger().Fatal(err)
				}
				stops = append(stops, stop)
			}

			// Block until we get a signal.
			<-ctx.Done()

			utils.Logger().Warn("commencing shutdown")

			// Give ongoing work time to finish.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// stop in reverse order for dependency purposes
			for i := len(stops) - 1; i >= 0; i-- {
				stops[i](shutdownCtx)
			}

			utils.Logger().Warn("shutdown complete")
		}
	}
}
