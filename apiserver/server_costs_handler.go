package apiserver

import (
	"net/http"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/labstack/echo/v4"
)

func TotalCostHandler(store eventio.TotalCostReader) echo.HandlerFunc {
	return func(c echo.Context) error {
		costTotals, err := store.GetTotalCost()
		if err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		return c.JSON(http.StatusOK, costTotals)
	}
}
