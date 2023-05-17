package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/alphagov/paas-billing/metricsproxy"

	"github.com/alphagov/paas-billing/cfstore"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/apiserver"
	"github.com/alphagov/paas-billing/apiserver/auth"
	"github.com/alphagov/paas-billing/eventcollector"
	"github.com/alphagov/paas-billing/eventfetchers/cffetcher"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
)

type App struct {
	wg                sync.WaitGroup
	ctx               context.Context
	store             eventio.EventStore
	historicDataStore *cfstore.Store
	logger            lager.Logger
	cfg               Config
	Shutdown          context.CancelFunc
}

func (app *App) Init() error {
	if err := app.store.Init(); err != nil {
		return err
	}
	if err := app.historicDataStore.Init(); err != nil {
		return err
	}
	return nil
}

func (app *App) StartAppEventCollector() error {
	return app.startUsageEventCollector(cffetcher.App)
}

func (app *App) StartServiceEventCollector() error {
	return app.startUsageEventCollector(cffetcher.Service)
}

func (app *App) startUsageEventCollector(kind cffetcher.Kind) error {
	name := fmt.Sprintf("%s-usage-event-collector", kind)
	logger := app.logger.Session(name)
	fetcher, err := cffetcher.New(cffetcher.Config{
		Logger:       logger,
		Type:         kind,
		ClientConfig: app.cfg.CFFetcher.ClientConfig,
		FetchLimit:   app.cfg.CFFetcher.FetchLimit,
		RecordMinAge: app.cfg.CFFetcher.RecordMinAge,
	})
	if err != nil {
		return err
	}
	collector := eventcollector.New(eventcollector.Config{
		Logger:      logger,
		Store:       app.store,
		Fetcher:     fetcher,
		Schedule:    app.cfg.Collector.Schedule,
		MinWaitTime: app.cfg.Collector.MinWaitTime,
	})
	return app.start(name, logger, func() error {
		return collector.Run(app.ctx)
	})
}

func (app *App) StartAPIServer() error {
	name := "api"
	logger := app.logger.Session(name)
	uaaConfig, err := auth.CreateConfigFromEnv()
	if err != nil {
		return err
	}
	apiAuthenticator := &auth.UAA{
		Config: uaaConfig,
	}
	apiServer := apiserver.New(apiserver.Config{
		Store:         app.store,
		Authenticator: apiAuthenticator,
		Logger:        logger,
	})
	return app.start(name, logger, func() error {
		return apiserver.ListenAndServe(
			app.ctx,
			logger,
			apiServer,
			app.cfg.ListenAddr,
		)
	})
}

func (app *App) StartHealthServer() error {
	name := "health"
	logger := app.logger.Session(name)
	healthServer := apiserver.NewBaseServer(apiserver.Config{
		Store:  app.store,
		Logger: logger,
	})
	return app.start(name, logger, func() error {
		return apiserver.ListenAndServe(
			app.ctx,
			logger,
			healthServer,
			app.cfg.ListenAddr,
		)
	})
}

func (app *App) StartProxyMetricsServer() error {
	name := "proxymetrics"
	logger := app.logger.Session(name)
	appDiscoverer, err := instancediscoverer.New(instancediscoverer.Config{
		ClientConfig:   app.cfg.InstanceDiscoverer.ClientConfig,
		DiscoveryScope: app.cfg.InstanceDiscoverer.DiscoveryScope,
		Logger:         logger,
		ThisAppName:    app.cfg.VCAPApplication.ApplicationName,
	})
	if err != nil {
		return err
	}

	metricsProxy := metricsproxy.New(metricsproxy.Config{
		Logger: logger,
	})

	proxyMetricsServer := apiserver.NewProxyMetrics(apiserver.Config{
		Logger: logger,
	}, appDiscoverer, metricsProxy)

	return app.start(name, logger, func() error {
		return apiserver.ListenAndServe(
			app.ctx,
			logger,
			proxyMetricsServer,
			app.cfg.ListenAddr,
		)
	})
}

func (app *App) StartEventProcessor() error {
	name := "processor"
	logger := app.logger.Session(name)
	return app.start(name, logger, func() error {
		runRefreshAndConsolidateLoop(app.ctx, logger, app.cfg.Processor.Schedule, app.store)
		runPeriodicMetricsLoop(app.ctx, logger, app.cfg.Processor.PeriodicMetricsSchedule, app.store)
		return nil
	})
}

func runPeriodicMetricsLoop(ctx context.Context, logger lager.Logger, schedule time.Duration, store eventio.EventStore) {
	for {
		if err := store.RecordPeriodicMetrics(); err != nil {
			logger.Error("periodic-metrics-error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(schedule):
		}
	}
}

func runRefreshAndConsolidateLoop(ctx context.Context, logger lager.Logger, schedule time.Duration, store eventio.EventStore) {
	logger.Info("started")
	defer logger.Info("stopping")
	for {
		logger.Info("processing")
		if err := store.Refresh(); err != nil {
			logger.Error("refresh-error", err)
		} else if err := store.ConsolidateAll(); err != nil {
			logger.Error("consolidate-error", err)
		} else {
			logger.Info("processed", lager.Data{
				"next_processing_in": schedule.String(),
			})
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(schedule):
		}
	}
}

func (app *App) StartHistoricDataCollector() error {
	name := "historic-data-collector"
	logger := app.logger.Session(name)
	go func() {
		for {
			if err := app.historicDataStore.CollectServices(); err != nil {
				logger.Error("collect-services", err)
			}
			if err := app.historicDataStore.CollectServicePlans(); err != nil {
				logger.Error("collect-service-plans", err)
			}
			if err := app.historicDataStore.CollectOrgs(); err != nil {
				logger.Error("collect-orgs", err)
			}
			if err := app.historicDataStore.CollectSpaces(); err != nil {
				logger.Error("collect-spaces", err)
			}

			time.Sleep(app.cfg.HistoricDataCollector.Schedule)
		}
	}()
	return nil
}

func (app *App) start(name string, logger lager.Logger, fn func() error) error {
	app.wg.Add(1)
	go func() {
		logger.Info("starting")
		defer logger.Info("stopped")
		defer app.wg.Done()
		defer app.Shutdown()
		if err := fn(); err != nil {
			logger.Error("stop-with-error", err)
		}
	}()
	return nil
}

func (app *App) Wait() error {
	app.wg.Wait()
	return nil
}

func New(ctx context.Context, cfg Config) (*App, error) {
	ctx, shutdown := context.WithCancel(ctx)

	go func() {
		defer shutdown()
		<-ctx.Done()
		cfg.Logger.Info("stopping")
	}()

	if cfg.Logger == nil {
		cfg.Logger = lager.NewLogger("app")
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("Store or DatabaseURL must be provided in Config")
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}
	db.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)

	planConfigFile, err := cfg.ConfigFile()
	if err != nil {
		return nil, err
	}
	store, err := eventstore.NewFromConfig(ctx, db, cfg.Logger.Session("store"), planConfigFile)
	if err != nil {
		return nil, err
	}
	cfg.Store = store

	client, err := cfclient.NewClient(cfg.HistoricDataCollector.ClientConfig)
	if err != nil {
		return nil, err
	}

	historicDataStore, err := cfstore.New(cfstore.Config{
		Client: &cfstore.Client{Client: client},
		DB:     db,
		Logger: cfg.Logger.Session("historic-data-store"),
	})
	if err != nil {
		return nil, err
	}

	app := &App{
		cfg:               cfg,
		ctx:               ctx,
		Shutdown:          shutdown,
		store:             cfg.Store,
		historicDataStore: historicDataStore,
		logger:            cfg.Logger,
	}

	return app, nil
}
