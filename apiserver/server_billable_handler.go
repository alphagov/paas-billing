package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
			err := func() error { // so we can use defer in-loop
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

				// Assume rows is a slice of event data
				taskEvents := make(map[string]*eventio.BillableEvent)

				next := rows.Next()
				for next {
					b, err := rows.EventJSON()

					// Check if the resource type is "task"
					row, err := rows.Event()

					if row != nil && row.ResourceType == "task" {
						// Set the key as a combination of Org GUID and Space GUID
						key := fmt.Sprintf("%s-%s", row.OrgGUID, row.SpaceGUID)

						// Convert the price values to float
						priceInc, _ := strconv.ParseFloat(row.Price.IncVAT, 64)
						priceEx, _ := strconv.ParseFloat(row.Price.ExVAT, 64)

						event, exists := taskEvents[key]
						if !exists {
							const layout = "2006-01-02"

							rangeStart, _ := time.Parse(layout, c.QueryParam("range_start"))
							rangeStop, _ := time.Parse(layout, c.QueryParam("range_stop"))

							event = &eventio.BillableEvent{
								EventGUID:           row.EventGUID,
								EventStart:          rangeStart.Format("2006-01-02T00:00:00+00:00"),
								EventStop:           rangeStop.Format("2006-01-02T00:00:00+00:00"),
								ResourceGUID:        row.ResourceGUID,
								ResourceName:        "Total Task Events",
								ResourceType:        "task",
								OrgGUID:             row.OrgGUID,
								OrgName:             row.OrgName,
								SpaceGUID:           row.SpaceGUID,
								SpaceName:           row.SpaceName,
								PlanGUID:            row.PlanGUID,
								PlanName:            row.PlanName,
								QuotaDefinitionGUID: row.QuotaDefinitionGUID,
								Price: eventio.Price{
									Details: []eventio.PriceComponent{{
										Name:         "All tasks aggregated",
										PlanName:     "tasks",
										Start:        rangeStart.Format("2006-01-02T00:00:00+00:00"),
										Stop:         rangeStop.Format("2006-01-02T00:00:00+00:00"),
										CurrencyCode: "USD",
										VatRate:      "0.2",
									}},
									FloatIncVAT: priceInc,
									FloatExVAT:  priceEx,
								},
							}
							taskEvents[key] = event
						} else {
							// Add this priceInc to event.Price.IncVAT
							event.Price.FloatIncVAT = event.Price.FloatIncVAT + priceInc
							event.Price.FloatExVAT = event.Price.FloatExVAT + priceEx
						}

						// Skip the event as we will group them all into one event at the end
						next = rows.Next()
						continue
					}

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
				// loop over each task event and send it
				for _, event := range taskEvents {
					event.Price.IncVAT = fmt.Sprintf("%.16f", event.Price.FloatIncVAT)
					event.Price.ExVAT = fmt.Sprintf("%.16f", event.Price.FloatExVAT)
					event.Price.Details[0].IncVAT = fmt.Sprintf("%.16f", event.Price.FloatIncVAT)
					event.Price.Details[0].ExVAT = fmt.Sprintf("%.16f", event.Price.FloatExVAT)
					b, err := json.Marshal(event)
					if err != nil {
						return err
					}
					// send the delimiter
					if _, err := c.Response().Write([]byte(",\n")); err != nil {
						return err
					}
					if _, err := c.Response().Write(b); err != nil {
						return err
					}
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
