package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"code.cloudfoundry.org/lager"
)

var globalContext context.Context

func init() {
	ctx, shutdown := context.WithCancel(context.Background())
	globalContext = ctx
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		shutdown()
	}()
}

func Main(logger lager.Logger) error {
	app, err := NewFromEnv(globalContext, logger)
	if err != nil {
		return err
	}

	if err := app.Init(); err != nil {
		return err
	}

	if err := app.StartAppEventCollector(); err != nil {
		return err
	}

	if err := app.StartServiceEventCollector(); err != nil {
		return err
	}

	if err := app.StartComposeEventCollector(); err != nil {
		return err
	}

	if err := app.StartEventProcessor(); err != nil {
		return err
	}

	if err := app.StartEventServer(); err != nil {
		return err
	}

	logger.Info("started")
	return app.Wait()
}

func main() {
	logger := getDefaultLogger()
	logger.Info("starting")
	defer logger.Info("stopped")
	if err := Main(logger); err != nil {
		logger.Error("exit-error", err)
		os.Exit(1)
	}
}
