package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/collector"
	collector_cf "github.com/alphagov/paas-billing/collector/cloudfoundry"
	collector_compose "github.com/alphagov/paas-billing/collector/compose"
	"github.com/alphagov/paas-billing/compose"
	"github.com/alphagov/paas-billing/reporter"
	"github.com/alphagov/paas-billing/schema"
	"github.com/alphagov/paas-billing/server"
	"github.com/alphagov/paas-billing/server/auth"
	"github.com/pkg/errors"
)

var (
	logger = createLogger()
)

func createLogger() lager.Logger {
	logger := lager.NewLogger("paas-billing")
	logLevel := lager.INFO
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		logLevel = lager.DEBUG
	}
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
}

func createCFClient() (cloudfoundry.Client, error) {
	config := cloudfoundry.CreateConfigFromEnv()
	return cloudfoundry.NewClient(config)
}

func createComposeClient() (compose.Client, error) {
	composeApiKey := os.Getenv("COMPOSE_API_KEY")
	if composeApiKey == "" {
		return nil, errors.New("you must define COMPOSE_API_KEY")
	}
	return compose.NewClient(composeApiKey)
}

func Main() error {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return errors.Wrap(err, "failed to connect to database")
	}

	configFile, err := getConfigFilepath()
	if err != nil {
		return errors.Wrap(err, "failed to locate configuration file")
	}

	schema, err := schema.NewFromConfig(db, configFile)
	if err != nil {
		return err
	}

	logger.Info("initializing")
	initStart := time.Now()
	if err := schema.Init(); err != nil {
		return errors.Wrap(err, "failed to initialise database schema")
	}
	logger.Info("initialized", lager.Data{
		"duration": time.Since(initStart).String(),
	})

	cfClient, clientErr := createCFClient()
	if clientErr != nil {
		return errors.Wrap(clientErr, "failed to connect to Cloud Foundry")
	}

	composeClient, err := createComposeClient()
	if err != nil {
		return err
	}

	collectorConfig, err := collector.CreateConfigFromEnv()
	if err != nil {
		return errors.Wrap(err, "configuration error")
	}

	uaaConfig, err := auth.CreateConfigFromEnv()
	if err != nil {
		return err
	}
	apiAuthenticator := &auth.UAA{uaaConfig}

	ctx, shutdown := context.WithCancel(context.Background())
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		<-signalChan
		shutdown()
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		logger.Info("started app usage event collector")
		defer logger.Info("stopped app usage event collector")
		defer wg.Done()
		defer shutdown()

		appUsageEventsCollector := collector.New(
			collectorConfig,
			logger,
			collector_cf.NewEventFetcher(
				db,
				cloudfoundry.NewAppUsageEventsAPI(cfClient, logger),
			),
		)
		appUsageEventsCollector.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		logger.Info("started service usage event collector")
		defer logger.Info("stopped service usage event collector")
		defer wg.Done()
		defer shutdown()

		serviceUsageEventsCollector := collector.New(
			collectorConfig,
			logger,
			collector_cf.NewEventFetcher(
				db,
				cloudfoundry.NewServiceUsageEventsAPI(cfClient, logger),
			),
		)
		serviceUsageEventsCollector.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		logger.Info("started compose events collector")
		defer logger.Info("stopped compose events collector")
		defer wg.Done()
		defer shutdown()

		composeEventsCollector := collector.New(
			collectorConfig,
			logger,
			collector_compose.NewEventFetcher(
				db,
				composeClient,
			),
		)
		composeEventsCollector.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		logger.Info("starting schema updater")
		defer logger.Info("stopped schema updater")
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Hour):
			}
			logger.Info("refreshing schema")
			if err := schema.Refresh(); err != nil {
				logger.Error("refresh", err)
			}
			logger.Info("refreshed schema")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer logger.Info("stopped api server")
		logger.Info("starting api server")
		client := reporter.New(db)
		s := server.New(server.Config{
			BillingClient: client,
			Authenticator: apiAuthenticator,
		})
		port := os.Getenv("PORT")
		if port == "" {
			port = "8881"
		}
		server.ListenAndServe(ctx, s, fmt.Sprintf(":%s", port))
	}()

	wg.Wait()
	return nil
}

func main() {
	if err := Main(); err != nil {
		logger.Error("main", err)
		os.Exit(1)
	}
	logger.Info("shutdown")
}

func appRootDir() string {
	p := os.Getenv("APP_ROOT")
	if p != "" {
		return p
	}
	pwd := os.Getenv("PWD")
	if pwd == "" {
		pwd, _ = os.Getwd()
	}
	return pwd
}

func getConfigFilepath() (string, error) {
	root := appRootDir()
	p := filepath.Join(root, "config.json")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return "", fmt.Errorf("%s does not exist", p)
	}
	return p, nil
}
