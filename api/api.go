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

const resourceDurationsViewName = "resource_usage"

type SimulatedEvents struct {
	Events []SimulatedEvent `json:"events"`
}

type SimulatedEvent struct {
	ResourceName  string `json:"resource_name"`
	SpaceGUID     string `json:"space_guid"`
	PlanGUID      string `json:"plan_guid"`
	MemoryInMB    uint   `json:"memory_in_mb"`
	StorageInMB   uint   `json:"storage_in_mb"`
	NumberOfNodes uint   `json:"number_of_nodes"`
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

		tempTableName := "temp_resource_usage"
		_, err = dbTx.Exec(`CREATE TEMPORARY TABLE ` + tempTableName + ` (
				event_guid serial,
				resource_guid text,
				resource_name text,
				org_guid text,
				space_guid text,
				plan_guid text,
				memory_in_mb numeric,
				storage_in_mb numeric,
				number_of_nodes integer,
				duration tstzrange
			)`,
		)
		if err != nil {
			return err
		}
		stmt, err := dbTx.Prepare(`INSERT INTO ` + tempTableName + ` (
			resource_guid,
			resource_name,
			org_guid,
			space_guid,
			plan_guid,
			memory_in_mb,
			storage_in_mb,
			number_of_nodes,
			duration
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, tstzrange($9, $10)
		)`)
		if err != nil {
			return err
		}
		for _, event := range events.Events {
			_, err = stmt.Exec(
				event.ResourceName+"-guid",
				event.ResourceName,
				orgGUID,
				event.SpaceGUID,
				event.PlanGUID,
				event.MemoryInMB,
				event.StorageInMB,
				event.NumberOfNodes,
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
		return generateReport(orgGUID, resourceDurationsViewName, c, db)
	}
}

func ListOrgUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
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
		return withAuthorizedResources(Single, resourceDurationsViewName, c, db, `
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
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
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
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
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
		return withAuthorizedResources(Single, resourceDurationsViewName, c, db, `
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
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
			select
				org_guid,
				space_guid,
				resource_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
			where
				org_guid = $1
			group by
				org_guid, space_guid, resource_guid
			order by
				resource_guid
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
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
			select
				org_guid,
				space_guid,
				resource_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
			where
				space_guid = $1
			group by
				org_guid, space_guid, resource_guid
			order by
				resource_guid
		`, spaceGUID)
	}
}

func ListResourceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
			select
				org_guid,
				space_guid,
				resource_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
			group by
				space_guid, org_guid, resource_guid
			order by
				resource_guid
		`)
	}
}

func GetResourceUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		resourceGUID := c.Param("resource_guid")
		if resourceGUID == "" {
			return errors.New("missing resource_guid")
		}
		return withAuthorizedResources(Single, resourceDurationsViewName, c, db, `
			select
				org_guid,
				space_guid,
				resource_guid,
				(sum(price_inc_vat) * 100)::bigint as price_in_pence_inc_vat,
				(sum(price_ex_vat) * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
			where
				resource_guid = $1
			group by
				org_guid, space_guid, resource_guid
			order by
				resource_guid
		`, resourceGUID)
	}
}

func ListEventUsageForResource(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		resourceGUID := c.Param("resource_guid")
		if resourceGUID == "" {
			return errors.New("missing resource_guid")
		}
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
			select
				resource_guid,
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
				resource_guid = $1
			group by
				resource_guid, event_guid, pricing_plan_id, pricing_plan_name, org_guid, space_guid, duration
			order by
				resource_guid, event_guid, pricing_plan_id
		`, resourceGUID)
	}
}

func ListEventUsage(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
			select
				resource_guid,
				org_guid,
				space_guid,
				pricing_plan_id,
				pricing_plan_name,
				resource_name,
				memory_in_mb,
				storage_in_mb,
				number_of_nodes,
				iso8601(lower(duration)) as from,
				iso8601(upper(duration)) as to,
				sum(price_inc_vat::bigint) as price_inc_vat,
				sum(price_ex_vat::bigint) as price_ex_vat
			from
				monetized_resources
			group by
				resource_guid, event_guid, pricing_plan_id, pricing_plan_name, org_guid, space_guid,
				resource_name, memory_in_mb, storage_in_mb, number_of_nodes, duration
			order by
				resource_guid, event_guid, pricing_plan_id
		`)
	}
}

func ListEventUsageRaw(db db.SQLClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return withAuthorizedResources(Many, resourceDurationsViewName, c, db, `
			select
				*,
				(price_inc_vat * 100)::bigint as price_in_pence_inc_vat,
				(price_ex_vat * 100)::bigint as price_in_pence_ex_vat
			from
				monetized_resources
			order by
				resource_guid, event_guid, pricing_plan_id, pricing_plan_component_id, lower(duration)
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

func authorizedSpaceFilter(authorizer auth.Authorizer, resourceDurationsViewName string, rng RangeParams, sql string, args []interface{}) (string, []interface{}, error) {
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

	return monetizedResourcesFilter(cond, resourceDurationsViewName, rng, sql, args)
}

func monetizedResourcesFilter(filterCondition string, resourceDurationsViewName string, rng RangeParams, sql string, args []interface{}) (string, []interface{}, error) {
	templateVars := struct {
		TableName            string
		RangeFromPlaceholder string
		RangeToPlaceholder   string
		Condition            string
		SQL                  string
	}{
		TableName:            resourceDurationsViewName,
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
				b.event_guid,
				b.resource_guid,
				b.resource_name,
				b.org_guid,
				b.space_guid,
				coalesce(b.memory_in_mb, vpp.memory_in_mb)::numeric as memory_in_mb,
				coalesce(b.storage_in_mb, vpp.storage_in_mb)::numeric as storage_in_mb,
				coalesce(b.number_of_nodes, vpp.number_of_nodes)::integer as number_of_nodes,
				r.request_range * vpp.valid_for * vcr.valid_for * b.duration as duration,
				vpp.id AS pricing_plan_id,
				vpp.name AS pricing_plan_name,
				ppc.id AS pricing_plan_component_id,
				ppc.name AS pricing_plan_component_name,
				ppc.formula,
				eval_formula(
					coalesce(b.memory_in_mb, vpp.memory_in_mb)::numeric,
					coalesce(b.storage_in_mb, vpp.storage_in_mb)::numeric,
					coalesce(b.number_of_nodes, vpp.number_of_nodes)::integer,
					r.request_range * vpp.valid_for * vcr.valid_for * b.duration,
					ppc.formula
				) * vcr.rate as price_ex_vat,
				eval_formula(
					coalesce(b.memory_in_mb, vpp.memory_in_mb)::numeric,
					coalesce(b.storage_in_mb, vpp.storage_in_mb)::numeric,
					coalesce(b.number_of_nodes, vpp.number_of_nodes)::integer,
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
				pricing_plan_components ppc
			on
				ppc.pricing_plan_id = vpp.id
			inner join
				valid_currency_rates vcr
			on
				vcr.valid_for && vpp.valid_for
				and vcr.valid_for && b.duration
				and vcr.valid_for && r.request_range
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

func withAllResources(rt resourceType, resourceDurationsViewName string, c echo.Context, db db.SQLClient, sql string, args ...interface{}) (err error) {
	rng, ok := c.Get("range").(RangeParams)
	if !ok {
		return errors.New("bad request: no range params in context")
	}
	sql, args, err = monetizedResourcesFilter("", resourceDurationsViewName, rng, sql, args)
	if err != nil {
		return err
	}
	return render(rt, c, db, sql, args...)
}

func withAuthorizedResources(rt resourceType, resourceDurationsViewName string, c echo.Context, db db.SQLClient, sql string, args ...interface{}) (err error) {
	rng, ok := c.Get("range").(RangeParams)
	if !ok {
		return errors.New("bad request: no range params in context")
	}
	authorizer, ok := c.Get("authorizer").(auth.Authorizer)
	if !ok {
		return errors.New("unauthorized: no authorizer in context")
	}
	sql, args, err = authorizedSpaceFilter(authorizer, resourceDurationsViewName, rng, sql, args)
	if err != nil {
		return err
	}
	return render(rt, c, db, sql, args...)
}

func generateReport(orgGUID string, resourceDurationsViewName string, c echo.Context, db db.SQLClient) error {
	return withAllResources(Single, resourceDurationsViewName, c, db, `
		with
		resources as (
			select
				resource_name,
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
				resource_name, space_guid, pricing_plan_id, pricing_plan_name
			order by
				resource_name, space_guid, pricing_plan_id
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
