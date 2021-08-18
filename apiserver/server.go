package apiserver

import (
	"context"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-billing/apiserver/auth"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type Config struct {
	// Authenticator sets the auth mechanism (required)
	Authenticator auth.Authenticator
	// Store sets the Store used for querying events (required)
	Store eventio.EventStore
	// Logger sets the request logger
	Logger lager.Logger
	// EnablePanic will cause the server to crash on panic if set to true
	EnablePanic bool
	// Version - 1 or 2. 1 will calculate bills in the app, 2 will call the database functions to do so.
	Version int
}

// New creates a new server. Use ListenAndServe to start accepting connections.
func New(cfg Config) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = errorHandler

	if !cfg.EnablePanic {
		e.Use(middleware.Recover())
	}

	if cfg.Logger != nil {
		echoCompatibleLogger := NewLogger(cfg.Logger)
		e.Logger = echoCompatibleLogger
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Output: echoCompatibleLogger,
		}))
	}

	e.GET("/vat_rates", VATRatesHandler(cfg.Store))
	e.GET("/currency_rates", CurrencyRatesHandler(cfg.Store))
	e.GET("/pricing_plans", PricingPlansHandler(cfg.Store))
	e.GET("/forecast_events", ForecastEventsHandler(cfg.Store)) // TODO: can we handle this in new billing?
	e.GET("/usage_events", UsageEventsHandler(cfg.Store, cfg.Authenticator))
	e.GET("/billable_events", BillableEventsHandler(cfg.Store, cfg.Store, cfg.Authenticator)) // TODO - version difference
	e.GET("/totals", TotalCostHandler(cfg.Store)) // TODO - version difference

	e.GET("/", status)

	return e
}

func status(c echo.Context) error {
	return c.JSONPretty(http.StatusOK, map[string]bool{
		"ok": true,
	}, "  ")
}

func ListenAndServe(ctx context.Context, logger lager.Logger, e *echo.Echo, addr string) error {

	ctx, shutdown := context.WithCancel(ctx)

	go func() {
		defer shutdown()
		logger.Info("started", lager.Data{
			"addr": addr,
		})
		if err := e.Start(addr); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				e.Logger.Error("listen-and-serve-error", err)
			}
		}
	}()

	// Wait for parent context to get cancelled then drain with a 10s timeout
	<-ctx.Done()
	e.Logger.Info("stopping")
	drainCtx, cancelDrain := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelDrain()
	return e.Shutdown(drainCtx)
}
