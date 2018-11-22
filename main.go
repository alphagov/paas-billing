package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alphagov/paas-billing/cfstore"
	cfclient "github.com/cloudfoundry-community/go-cfclient"

	"code.cloudfoundry.org/lager"
)

func cfDataCollector(databaseUrl string, logger lager.Logger) error {
	client, err := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        os.Getenv("CF_API_ADDRESS"),
		Username:          os.Getenv("CF_USERNAME"),
		Password:          os.Getenv("CF_PASSWORD"),
		ClientID:          os.Getenv("CF_CLIENT_ID"),
		ClientSecret:      os.Getenv("CF_CLIENT_SECRET"),
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
		Token:             os.Getenv("CF_TOKEN"),
		UserAgent:         os.Getenv("CF_USER_AGENT"),
		HttpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	})
	if err != nil {
		return err
	}
	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		return err
	}
	cfHistoricData, err := cfstore.New(cfstore.Config{
		Client: &cfstore.Client{Client: client},
		DB:     db,
	})
	if err != nil {
		return err
	}
	if err := cfHistoricData.Init(); err != nil {
		return err
	}
	go func() {
		for {
			if err := cfHistoricData.CollectServices(); err != nil {
				logger.Error("collect-services", err)
				continue
			}
			if err := cfHistoricData.CollectServicePlans(); err != nil {
				logger.Error("collect-service-plans", err)
				continue
			}
			if err := cfHistoricData.CollectOrgs(); err != nil {
				logger.Error("collect-orgs", err)
				continue
			}
			if err := cfHistoricData.CollectSpaces(); err != nil {
				logger.Error("collect-spaces", err)
				continue
			}
			time.Sleep(10 * time.Second)
		}
	}()
	return nil
}

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

	cfDataCollector(cfg.DatabaseURL, cfg.Logger)

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
	if err := app.StartEventServer(); err != nil {
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
