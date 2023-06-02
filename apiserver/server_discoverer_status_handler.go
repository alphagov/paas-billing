package apiserver

import (
	"github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/labstack/echo/v4"
	"net/http"
)

func DiscovererStatusHandler(discoverer instancediscoverer.CFAppDiscoverer) echo.HandlerFunc {
	return func(c echo.Context) error {
		success := true
		status := http.StatusOK
		err := discoverer.Ping()

		if err != nil {
			success = false
			status = http.StatusInternalServerError
		}

		return c.JSONPretty(status, map[string]bool{
			"ok": success,
		}, "  ")
	}
}
