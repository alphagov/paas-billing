package api

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/alphagov/paas-billing/auth"
	"github.com/alphagov/paas-billing/db"
	"github.com/labstack/echo"
)

const billableViewName = "billable"

type SimulatedEvents struct {
	Events []SimulatedEvent `json:"events"`
}

type SimulatedEvent struct {
	Name       string `json:"name"`
	SpaceGUID  string `json:"space_guid"`
	PlanGUID   string `json:"plan_guid"`
	MemoryInMB int    `json:"memory_in_mb"`
}

func NewSimulatedReportHandler(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		orgGUID := "simulated-org"
		rng, ok := c.Get("range").(RangeParams)
		if !ok {
			return errors.New("bad request: no range params in context")
		}

		var events SimulatedEvents
		err := c.Bind(&events)
		if err != nil {
			return err
		}

		dbTx, err := db.BeginTx()
		if err != nil {
			return err
		}
		defer dbTx.Rollback()

		tempTableName := "temp_billable"
		_, err = dbTx.Exec(`CREATE TEMPORARY TABLE ` + tempTableName + ` (
				id serial,
				guid text,
				name text,
				org_guid text,
				space_guid text,
				plan_guid text,
				memory_in_mb numeric,
				duration tstzrange
			)`,
		)
		if err != nil {
			return err
		}
		stmt, err := dbTx.Prepare(`INSERT INTO ` + tempTableName + ` (
		  guid,
			name,
			org_guid,
			space_guid,
			plan_guid,
			memory_in_mb,
			duration
		) VALUES (
			$1, $2, $3, $4, $5, $6, tstzrange($7, $8)
		)`)
		if err != nil {
			return err
		}
		for _, event := range events.Events {
			_, err = stmt.Exec(
				event.Name+"-guid",
				event.Name,
				orgGUID,
				event.SpaceGUID,
				event.PlanGUID,
				event.MemoryInMB,
				rng.From,
				rng.To,
			)
			if err != nil {
				return err
			}
		}

		return generateReport(orgGUID, tempTableName, c, dbTx)
	}
}

func NewOrgReportHandler(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		orgGUID := c.Param("org_guid")
		if orgGUID == "" {
			return errors.New("missing org_guid")
		}
		return generateReport(orgGUID, billableViewName, c, db)
	}
}

func ListOrgUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				org_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Single, billableViewName, c, db, `
			select
				org_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Single, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Single, billableViewName, c, db, `
			select
				org_guid,
				space_guid,
				guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
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
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				guid,
				org_guid,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				iso8601(lower(duration)) as from,
				iso8601(upper(duration)) as to,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources mr
			where
				guid = $1
			group by
				guid, id, pricing_plan_id, pricing_plan_name, org_guid, space_guid, duration
			order by
				guid, id, pricing_plan_id
		`, resourceGUID)
	}
}

func ListEventUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, billableViewName, c, db, `
			select
				guid,
				org_guid,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				name,
				memory_in_mb,
				iso8601(lower(duration)) as from,
				iso8601(upper(duration)) as to,
				sum(price_inc_vat::bigint) as price_inc_vat,
				sum(price_ex_vat::bigint) as price_ex_vat
			from
				monetized_resources
			group by
				guid, id, pricing_plan_id, pricing_plan_name, org_guid, space_guid,
				name, memory_in_mb, duration
			order by
				guid, id, pricing_plan_id
		`)
	}
}

type resourceType int

const (
	Invalid resourceType = iota
	Empty
	Single
	Many
)

func authorizedSpaceFilter(authorizer auth.Authorizer, billableTableName string, rng RangeParams, sql string, args []interface{}) (string, []interface{}, error) {
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

	return monetizedResourcesFilter(cond, billableTableName, rng, sql, args)
}

func monetizedResourcesFilter(filterCondition string, billableTableName string, rng RangeParams, sql string, args []interface{}) (string, []interface{}, error) {
	templateVars := struct {
		TableName            string
		RangeFromPlaceholder string
		RangeToPlaceholder   string
		Condition            string
		SQL                  string
	}{
		TableName:            billableTableName,
		RangeFromPlaceholder: fmt.Sprintf("$%d", len(args)+1),
		RangeToPlaceholder:   fmt.Sprintf("$%d", len(args)+2),
		Condition:            filterCondition,
		SQL:                  sql,
	}
	templateSQL := `
		with
		valid_pricing_plans as (
			select
				pp.*,
				tstzrange(valid_from, lead(valid_from, 1, 'infinity') over plans) as valid_for
			from
				pricing_plans pp
			window
				plans as (partition by plan_guid order by valid_from rows between current row and 1 following)
		),
		valid_currency_rates as (
			select
				cr.*,
				tstzrange(valid_from, lead(valid_from, 1, 'infinity') over currencies) as valid_for
			from
				currency_rates cr
			window
				currencies as (partition by code order by valid_from rows between current row and 1 following)
		),
		authorized_resources as (
			select *
			from {{ .TableName }}
			{{ .Condition }}
		),
		request_range as (
			select tstzrange( {{ .RangeFromPlaceholder }}, {{ .RangeToPlaceholder }} ) as request_range
		),
		monetized_resources as (
			select
				b.id,
				b.guid,
				b.name,
				b.org_guid,
				b.space_guid,
				b.memory_in_mb,
				r.request_range * vpp.valid_for * vcr.valid_for * b.duration as duration,
				vpp.id AS pricing_plan_id,
				vpp.name AS pricing_plan_name,
				ppc.id AS pricing_plan_component_id,
				ppc.name AS pricing_plan_component_name,
				ppc.formula,
				eval_formula(
					b.memory_in_mb,
					r.request_range * vpp.valid_for * vcr.valid_for * b.duration,
					ppc.formula
				) * vcr.rate as price_ex_vat,
				eval_formula(
					b.memory_in_mb,
					r.request_range * vpp.valid_for * vcr.valid_for * b.duration,
					ppc.formula
				) * vcr.rate * (1 + vr.rate) as price_inc_vat,
				vr.name as vat_rate_name
			from
				authorized_resources b
			cross join
				request_range r
			inner join
				valid_pricing_plans vpp
			on
				b.plan_guid = vpp.plan_guid
				and vpp.valid_for && b.duration
			  and vpp.valid_for && r.request_range
		  inner join
				valid_currency_rates vcr
			on
				vcr.valid_for && vpp.valid_for
				and vcr.valid_for && b.duration
			  and vcr.valid_for && r.request_range
			inner join
				pricing_plan_components ppc
			on
				ppc.pricing_plan_id = vpp.id
				and ppc.currency = vcr.code
			inner join
				vat_rates vr
			on
				ppc.vat_rate_id = vr.id
			where
				b.duration && r.request_range
		),
		q as (
			{{ .SQL }}
		)
		select * from q
	`
	var buf bytes.Buffer
	tmpl, err := template.New("sql").Parse(templateSQL)
	if err != nil {
		return "", args, err
	}
	err = tmpl.Execute(&buf, templateVars)
	if err != nil {
		return "", args, err
	}

	return buf.String(), append(args, rng.From, rng.To), nil
}

func withAllResources(rt resourceType, billableTableName string, c echo.Context, db db.SQLClient, sql string, args ...interface{}) (err error) {
	rng, ok := c.Get("range").(RangeParams)
	if !ok {
		return errors.New("bad request: no range params in context")
	}
	sql, args, err = monetizedResourcesFilter("", billableTableName, rng, sql, args)
	if err != nil {
		return err
	}
	return render(rt, c, db, sql, args...)
}

func withAuthorizedResources(rt resourceType, billableTableName string, c echo.Context, db db.SQLClient, sql string, args ...interface{}) (err error) {
	rng, ok := c.Get("range").(RangeParams)
	if !ok {
		return errors.New("bad request: no range params in context")
	}
	authorizer, ok := c.Get("authorizer").(auth.Authorizer)
	if !ok {
		return errors.New("unauthorized: no authorizer in context")
	}
	sql, args, err = authorizedSpaceFilter(authorizer, billableTableName, rng, sql, args)
	if err != nil {
		return err
	}
	return render(rt, c, db, sql, args...)
}

func generateReport(orgGUID string, billableTableName string, c echo.Context, db db.SQLClient) error {
	return withAllResources(Single, billableTableName, c, db, `
		with
		resources as (
			select
				name,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				sum(to_seconds(duration)) as duration,
				sum(price_ex_vat * 100)::bigint as price_ex_vat,
				sum(price_inc_vat * 100)::bigint as price_inc_vat
			from
				monetized_resources
			where
				org_guid = $1
			group by
				name, space_guid, pricing_plan_id, pricing_plan_name
			order by
				name, space_guid, pricing_plan_id
		),
		space_resources as (
			select
				t.space_guid,
				sum(t.price_ex_vat) as price_ex_vat,
				sum(t.price_inc_vat) as price_inc_vat,
				json_agg(row_to_json(t.*)) as resources
			from
				resources t
			group by
				space_guid
			order by
				space_guid
		)
		select
			$1 org_guid,
			sum(t.price_ex_vat) as price_ex_vat,
			sum(t.price_inc_vat) as price_inc_vat,
			json_agg(row_to_json(t.*)) as spaces
		from
			space_resources t
	`, orgGUID)
}

func render(rt resourceType, c echo.Context, db db.SQLClient, sql string, args ...interface{}) error {
	var r io.Reader
	if rt == Single {
		r = db.QueryRowJSON(sql, args...)
	} else if rt == Many {
		r = db.QueryJSON(sql, args...)
	} else if rt == Empty {
		_, err := db.Exec(sql, args...)
		if err != nil {
			return err
		}
		r = bytes.NewReader(nil)
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
			if rt == Empty {
				c.Response().Write([]byte(`{"success":true}`))
				return nil
			}

			written, err := io.Copy(c.Response(), r)
			if err != nil {
				return err
			}

			if rt == Single && written == 0 {
				c.Response().WriteHeader(http.StatusNotFound)
				c.Response().Write([]byte(`{"error":{"message":"not found"}}`))
			}

			return nil
		} else if accept == echo.MIMETextHTML || accept == echo.MIMETextHTMLCharsetUTF8 {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
			c.Response().WriteHeader(http.StatusOK)
			return Render(c, r, int(rt))
		}
	}
	return c.HTML(http.StatusNotAcceptable, "unacceptable")
}
