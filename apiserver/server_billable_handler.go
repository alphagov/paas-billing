package apiserver

import (
	"context"
	"net/http"

	"github.com/alphagov/paas-billing/apiserver/auth"
	"github.com/alphagov/paas-billing/eventio"
	"github.com/labstack/echo/v4"
)

func BillableEventsHandler(store eventio.BillableEventReader, consolidatedStore eventio.ConsolidatedBillableEventReader, uaa auth.Authenticator) echo.HandlerFunc {
	return func(c echo.Context) error {
		sentOKHeader := false
		sendOKHeader := func() error {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			c.Response().WriteHeader(http.StatusOK)
			if _, err := c.Response().Write([]byte("[\n")); err != nil {
				return err
			}
			c.Response().Flush()
			sentOKHeader = true
			return nil
		}

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

		months, err := filter.SplitByMonth()
		if err != nil {
			return err
		}

		delim := ""
		for _, monthFilter := range months {
			err := func () error {  // so we can use defer in-loop
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
				defer rows.Close()

				next := rows.Next()
				for next {
					b, err := rows.EventJSON()
					if err != nil {
						return err
					}
					if !sentOKHeader {
						// this is sent as late as possible because any errors encountered after
						// this won't be communicated to the client correctly
						if err := sendOKHeader(); err != nil {
							return err
						}
					}
					if _, err := c.Response().Write([]byte(delim)); err != nil {
						return err
					}
					if _, err := c.Response().Write(b); err != nil {
						return err
					}
					delim = ",\n"
					next = rows.Next()
					c.Response().Flush()
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
		if !sentOKHeader {
			if err := sendOKHeader(); err != nil {
				return err
			}
		}
		if _, err := c.Response().Write([]byte("\n]\n")); err != nil {
			return err
		}
		return nil
	}
}
