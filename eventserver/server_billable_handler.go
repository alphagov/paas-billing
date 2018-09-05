package eventserver

import (
	"net/http"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventserver/auth"
	"github.com/labstack/echo"
	"io"
)

func BillableEventsHandler(store eventio.BillableEventReader, consolidatedStore eventio.ConsolidatedBillableEventReader, uaa auth.Authenticator) echo.HandlerFunc {
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

		var rows eventio.BillableEventRows
		ok, err := consolidatedStore.IsRangeConsolidated(filter)
		if err != nil {
			return err
		}
		if ok {
			rows, err = consolidatedStore.GetConsolidatedBillableEventRows(filter)
		} else {
			rows, err = store.GetBillableEventRows(filter)
		}
		if err != nil {
			return err
		}
		defer rows.Close()

		// stream response to client
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return WriteRowsAsJson(c.Response(), c.Response(), rows)
	}
}

func WriteRowsAsJson(writer io.Writer, flusher http.Flusher, rows eventio.BillableEventRows) error {
	if _, err := writer.Write([]byte("[\n")); err != nil {
		return err
	}
	flusher.Flush()
	next := rows.Next()
	delim := "\n"
	for next {
		b, err := rows.EventJSON()
		if err != nil {
			return err
		}
		if _, err := writer.Write(b); err != nil {
			return err
		}
		next = rows.Next()
		if next {
			delim = ",\n"
		} else {
			delim = "\n"
		}
		if _, err := writer.Write([]byte(delim)); err != nil {
			return err
		}
		flusher.Flush()
	}
	if _, err := writer.Write([]byte("]\n")); err != nil {
		return err
	}
	flusher.Flush()
	return rows.Err()
}
