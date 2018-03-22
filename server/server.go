package server

import (
	"context"
	"fmt"
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
	"github.com/lib/pq"
)

func New(db db.SQLClient, authority auth.Authenticator, cf cloudfoundry.Client) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = errorHandler
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

	e.POST("/forecast/report", api.NewSimulatedReportHandler(db))

	authGroup.GET("/report/:org_guid", api.NewOrgReportHandler(db))

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
	authGroup.GET("/events/raw", api.ListEventUsageRaw(db)) // TODO: this is a temporary endpoint
	authGroup.GET("/resources/:resource_guid/events", api.ListEventUsageForResource(db))

	// Pricing data API
	authGroup.GET("/pricing_plans", api.ListPricingPlans(db))
	authGroup.GET("/pricing_plans/:pricing_plan_id", api.GetPricingPlan(db))
	authGroup.GET("/pricing_plans/:pricing_plan_id/components", api.ListPricingPlanComponentsByPlan(db))
	adminGroup.POST("/pricing_plans", api.CreatePricingPlan(db))
	adminGroup.PUT("/pricing_plans/:pricing_plan_id", api.UpdatePricingPlan(db))
	adminGroup.DELETE("/pricing_plans/:pricing_plan_id", api.DestroyPricingPlan(db))
	adminGroup.POST("/seed_pricing_plans", api.CreateMissingPricingPlans(db))
	authGroup.GET("/pricing_plan_components", api.ListPricingPlanComponents(db))
	authGroup.GET("/pricing_plan_components/:id", api.GetPricingPlanComponent(db))
	adminGroup.POST("/pricing_plan_components", api.CreatePricingPlanComponent(db))
	adminGroup.PUT("/pricing_plan_components/:id", api.UpdatePricingPlanComponent(db))
	adminGroup.DELETE("/pricing_plan_components/:id", api.DestroyPricingPlanComponent(db))
	authGroup.GET("/vat_rates", api.ListVATRates(db))
	authGroup.GET("/vat_rates/:id", api.GetVATRate(db))
	adminGroup.POST("/vat_rates", api.CreateVATRate(db))
	adminGroup.PUT("/vat_rates/:id", api.UpdateVATRate(db))
	adminGroup.DELETE("/vat_rates/:id", api.DestroyVATRate(db))

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
