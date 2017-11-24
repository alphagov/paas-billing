package server

import (
	"context"
	"crypto/subtle"
	"os"
	"time"

	"github.com/alphagov/paas-usage-events-collector/api"
	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
)

var (
	BASIC_PASSWORD = os.Getenv("BASIC_PASSWORD")
)

func New(db db.SQLClient) *echo.Echo {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	e.Use(middleware.Recover())

	if BASIC_PASSWORD != "" {
		e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
			usernameCorrectness := subtle.ConstantTimeCompare([]byte(username), []byte("admin")) == 1
			passwordCorrectness := subtle.ConstantTimeCompare([]byte(password), []byte(BASIC_PASSWORD)) == 1
			if usernameCorrectness && passwordCorrectness {
				return true, nil
			}
			return false, nil
		}))
	}

	e.GET("/usage", api.NewUsageHandler(db))   // FIXME: this is redundent
	e.GET("/report", api.NewReportHandler(db)) // FIXME: this should be an endpoint for fetching complete bills

	// Usage data API
	e.GET("/organisations", api.ListOrgUsage(db))
	e.GET("/organisations/:org_guid", api.GetOrgUsage(db))
	e.GET("/organisations/:org_guid/spaces", api.ListSpacesUsageForOrg(db))
	e.GET("/organisations/:org_guid/resources", api.ListResourceUsageForOrg(db))
	e.GET("/spaces", api.ListSpacesUsage(db))
	e.GET("/spaces/:space_guid", api.GetSpaceUsage(db))
	e.GET("/spaces/:space_guid/resources", api.ListResourceUsageForSpace(db))
	e.GET("/resources", api.ListResourceUsage(db))
	e.GET("/resources/:resource_guid", api.GetResourceUsage(db))
	e.GET("/events", api.ListEventUsage(db))
	e.GET("/resources/:resource_guid/events", api.ListEventUsageForResource(db))

	// Pricing data API
	e.GET("/pricing_plans", api.ListPricingPlans(db))
	e.GET("/pricing_plans/:pricing_plan_id", api.GetPricingPlan(db))
	e.POST("/pricing_plans", api.CreatePricingPlan(db))
	e.PUT("/pricing_plans/:pricing_plan_id", api.UpdatePricingPlan(db))
	e.DELETE("/pricing_plans/:pricing_plan_id", api.DestroyPricingPlan(db))

	return e
}

func ListenAndServe(ctx context.Context, e *echo.Echo, addr string) {

	// Start server
	go func() {
		if err := e.Start(addr); err != nil {
			e.Logger.Info("shutting down the server", err)
		} else {
			e.Logger.Info("shutting down the server")
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
