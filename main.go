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
	cfclient "github.com/cloudfoundry-community/go-cfclient"
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
	cfConfig := &cfclient.Config{
		ApiAddress:        os.Getenv("CF_API_ADDRESS"),
		Username:          os.Getenv("CF_USERNAME"),
		Password:          os.Getenv("CF_PASSWORD"),
		ClientID:          os.Getenv("CF_CLIENT_ID"),
		ClientSecret:      os.Getenv("CF_CLIENT_SECRET"),
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
	}

	return cloudfoundry.NewClientWrapper(cfConfig)
}

func main() {
	logger := createLogger()

	client, clientErr := createCFClient()
	if clientErr != nil {
		logger.Error("init", clientErr)
		os.Exit(1)
		return
	}

	collectorConfig := collector.Config{
		DefaultSchedule: os.Getenv("COLLECTOR_DEFAULT_SCHEDULE"),
		MinWaitTime:     os.Getenv("COLLECTOR_MIN_WAIT_TIME"),
		FetchLimit:      os.Getenv("COLLECTOR_FETCH_LIMIT"),
		RecordMinAge:    os.Getenv("COLLECTOR_RECORD_MIN_AGE"),
	}

	collector, err := collector.New(client, collectorConfig, logger)
	if err != nil {
		logger.Error("init", clientErr)
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
