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

	// Validate and parse range query param
	e.Use(api.ValidateRangeParams)

	authGroup := e.Group("", auth.UAATokenAuthentication(authority))
	adminGroup := authGroup.Group("", auth.AdminOnly)

	// Deprecated endpoint, favor /resources and /events
	authGroup.GET("/usage", api.NewUsageHandler(db))

	authGroup.GET("/report/:org_guid", api.NewReportHandler(db))

	// Usage and Billing API
	authGroup.GET("/organisations", api.ListOrgUsage(db))
	authGroup.GET("/organisations/:org_guid", api.GetOrgUsage(db))
	authGroup.GET("/organisations/:org_guid/spaces", api.ListSpacesUsageForOrg(db))
	authGroup.GET("/organisations/:org_guid/resources", api.ListResourceUsageForOrg(db))
	authGroup.GET("/spaces", api.ListSpacesUsage(db))
	authGroup.GET("/spaces/:space_guid", api.GetSpaceUsage(db))
	authGroup.GET("/spaces/:space_guid/resources", api.ListResourceUsageForSpace(db))
	authGroup.GET("/resources", api.ListResourceUsage(db))
	authGroup.GET("/resources/:resource_guid", api.GetResourceUsage(db))
	authGroup.GET("/events", api.ListEventUsage(db))
	authGroup.GET("/resources/:resource_guid/events", api.ListEventUsageForResource(db))

	// Pricing data API
	authGroup.GET("/pricing_plans", api.ListPricingPlans(db))
	authGroup.GET("/pricing_plans/:pricing_plan_id", api.GetPricingPlan(db))
	adminGroup.POST("/pricing_plans", api.CreatePricingPlan(db))
	adminGroup.PUT("/pricing_plans/:pricing_plan_id", api.UpdatePricingPlan(db))
	adminGroup.DELETE("/pricing_plans/:pricing_plan_id", api.DestroyPricingPlan(db))
	adminGroup.POST("/seed_pricing_plans", api.CreateMissingPricingPlans(db))
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
