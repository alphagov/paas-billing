package eventserver

import (
	"net/http"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventserver/auth"
	"github.com/labstack/echo"
)

func UsageEventsHandler(store eventio.UsageEventReader, uaa auth.Authenticator) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestedOrgs := c.Request().URL.Query()["org_guid"]
		if ok, err := authorize(c, uaa, requestedOrgs); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, err)
		} else if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}
		// parse params
		filter := eventio.EventFilter{
			RangeStart: c.QueryParam("range_start"),
			RangeStop:  c.QueryParam("range_stop"),
			OrgGUIDs:   requestedOrgs,
		}
		if err := filter.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		// query the store
		rows, err := store.GetUsageEventRows(filter)
		if err != nil {
			return err
		}
		defer rows.Close()
		// stream response to client
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if _, err := c.Response().Write([]byte("[\n")); err != nil {
			return err
		}
		c.Response().Flush()
		next := rows.Next()
		delim := "\n"
		for next {
			b, err := rows.EventJSON()
			if err != nil {
				return err
			}
			if _, err := c.Response().Write(b); err != nil {
				return err
			}
			next = rows.Next()
			if next {
				delim = ",\n"
			} else {
				delim = "\n"
			}
			if _, err := c.Response().Write([]byte(delim)); err != nil {
				return err
			}
			c.Response().Flush()
		}
		if _, err := c.Response().Write([]byte("]\n")); err != nil {
			return err
		}
		c.Response().Flush()
		return rows.Err()
	}
}
