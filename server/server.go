package server

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-billing/reporter"
	"github.com/alphagov/paas-billing/server/api"
	"github.com/alphagov/paas-billing/server/auth"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/lib/pq"
)

type Config struct {
	Authenticator auth.Authenticator
	BillingClient *reporter.Reporter
	Logger        lager.Logger
}

func New(cfg Config) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	e.HTTPErrorHandler = errorHandler

	if cfg.Logger != nil {
		e.Logger = NewLogger(cfg.Logger)
	}

	// e.GET("/forecast_events", api.ForecastEventsHandler(reporter))
	// e.GET("/usage_events", api.UsageEventsHandler(cfg.BillingClient, cfg.Authenticator))
	e.GET("/billable_events", api.BillableEventsHandler(cfg.BillingClient, cfg.Authenticator))

	e.GET("/", listRoutes)

	return e
}

type ErrorResponse struct {
	Error      string `json:"error"`
	Constraint string `json:"constraint,omitempty"`
}

func errorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	resp := ErrorResponse{
		Error: "internal server error",
	}

	switch v := err.(type) {
	case *echo.HTTPError:
		code = v.Code
		resp.Error = fmt.Sprintf("%s", v.Message)
	case *pq.Error:
		if v.Code.Name() == "check_violation" {
			code = http.StatusBadRequest
			resp.Error = "constraint violation"
			resp.Constraint = v.Constraint
		}
	}

	c.Logger().Error(err)
	if err := c.JSON(code, resp); err != nil {
		c.Logger().Error(err)
	}
}

func listRoutes(c echo.Context) error {
	routes := c.Echo().Routes()
	sort.Slice(routes, func(i, j int) bool { return routes[i].Path < routes[j].Path })

	return c.JSONPretty(http.StatusOK, routes, "  ")
}

func ListenAndServe(ctx context.Context, e *echo.Echo, addr string) {

	// Start server
	go func() {
		if err := e.Start(addr); err != nil {
			e.Logger.Info(err)
		}
	}()

	// Wait for parent context to get cancelled then drain with a 10s timeout
	<-ctx.Done()
	drainCtx, cancelDrain := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelDrain()
	if err := e.Shutdown(drainCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
