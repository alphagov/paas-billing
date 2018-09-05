package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/alphagov/paas-billing/cfstore"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventcollector"
	"github.com/alphagov/paas-billing/eventfetchers/cffetcher"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventserver"
	"github.com/alphagov/paas-billing/eventserver/auth"
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

func (app *App) StartEventServer() error {
	name := "api"
	logger := app.logger.Session(name)
	uaaConfig, err := auth.CreateConfigFromEnv()
	if err != nil {
		return err
	}
	apiAuthenticator := &auth.UAA{
		Config: uaaConfig,
	}
	apiServer := eventserver.New(eventserver.Config{
		Store:         app.store,
		Authenticator: apiAuthenticator,
		Logger:        logger,
	})
	addr := fmt.Sprintf(":%d", app.cfg.ServerPort)
	return app.start(name, logger, func() error {
		return eventserver.ListenAndServe(
			app.ctx,
			logger,
			apiServer,
			addr,
		)
	})
}

func (app *App) StartEventProcessor() error {
	name := "processor"
	logger := app.logger.Session(name)
	return app.start(name, logger, func() error {
		logger.Info("started")
		defer logger.Info("stopping")
		for {
			select {
			case <-app.ctx.Done():
				return nil
			case <-time.After(app.cfg.Processor.Schedule):
				logger.Info("processing")
				if err := app.store.Refresh(); err != nil {
					logger.Error("refresh-error", err)
					continue
				}
			}
			logger.Info("processed", lager.Data{
				"next_processing_in": app.cfg.Processor.Schedule.String(),
			})
		}
	})
}

func (app *App) StartHistoricDataCollector() error {
	name := "historic-data-collector"
	logger := app.logger.Session(name)
	go func() {
		for {
			if err := app.historicDataStore.CollectServices(); err != nil {
				logger.Error("collect-services", err)
				continue
			}
			if err := app.historicDataStore.CollectServicePlans(); err != nil {
				logger.Error("collect-service-plans", err)
				continue
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

	if cfg.Store == nil {
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("Store or DatabaseURL must be provided in Config")
		}
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to database")
		}
		planConfigFile, err := cfg.ConfigFile()
		if err != nil {
			return nil, err
		}
		store, err := eventstore.NewFromConfig(ctx, db, cfg.Logger.Session("store"), planConfigFile)
		if err != nil {
			return nil, err
		}
		cfg.Store = store
	}

	client, err := cfclient.NewClient(cfg.HistoricDataCollector.ClientConfig)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
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
