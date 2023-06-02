package apiserver

import (
	"fmt"
	"github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/alphagov/paas-billing/metricsproxy"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

func MetricsProxyHandler(proxy metricsproxy.MetricsProxy, discoverer instancediscoverer.CFAppDiscoverer) echo.HandlerFunc {
	return func(c echo.Context) error {
		appName := c.Param("appName")
		app, err := discoverer.GetSpaceAppByName(appName)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusNotFound, "app not found")
		}

		rawAppInstanceID := c.Param("appInstanceID")
		appInstanceID, err := strconv.Atoi(rawAppInstanceID)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest, fmt.Errorf("appInstanceID (%s) must be an integer", rawAppInstanceID))
		}

		urls, err := discoverer.GetAppRouteURLsByName(appName)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError, "could not get urls for app")
		}
		if len(urls) == 0 {
			return echo.NewHTTPError(
				http.StatusNotFound, "there were no urls for app")
		}

		appUrl := urls[0]
		appUrl.Path = "/metrics"

		headers := map[string]string{"X-Cf-App-Instance": fmt.Sprintf("%s:%d", app.Guid, appInstanceID)}

		proxy.ForwardRequestToURL(c.Response(), c.Request(), appUrl, headers)
		return nil
	}
}
