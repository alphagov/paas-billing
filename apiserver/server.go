package apiserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	prom_client "github.com/prometheus/client_golang/prometheus"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/alphagov/paas-billing/metricsproxy"
	"github.com/labstack/echo-contrib/prometheus"

	"github.com/alphagov/paas-billing/apiserver/auth"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lestrrat-go/jwx/jwk"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

type Config struct {
	// Authenticator sets the auth mechanism (required)
	Authenticator auth.Authenticator
	// Store sets the Store used for querying events (required)
	Store eventio.EventStore
	// Logger sets the request logger
	Logger lager.Logger
	// EnablePanic will cause the server to crash on panic if set to true
	EnablePanic bool

	UAATokenKeysUrl string
}

// CacheHeaders sets the cache headers to prevent caching
func CacheHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("Pragma", "no-cache")
		c.Response().Header().Set("Expires", "0")
		return next(c)
	}
}

// New creates a base new server. Use ListenAndServe to start accepting connections.
// It will only serve the status page
func NewBaseServer(cfg Config) *echo.Echo {
	e := echo.New()
	e.Use(CacheHeaders)
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

	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	e.GET("/", EventStoreStatusHandler(cfg.Store))

	return e
}

func addGroup(e *echo.Echo, groupName string, handlerFunc echo.HandlerFunc, cfg Config) {
	r := e.Group(groupName)
	config := echojwt.Config{
		KeyFunc: func(token *jwt.Token) (interface{}, error) {
			return getKey(cfg.UAATokenKeysUrl, token)
		},
	}
	r.Use(echojwt.WithConfig(config))
	r.GET("", handlerFunc)
}

// New creates a new server. Use ListenAndServe to start accepting connections.
// Serves api functions
func New(cfg Config) *echo.Echo {

	e := NewBaseServer(cfg)

	e.GET("/vat_rates", VATRatesHandler(cfg.Store))
	e.GET("/currency_rates", CurrencyRatesHandler(cfg.Store))
	e.GET("/pricing_plans", PricingPlansHandler(cfg.Store))
	e.GET("/forecast_events", ForecastEventsHandler(cfg.Store))
	addGroup(e, "/usage_events", UsageEventsHandler(cfg.Store, cfg.Authenticator), cfg)
	addGroup(e, "/billable_events", BillableEventsHandler(cfg.Store, cfg.Store, cfg.Authenticator), cfg)
	e.GET("/totals", TotalCostHandler(cfg.Store))

	return e
}

func getKey(uaaUrl string, token *jwt.Token) (interface{}, error) {

	keySet, err := jwk.Fetch(context.Background(), uaaUrl)
	if err != nil {
		return nil, err
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have a key ID in the kid field")
	}

	key, found := keySet.LookupKeyID(keyID)

	if !found {
		return nil, fmt.Errorf("unable to find key %q", keyID)
	}

	var pubkey interface{}
	if err := key.Raw(&pubkey); err != nil {
		return nil, fmt.Errorf("Unable to get the public key. Error: %s", err.Error())
	}

	return pubkey, nil
}

func NewProxyMetrics(cfg Config, discoverer instancediscoverer.CFAppDiscoverer, proxy metricsproxy.MetricsProxy) *echo.Echo {
	e := NewBaseServer(cfg)

	e.GET("/discovery/:appName", MetricsDiscoveryHandler(discoverer))
	e.GET("/proxymetrics/:appName/:appInstanceID", MetricsProxyHandler(proxy, discoverer))

	e.GET("/", DiscovererStatusHandler(discoverer))

	return e
}

func ListenAndServe(ctx context.Context, logger lager.Logger, e *echo.Echo, addr string) error {

	ctx, shutdown := context.WithCancel(ctx)

	go func() {
		defer shutdown()
		var _ = prom_client.DefaultRegisterer
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
