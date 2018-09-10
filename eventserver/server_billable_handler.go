package eventserver

import (
	"context"
	"fmt"
	"net/http"

	"io"

	"strings"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventserver/auth"
	"github.com/labstack/echo"
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

		storeCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// query the store
		rowOfRows := RowOfRows{}
		defer rowOfRows.Close()

		months, err := filter.SplitByMonth()
		if err != nil {
			return err
		}
		for _, monthFilter := range months {
			isConsolidated, err := consolidatedStore.IsRangeConsolidated(monthFilter)
			if err != nil {
				return err
			}
			var rows eventio.BillableEventRows
			if isConsolidated {
				rows, err = consolidatedStore.GetConsolidatedBillableEventRows(storeCtx, monthFilter)
			} else {
				rows, err = store.GetBillableEventRows(storeCtx, monthFilter)
			}
			if err != nil {
				return err
			}
			rowOfRows.RowsCollection = append(rowOfRows.RowsCollection, rows)
		}

		// stream response to client
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return WriteRowsAsJson(c.Response(), c.Response(), &rowOfRows)
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

type RowOfRows struct {
	RowsCollection []eventio.BillableEventRows
	index          int
}

func (r *RowOfRows) Next() bool {
	if r.index >= len(r.RowsCollection) {
		return false
	}
	if r.RowsCollection[r.index].Next() {
		return r.Err() == nil
	}
	r.index = r.index + 1
	return r.Next()
}

func (r *RowOfRows) Close() error {
	errs := []string{}
	for _, r := range r.RowsCollection {
		err := r.Close()
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("errors happened calling RowOfRows.Close(): %s", strings.Join(errs, " | "))
}

func (r *RowOfRows) Err() error {
	if r.index >= len(r.RowsCollection) {
		return nil
	}
	return r.RowsCollection[r.index].Err()
}

func (r *RowOfRows) EventJSON() ([]byte, error) {
	if r.index >= len(r.RowsCollection) {
		return nil, fmt.Errorf("no more data in RowsCollection")
	}
	return r.RowsCollection[r.index].EventJSON()
}

func (r *RowOfRows) Event() (*eventio.BillableEvent, error) {
	if r.index >= len(r.RowsCollection) {
		return nil, fmt.Errorf("no more data in RowsCollection")
	}
	return r.RowsCollection[r.index].Event()
}
