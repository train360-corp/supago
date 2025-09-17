package main

import (
	"context"
	"github.com/train360-corp/supago"
	"go.uber.org/zap/zapcore"
	"os/signal"
	"syscall"
)

// encryptionKey is used for SupaBase Vault (must be persisted between restarts)
// Should not be inlined like below, and should be stored/retrieved more securely; consider:
// - supago.EncryptionKeyFromFile (generates a secret from a filepath and reads therefrom)
var encryptionKey = supago.StaticEncryptionKey("d9bf2393c65c006cc83625f85a27cc50882a391b1e0ab4fd4c2535dbe1f8a283")

// use any zap logger of choice, customized for specific use-case, or even disable logging altogether:
// var logger = zap.NewNop().Sugar() // No-Op / non-operational logger
var logger = supago.NewOpinionatedLogger(zapcore.InfoLevel, false)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Infof("SupaGo starting")

	// build a config
	cfg := supago.ConfigBuilder().
		Platform("example-project").
		GetEncryptionKeyUsing(encryptionKey).
		Build()

	// create a new SupaGo instance
	// this example uses all (supported) services; alternatively, add individual services as needed; e.g.:
	// AddService(supago.Services.Postgres, supago.Services.Kong)
	sg := supago.New(cfg).
		SetLogger(logger).
		AddServices(supago.Services.All)

	// run services
	if err := sg.RunForcefully(ctx); err != nil {
		logger.Errorf("an error occured while running services: %v", err)
	}

	// TODO: run custom backend services, interact with supabase, etc.

	<-ctx.Done() // block until done

	logger.Warnf("stop-signal recieved")

	sg.Stop() // shutdown services
}
