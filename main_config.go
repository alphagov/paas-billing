package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alphagov/paas-billing/eventcollector"
	"github.com/alphagov/paas-billing/eventfetchers/cffetcher"
	"github.com/alphagov/paas-billing/eventfetchers/composefetcher"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"

	"code.cloudfoundry.org/lager"
)

type Config struct {
	Logger         lager.Logger
	Store          eventio.EventStore
	DatabaseURL    string
	Collector      eventcollector.Config
	CFFetcher      cffetcher.Config
	ComposeFetcher composefetcher.Config
	ServerPort     int
	Processor      ProcessorConfig
}

type ProcessorConfig struct {
	Schedule time.Duration
}

func NewFromEnv(ctx context.Context, logger lager.Logger) (*App, error) {
	planConfigFile, err := getConfigFilepath()
	if err != nil {
		return nil, err
	}

	connstr := getEnvString("DATABASE_URL")
	db, err := sql.Open("postgres", connstr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	store, err := eventstore.NewFromConfig(ctx, db, logger.Session("store"), planConfigFile)
	if err != nil {
		return nil, err
	}

	cfg := Config{
		Logger: logger,
		Store:  store,
		Collector: eventcollector.Config{
			Schedule:    getEnvWithDefaultDuration("COLLECTOR_DEFAULT_SCHEDULE", 15*time.Minute),
			MinWaitTime: getEnvWithDefaultDuration("COLLECTOR_MIN_WAIT_TIME", 3*time.Second),
		},
		CFFetcher: cffetcher.Config{
			ClientConfig: &cfclient.Config{
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
			},
			RecordMinAge: getEnvWithDefaultDuration("COLLECTOR_RECORD_MIN_AGE", 10*time.Minute),
			FetchLimit:   getEnvWithDefaultInt("COLLECTOR_FETCH_LIMIT", 50),
		},
		ComposeFetcher: composefetcher.Config{
			APIKey:     getEnvString("COMPOSE_API_KEY"),
			FetchLimit: getEnvWithDefaultInt("COLLECTOR_FETCH_LIMIT", 100),
		},
		Processor: ProcessorConfig{
			Schedule: getEnvWithDefaultDuration("PROCESSOR_SCHEDULE", 30*time.Minute),
		},
		ServerPort: getEnvWithDefaultInt("PORT", 8881),
	}
	return New(ctx, cfg)
}

func getEnvWithDefaultDuration(k string, def time.Duration) time.Duration {
	v := getEnvWithDefaultString(k, "")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(err)
	}
	return d
}

func getEnvWithDefaultInt(k string, def int) int {
	v := getEnvWithDefaultString(k, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(err)
	}
	return n
}

func getEnvWithDefaultString(k string, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func getEnvString(k string) string {
	v := getEnvWithDefaultString(k, "")
	if v == "" {
		panic(fmt.Sprintf("environment variable %s is required", k))
	}
	return v
}

func getDefaultLogger() lager.Logger {
	logger := lager.NewLogger("paas-billing")
	logLevel := lager.INFO
	if strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug" {
		logLevel = lager.DEBUG
	}
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger
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
