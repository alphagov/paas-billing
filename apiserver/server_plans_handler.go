package apiserver

import (
	"net/http"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/labstack/echo"
)

func PricingPlansHandler(store eventio.PricingPlanReader) echo.HandlerFunc {
	return func(c echo.Context) error {
		filter := eventio.PricingPlanFilter{
			RangeStart: c.QueryParam("range_start"),
			RangeStop:  c.QueryParam("range_stop"),
		}
		plans, err := store.GetPricingPlans(filter)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		return c.JSON(http.StatusOK, plans)
	}
}
