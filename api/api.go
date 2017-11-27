package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alphagov/paas-usage-events-collector/auth"
	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/labstack/echo"
)

type UsageParams struct {
	From time.Time
	To   time.Time
}

func ParseUsageParams(r *http.Request) (*UsageParams, error) {
	q := r.URL.Query()
	params := &UsageParams{
		To: time.Now().UTC().Add(24 * time.Hour),
	}
	from := q.Get("from")
	if from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return nil, err
		}
		params.From = t
	}
	to := q.Get("to")
	if to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return nil, err
		}
		params.To = t
	}
	return params, nil
}

func NewUsageHandler(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, c, db, `
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
				authorized_resources
			order by
				guid, id
		`)
	}
}

func NewReportHandler(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Single, c, db, `
			with
			resources as (
				select
					guid,
					org_guid,
					space_guid,
					pricing_plan_id,
					pricing_plan_name,
					sum(to_seconds(duration)) as duration,
					sum(price * 100)::bigint as price
				from
					authorized_resources
				group by
					guid, space_guid, org_guid, pricing_plan_id, pricing_plan_name
				order by
					guid, space_guid, org_guid, pricing_plan_id
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
				order by
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
				order by
					org_guid
			)
			select
				(select sum(t.price) from resources t) as price,
				(select json_agg(row_to_json(t.*)) from org_resources t) as orgs
			from
				org_resources
			limit 1
		`)
	}
}

func ListOrgUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, c, db, `
			select
				org_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			group by
				org_guid
			order by
				org_guid
		`)
	}
}

func GetOrgUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return withAuthorizedResources(Single, c, db, `
			select
				org_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			where
				org_guid = $1
			group by
				org_guid
			limit 1
		`, orgGUID)
	}
}

func ListSpacesUsageForOrg(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return withAuthorizedResources(Many, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			where
				org_guid = $1
			group by
				org_guid, space_guid
			order by
				space_guid
		`, orgGUID)
	}
}

func ListSpacesUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			group by
				org_guid, space_guid
			order by
				space_guid
		`)
	}
}

func GetSpaceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		spaceGUID := c.Param("space_guid")
		if spaceGUID == "" {
			return errors.New("missing space_guid")
		}
		return withAuthorizedResources(Single, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			where
				space_guid = $1
			group by
				org_guid, space_guid
			limit 1
		`, spaceGUID)
	}
}

func ListResourceUsageForOrg(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return withAuthorizedResources(Many, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			where
				org_guid = $1
			group by
				org_guid, space_guid, guid
			order by
				guid
		`, orgGUID)
	}
}

func ListResourceUsageForSpace(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		spaceGUID := c.Param("space_guid")
		if spaceGUID == "" {
			return errors.New("missing space_guid")
		}
		return withAuthorizedResources(Many, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			where
				space_guid = $1
			group by
				org_guid, space_guid, guid
			order by
				guid
		`, spaceGUID)
	}
}

func ListResourceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			group by
				space_guid, org_guid, guid
			order by
				guid
		`)
	}
}

func GetResourceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		resourceGUID := c.Param("resource_guid")
		if resourceGUID == "" {
			return errors.New("missing resource_guid")
		}
		return withAuthorizedResources(Single, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price) * 100)::bigint as price_in_pence
			from
				authorized_resources
			where
				guid = $1
			group by
				org_guid, space_guid, guid
			order by
				guid
		`, resourceGUID)
	}
}

func ListEventUsageForResource(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		resourceGUID := c.Param("resource_guid")
		if resourceGUID == "" {
			return errors.New("missing resource_guid")
		}
		return withAuthorizedResources(Many, c, db, `
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
				authorized_resources
			where
				guid = $1
			order by
				id, pricing_plan_id, guid
		`, resourceGUID)
	}
}

func ListEventUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, c, db, `
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
				authorized_resources
			order by
				id, pricing_plan_id, guid
		`)
	}
}

type resourceType int

const (
	Invalid resourceType = iota
	Single
	Many
)

func authorizedSpaceFilter(c echo.Context, args []interface{}) (string, []interface{}, error) {
	authorized, ok := c.Get("authorizer").(auth.Authorizer)
	if !ok {
		return "", args, errors.New("unauthorized: no authorizer in context")
	}
	spaces, err := authorized.Spaces()
	if err != nil {
		return "", args, err
	}
	if len(spaces) < 1 {
		return "", args, errors.New("unauthorized: you are not authorized to view any space usage data")
	}
	conditions := make([]string, len(spaces))
	for i, guid := range spaces {
		args = append(args, guid)
		conditions[i] = fmt.Sprintf("space_guid = $%d", len(args))
	}
	return strings.Join(conditions, " or "), args, nil
}

func withAuthorizedResources(rt resourceType, c echo.Context, db db.SQLClient, sql string, args ...interface{}) error {
	params, err := ParseUsageParams(c.Request())
	if err != nil {
		return err
	}
	spaceFilter, args, err := authorizedSpaceFilter(c, args)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf(`
		with authorized_resources as (
			select *
			from billable_range(tstzrange('%s', '%s'))
			where %s
		),
		q as (
			%s
		)
		select * from q
	`, params.From.Format(time.RFC3339), params.To.Format(time.RFC3339), spaceFilter, sql)
	return render(rt, c, db, sql, args...)
}

func render(rt resourceType, c echo.Context, db db.SQLClient, sql string, args ...interface{}) error {
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
