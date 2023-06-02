package apiserver

import (
	"fmt"
	"github.com/alphagov/paas-billing/instancediscoverer"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type MetricsTarget struct {
	Targets []string      `json:"targets"`
	Labels  MetricsLabels `json:"labels"`
}

type MetricsLabels struct {
	MetricsPath     string `json:"__metrics_path__"`
	OrgName         string `json:"__meta_target_orgName"`
	SpaceName       string `json:"__meta_target_spaceName"`
	ApplicationName string `json:"__meta_target_applicationName"`
	ApplicationId   string `json:"__meta_target_applicationID"`
	InstanceNumber  string `json:"__meta_target_instanceNumber"`
	InstanceID      string `json:"__meta_target_instanceId"`
}

func MetricsDiscoveryHandler(discoverer instancediscoverer.CFAppDiscoverer) echo.HandlerFunc {
	return func(c echo.Context) error {
		var targets []MetricsTarget
		appName := c.Param("appName")
		app, err := discoverer.GetSpaceAppByName(appName)
		if err != nil {
			if cfclient.IsAppNotFoundError(err) || err == instancediscoverer.AccessDeniedError {
				return c.JSON(http.StatusNotFound, []string{})
			}
			return c.JSON(http.StatusInternalServerError, []string{})
		}

		for i := 0; i < app.Instances; i++ {
			target := MetricsTarget{
				Targets: []string{c.Request().Host},
				Labels: MetricsLabels{
					MetricsPath:     fmt.Sprintf("/proxymetrics/%s/%d", app.Name, i),
					OrgName:         discoverer.Org().Name,
					SpaceName:       discoverer.Space().Name,
					ApplicationName: app.Name,
					ApplicationId:   app.Guid,
					InstanceNumber:  strconv.Itoa(i),
					InstanceID:      fmt.Sprintf("%s:%d", app.Guid, i),
				},
			}
			targets = append(targets, target)
		}
		return c.JSONPretty(http.StatusOK, targets, "  ")
	}
}
