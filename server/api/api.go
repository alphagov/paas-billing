package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alphagov/paas-billing/reporter"
	"github.com/alphagov/paas-billing/server/auth"
	"github.com/labstack/echo"
)

func UsageEventsHandler(billingClient *reporter.Reporter, uaa auth.Authenticator) echo.HandlerFunc {
	return func(c echo.Context) error {
		return fmt.Errorf("not imp")
	}
}

func BillableEventsHandler(billingClient *reporter.Reporter, uaa auth.Authenticator) echo.HandlerFunc {
	return func(c echo.Context) error {
		// check if token has an operator scope (cloud_controler.admin / global_auditor)
		token, err := auth.GetTokenFromRequest(c)
		if err != nil {
			return unauthorized(c, err)
		}
		authorizer, err := uaa.NewAuthorizer(token)
		if err != nil {
			return unauthorized(c, err)
		}
		isAdmin, err := authorizer.Admin()
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		if !isAdmin {
			return echo.NewHTTPError(http.StatusUnauthorized, "billing data currently requires cloud_controller.admin or cloud_controller.global_auditor scope")
		}
		// parse params
		filter := reporter.EventFilter{
			RangeStart: c.QueryParam("range_start"),
			RangeStop:  c.QueryParam("range_stop"),
			OrgGUIDs:   c.Request().URL.Query()["org_guid"],
		}
		if err := filter.Validate(); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		// query the data and stream rows back
		rows, err := billingClient.GetBillableEventRows(filter)
		if err != nil {
			return err
		}
		defer rows.Close()
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)
		for rows.Next() {
			b, err := rows.EventJSON()
			if err != nil {
				return err
			}
			if _, err := c.Response().Write(b); err != nil {
				return err
			}
			if _, err := c.Response().Write([]byte("\n")); err != nil {
				return err
			}
			c.Response().Flush()
		}
		return rows.Err()
	}
}

func UsageForcastHandler(r *reporter.Reporter) echo.HandlerFunc {
	return func(c echo.Context) error {
		return fmt.Errorf("notimp")
		// filter := reporter.EventFilter{
		// 	RangeStart: c.QueryParam("range_start"),
		// 	RangeStop:  c.QueryParam("range_stop"),
		// 	OrgGUIDs:   c.Request().URL.Query()["org_guid"],
		// }
		// rows, err := r.GetBillableEventRows(filter)
		// if err != nil {
		// 	return err
		// }
		// defer rows.Close()
		// c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		// c.Response().WriteHeader(http.StatusOK)
		// for rows.Next() {
		// 	b, err := rows.EventJSON()
		// 	if err != nil {
		// 		return err
		// 	}
		// 	if _, err := c.Response().Write(b); err != nil {
		// 		return err
		// 	}
		// 	if _, err := c.Response().Write([]byte("\n")); err != nil {
		// 		return err
		// 	}
		// 	c.Response().Flush()
		// }
		// return rows.Err()
	}
}

func unauthorized(c echo.Context, err error) error {
	acceptHeader := c.Request().Header.Get(echo.HeaderAccept)
	accepts := strings.Split(acceptHeader, ",")
	for _, accept := range accepts {
		if accept == echo.MIMETextHTML || accept == echo.MIMETextHTMLCharsetUTF8 {
			return c.Redirect(http.StatusFound, "/oauth/authorize")
		}
	}
	return echo.NewHTTPError(http.StatusUnauthorized, err)
}
