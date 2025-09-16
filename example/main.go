package main

import (
	"context"
	"github.com/train360-corp/supago"
	"go.uber.org/zap/zapcore"
	"os/signal"
	"syscall"
)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := supago.NewOpinionatedLogger(zapcore.InfoLevel, false)
	logger.Infof("SupaGo starting")

	sg := supago.New(*supago.NewRandomConfig("example-project")).
		SetLogger(logger).
		AddServices(supago.Services.All())

	if err := sg.RunForcefully(ctx); err != nil {
		logger.Errorf("an error occured while running services: %v", err)
	}

	// TODO: run custom backend services, interact with supabase, etc.

	<-ctx.Done() // block until done

	logger.Warnf("stop-signal recieved")

	sg.Stop() // shutdown services
}
