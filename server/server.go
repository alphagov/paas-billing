package server

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/alphagov/paas-billing/api"
	"github.com/alphagov/paas-billing/auth"
	"github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/db"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
)

func New(db db.SQLClient, authority auth.Authenticator, cf cloudfoundry.Client) *echo.Echo {
	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	// Allow HTML forms to override POST method using _method param
	e.Pre(middleware.MethodOverrideWithConfig(middleware.MethodOverrideConfig{
		Getter: middleware.MethodFromForm("_method"),
	}))

	// Never crash on panic
	e.Use(middleware.Recover())

	// Require a token for all requests
	e.Use(auth.UAATokenAuthentication(authority))

	// Validate and parse range query param
	e.Use(api.ValidateRangeParams)

	// Deprecated endpoint, favor /resources and /events
	e.GET("/usage", api.NewUsageHandler(db))

	e.GET("/report/:org_guid", api.NewReportHandler(db))

	// Usage and Billing API
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
	e.POST("/pricing_plans", auth.AdminOnly(api.CreatePricingPlan(db)))
	e.PUT("/pricing_plans/:pricing_plan_id", auth.AdminOnly(api.UpdatePricingPlan(db)))
	e.DELETE("/pricing_plans/:pricing_plan_id", auth.AdminOnly(api.DestroyPricingPlan(db)))
	e.POST("/seed_pricing_plans", auth.AdminOnly(api.CreateMissingPricingPlans(db)))

	e.GET("/", listRoutes)

	return e
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
