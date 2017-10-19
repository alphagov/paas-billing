package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	"github.com/alphagov/paas-usage-events-collector/collector"
	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/pkg/errors"
)

func createLogger() lager.Logger {
	logger := lager.NewLogger("usage-events-collector")
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

func main() {
	logger := createLogger()

	sqlClient := db.NewPostgresClient(os.Getenv("DATABASE_URL"))

	if err := sqlClient.InitSchema(); err != nil {
		logger.Error("init", errors.Wrap(err, "failed to initialise database schema"))
		os.Exit(1)
		return
	}

	client, clientErr := createCFClient()
	if clientErr != nil {
		logger.Error("init", errors.Wrap(clientErr, "failed to connect to Cloud Foundry"))
		os.Exit(1)
		return
	}

	collectorConfig := collector.CreateConfigFromEnv()

	collector, collectorErr := collector.New(
		cloudfoundry.NewAppUsageEventsAPI(client, logger),
		cloudfoundry.NewServiceUsageEventsAPI(client, logger),
		sqlClient,
		collectorConfig,
		logger,
	)

	if collectorErr != nil {
		logger.Error("init", errors.Wrap(collectorErr, "failed to initialise collector"))
		os.Exit(1)
		return
	}

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGINT)
	defer signal.Reset(syscall.SIGINT)

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		<-signalChan
		cancelFunc()
	}()

	collector.Run(ctx)
}
