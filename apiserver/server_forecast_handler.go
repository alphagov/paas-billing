package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	"github.com/labstack/echo/v4"
)

func ForecastEventsHandler(store eventio.BillableEventForecaster) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestedOrgGUIDs := c.Request().URL.Query()["org_guid"]
		for _, guid := range requestedOrgGUIDs {
			if guid != eventstore.DummyOrgGUID {
				return echo.NewHTTPError(http.StatusForbidden, fmt.Errorf("you are not authorized to forecast events for org '%s'", guid))
			}
		}
		// parse params
		filter := eventio.EventFilter{
			RangeStart: c.QueryParam("range_start"),
			RangeStop:  c.QueryParam("range_stop"),
			OrgGUIDs:   []string{eventstore.DummyOrgGUID},
		}
		if err := filter.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		inputEventData := c.QueryParam("events")
		if inputEventData == "" {
			return echo.NewHTTPError(http.StatusBadRequest, errors.New("events param is required"))
		}
		var inputEvents []eventio.UsageEvent
		if err := json.Unmarshal([]byte(inputEventData), &inputEvents); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		storeCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// query the store
		rows, err := store.ForecastBillableEventRows(storeCtx, inputEvents, filter)
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
