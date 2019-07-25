package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.cloudfoundry.org/lager"
)

func Main(ctx context.Context, logger lager.Logger) error {
	cfg, err := NewConfigFromEnv()
	if err != nil {
		return err
	}
	cfg.Logger = logger

	app, err := New(ctx, cfg)
	if err != nil {
		return err
	}

	if len(os.Args) < 2 {
		return errors.New("Please provide a command to run [api | collector]")
	}
	switch command := os.Args[1]; command {
	case "collector":
		return startCollector(app, cfg)
	case "api":
		return startAPI(app, cfg)
	default:
		return fmt.Errorf("Subcommand %s not recognised", command)
	}
}

func startCollector(app *App, cfg Config) error {
	if err := app.Init(); err != nil {
		return err
	}
	if err := app.StartAppEventCollector(); err != nil {
		return err
	}

	if err := app.StartServiceEventCollector(); err != nil {
		return err
	}

	if err := app.StartEventProcessor(); err != nil {
		return err
	}
	if err := app.StartHistoricDataCollector(); err != nil {
		return err
	}

	cfg.Logger.Info("started collector")
	return app.Wait()
}

func startAPI(app *App, cfg Config) error {
	if err := app.StartAPIServer(); err != nil {
		return err
	}
	cfg.Logger.Info("started API")
	return app.Wait()
}

func main() {
	ctx, shutdown := context.WithCancel(context.Background())

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		shutdown()
	}()

	logger := getDefaultLogger()
	logger.Info("starting")
	defer logger.Info("stopped")
	if err := Main(ctx, logger); err != nil {
		logger.Error("exit-error", err)
		os.Exit(1)
	}
}
