package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/alphagov/paas-usage-events-collector/auth"
	"github.com/alphagov/paas-usage-events-collector/db"
	"github.com/labstack/echo"
)

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
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return withAuthorizedResources(Many, c, db, `
			with
			resources as (
				select
					name,
					org_guid,
					space_guid,
					pricing_plan_id,
					pricing_plan_name,
					sum(to_seconds(duration)) as duration,
					sum(price * 100)::bigint as price
				from
					authorized_resources
				group by
					name, space_guid, org_guid, pricing_plan_id, pricing_plan_name
				order by
					name, space_guid, org_guid, pricing_plan_id
			),
			space_resources as (
				select
					t.org_guid,
					t.space_guid,
					sum(t.price) as price,
					json_agg(row_to_json(t.*)) as resources
				from
					resources t
				group by
					org_guid, space_guid
				order by
					org_guid, space_guid
			)
			select
				t.org_guid,
				sum(t.price) as price,
				json_agg(row_to_json(t.*)) as spaces
			from
				space_resources t
			where
				org_guid = $1
			group by
				org_guid
			order by
				org_guid
		`, orgGUID)
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

func authorizedSpaceFilter(authorizer auth.Authorizer, rng RangeParams, sql string, args []interface{}) (string, []interface{}, error) {
	cond := ""
	if !authorizer.Admin() {
		spaces, err := authorizer.Spaces()
		if err != nil {
			return sql, args, err
		}
		if len(spaces) < 1 {
			return sql, args, errors.New("unauthorized: you are not authorized to view any space usage data")
		}
		conditions := make([]string, len(spaces))
		for i, guid := range spaces {
			args = append(args, guid)
			conditions[i] = fmt.Sprintf("space_guid = $%d", len(args))
		}
		cond = "where " + strings.Join(conditions, " or ")
	}
	sql = fmt.Sprintf(`
		with authorized_resources as (
			select *
			from billable_range(tstzrange('%s', '%s'))
			%s
		),
		q as (
			%s
		)
		select * from q
	`, rng.From, rng.To, cond, sql)
	return sql, args, nil
}

func withAuthorizedResources(rt resourceType, c echo.Context, db db.SQLClient, sql string, args ...interface{}) (err error) {
	rng, ok := c.Get("range").(RangeParams)
	if !ok {
		return errors.New("bad request: no range params in context")
	}
	authorizer, ok := c.Get("authorizer").(auth.Authorizer)
	if !ok {
		return errors.New("unauthorized: no authorizer in context")
	}
	sql, args, err = authorizedSpaceFilter(authorizer, rng, sql, args)
	if err != nil {
		return err
	}
	return render(rt, c, db, sql, args...)
}

func render(rt resourceType, c echo.Context, db db.SQLClient, sql string, args ...interface{}) error {
	var r io.Reader
	if rt == Single {
		r = db.QueryRowJSON(sql, args...)
	} else if rt == Many {
		r = db.QueryJSON(sql, args...)
	}
	acceptHeader := c.Request().Header.Get(echo.HeaderAccept)
	accepts := strings.Split(acceptHeader, ",")
	overrideAccept := c.QueryParam("Accept")
	if overrideAccept != "" {
		accepts = []string{overrideAccept}
	}
	for _, accept := range accepts {
		if accept == echo.MIMEApplicationJSON || accept == echo.MIMEApplicationJSONCharsetUTF8 {
			c.Response().Writer.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			_, err := io.Copy(c.Response(), r)
			return err
		} else if accept == echo.MIMETextHTML || accept == echo.MIMETextHTMLCharsetUTF8 {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
			c.Response().WriteHeader(http.StatusOK)
			return Render(c, r, int(rt))
		}
	}
	return c.HTML(http.StatusNotAcceptable, "unacceptable")
}
