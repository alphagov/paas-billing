package apiserver

import (
	"net/http"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/labstack/echo"
)

func VATRatesHandler(store eventio.VATRateReader) echo.HandlerFunc {
	return func(c echo.Context) error {
		filter := eventio.TimeRangeFilter{
			RangeStart: c.QueryParam("range_start"),
			RangeStop:  c.QueryParam("range_stop"),
		}
		vatRates, err := store.GetVATRates(filter)
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		return c.JSON(http.StatusOK, vatRates)
	}
}
