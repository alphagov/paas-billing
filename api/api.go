package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/labstack/echo"
)

type UsageParams struct {
	From string
	To   string
}

func ParseUsageParams(r *http.Request) (*UsageParams, error) {
	q := r.URL.Query()
	params := &UsageParams{
		From: q.Get("from"),
		To:   q.Get("to"),
	}
	if params.From == "" {
		epoch := &time.Time{}
		params.From = epoch.UTC().Format(time.RFC3339)
	} else {
		if _, err := time.Parse(time.RFC3339, params.From); err != nil {
			return nil, err
		}
	}
	if params.To == "" {
		params.To = time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	} else {
		if _, err := time.Parse(time.RFC3339, params.To); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func NewUsageHandler(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		return respond(Many, c, db, `
			select
				guid,
				org_guid,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				name,
				memory_in_mb,
				iso8601(lower(duration)) as start,
				iso8601(upper(duration)) as stop,
				price::bigint as price
			from
				billable_range(tstzrange($1, $2))
			order by
				guid, id
		`, params.From, params.To)
	}
}

func NewReportHandler(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		return respond(Single, c, db, `
		with
			resources as (
				select
					guid,
					org_guid,
					space_guid,
					pricing_plan_id,
					pricing_plan_name,
					sum(to_seconds(duration)) as duration,
					sum(price)::bigint as price
				from
					billable_range(tstzrange($1, $2))
				group by
					guid, space_guid, org_guid, pricing_plan_id, pricing_plan_name
			),
			space_resources as (
				select
					r.org_guid,
					r.space_guid,
					(select sum(t.price) from resources t where t.space_guid = r.space_guid) as price,
					(select json_agg(row_to_json(t.*)) from resources t where t.space_guid = r.space_guid) as resources
				from
					resources r
				group by
					org_guid, space_guid
			),
			org_resources as (
				select
					s.org_guid,
					(select sum(t.price) from resources t where t.org_guid = s.org_guid) as price,
					(select json_agg(row_to_json(t.*)) from space_resources t where t.org_guid = s.org_guid) as spaces
				from
					space_resources s
				group by
					org_guid
			)
			select
				(select sum(t.price) from resources t) as price,
				(select json_agg(row_to_json(t.*)) from org_resources t) as orgs
			from
				org_resources
			limit 1
		`, params.From, params.To)
	}
}

func ListOrgUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		return respond(Many, c, db, `
			select
				org_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			group by
				org_guid
			order by
				org_guid
		`, params.From, params.To)
	}
}

func GetOrgUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return respond(Single, c, db, `
			select
				org_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				org_guid = $3
			group by
				org_guid
			limit 1
		`, params.From, params.To, orgGUID)
	}
}

func ListSpacesUsageForOrg(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return respond(Many, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				org_guid = $3
			group by
				org_guid, space_guid
			order by
				space_guid
		`, params.From, params.To, orgGUID)
	}
}

func ListSpacesUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		return respond(Many, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			group by
				org_guid, space_guid
			order by
				space_guid
		`, params.From, params.To)
	}
}

func GetSpaceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		spaceGUID := c.Param("space_guid")
		if spaceGUID == "" {
			return errors.New("missing space_guid")
		}
		return respond(Single, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				space_guid = $3
			group by
				org_guid, space_guid
			limit 1
		`, params.From, params.To, spaceGUID)
	}
}

func ListResourceUsageForOrg(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return respond(Many, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				org_guid = $3
			group by
				org_guid, space_guid, guid
			order by
				guid
		`, params.From, params.To, orgGUID)
	}
}

func ListResourceUsageForSpace(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		spaceGUID := c.Param("space_guid")
		if spaceGUID == "" {
			return errors.New("missing space_guid")
		}
		return respond(Many, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				space_guid = $3
			group by
				org_guid, space_guid, guid
			order by
				guid
		`, params.From, params.To, spaceGUID)
	}
}

func ListResourceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		return respond(Many, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			group by
				space_guid, org_guid, guid
			order by
				guid
		`, params.From, params.To)
	}
}

func GetResourceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		resourceGUID := c.Param("resource_guid")
		if resourceGUID == "" {
			return errors.New("missing resource_guid")
		}
		return respond(Single, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				guid = $3
			group by
				org_guid, space_guid, guid
			order by
				guid
		`, params.From, params.To, resourceGUID)
	}
}

func ListEventUsageForResource(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		resourceGUID := c.Param("resource_guid")
		if resourceGUID == "" {
			return errors.New("missing resource_guid")
		}
		return respond(Many, c, db, `
			select
				guid,
				org_guid,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				iso8601(lower(duration)) as from,
				iso8601(upper(duration)) as to,
				(price * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			where
				guid = $3
			order by
				id, pricing_plan_id, guid
		`, params.From, params.To, resourceGUID)
	}
}

func ListEventUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		params, err := ParseUsageParams(c.Request())
		if err != nil {
			return err
		}
		return respond(Many, c, db, `
			select
				guid,
				org_guid,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				iso8601(lower(duration)) as from,
				iso8601(upper(duration)) as to,
				(price * 100)::bigint as price_in_pence
			from
				billable_range(tstzrange($1, $2))
			order by
				id, pricing_plan_id, guid
		`, params.From, params.To)
	}
}

type resourceType int

const (
	Invalid resourceType = iota
	Single
	Many
)

func respond(rt resourceType, c echo.Context, db db.SQLClient, sql string, args ...interface{}) error {
	acceptHeader := c.Request().Header.Get(echo.HeaderAccept)
	accepts := strings.Split(acceptHeader, ",")
	for _, accept := range accepts {
		c.Logger().Debug("accepts", accepts)
		if accept == echo.MIMEApplicationJSON || accept == echo.MIMEApplicationJSONCharsetUTF8 {
			c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			if rt == Single {
				return db.QueryRowJSON(c.Response(), sql, args...)
			} else if rt == Many {
				return db.QueryJSON(c.Response(), sql, args...)
			}
		} else if accept == echo.MIMETextHTML || accept == echo.MIMETextHTMLCharsetUTF8 {
			return c.HTML(http.StatusOK, "TODO: respond to html type")

		}
	}
	return c.HTML(http.StatusNotAcceptable, "unacceptable")
}
