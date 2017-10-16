package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
)

func createLogger() lager.Logger {
	logger := lager.NewLogger("metrics")
	logLevel := lager.INFO
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		logLevel = lager.DEBUG
	}
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
}

// Main is the main function
func Main() error {
	logger := createLogger()

	cfClient, cfClientErr := cfclient.NewClient(&cfclient.Config{
		ApiAddress:        os.Getenv("CF_API_ADDRESS"),
		Username:          os.Getenv("CF_USERNAME"),
		Password:          os.Getenv("CF_PASSWORD"),
		ClientID:          os.Getenv("CF_CLIENT_ID"),
		ClientSecret:      os.Getenv("CF_CLIENT_SECRET"),
		SkipSslValidation: os.Getenv("CF_SKIP_SSL_VALIDATION") == "true",
	})
	if cfClientErr != nil {
		return errors.Wrap(cfClientErr, "failed to connect to Cloud Foundry API")
	}

	c := cloudfoundry.NewClient(cfClient, logger)

	events, err := c.GetAppUsageEvents(cloudfoundry.GUIDNil, 20, 5*time.Minute)
	if err != nil {
		return err
	}

	fmt.Println(events)

	return nil
}

func main() {
	if err := Main(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("shutdown gracefully")
}
