package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
				var totalTaskEvents int
				var totalNonTaskEvents int
				var priceIncVatSum float64
				var priceExVatSum float64
				var taskEvent eventio.BillableEvent

				next := rows.Next()
				for next {
					b, err := rows.EventJSON()

					// Check if the resource type is "task"
					row, err := rows.Event()

					if row != nil && row.ResourceType != "" && row.ResourceType == "task" {
						// Increase the count of total task events
						totalTaskEvents++

						// Convert the price values to float and add them to the total sum
						priceInc, _ := strconv.ParseFloat(row.Price.IncVAT, 64)
						priceEx, _ := strconv.ParseFloat(row.Price.ExVAT, 64)

						priceIncVatSum += priceInc
						priceExVatSum += priceEx

						if totalTaskEvents == 1 {
							taskEvent = eventio.BillableEvent{
								EventGUID:           row.EventGUID,
								EventStart:          row.EventStart,
								EventStop:           row.EventStop,
								ResourceGUID:        row.ResourceGUID,
								ResourceName:        "Total Task Events",
								ResourceType:        "task",
								OrgGUID:             row.OrgGUID,
								OrgName:             row.OrgName,
								SpaceGUID:           row.SpaceGUID,
								SpaceName:           "All Spaces",
								PlanGUID:            row.PlanGUID,
								PlanName:            row.PlanName,
								QuotaDefinitionGUID: row.QuotaDefinitionGUID,
								Price: eventio.Price{
									Details: row.Price.Details,
								},
							}
						}
						totalNonTaskEvents++

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
				if totalTaskEvents != 0 {
					// send a delimiter if we've sent any events
					if totalNonTaskEvents != 0 {
						if _, err := c.Response().Write([]byte(",\n")); err != nil {
							return err
						}
					}

					// Now create an aggregate event with the total task events and price sum
					taskEvent.Price.IncVAT = fmt.Sprintf("%.2f", priceIncVatSum)
					taskEvent.Price.ExVAT = fmt.Sprintf("%.2f", priceExVatSum)

					// Marshal it into json
					taskEventJSON, err := json.Marshal(taskEvent)
					if err != nil {
						return err
					}
					// Send it if there are any task events
					if _, err := c.Response().Write(taskEventJSON); err != nil {
						return err
					}
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
