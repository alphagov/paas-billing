package eventstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/lib/pq"
)

var _ eventio.BillableEventForecaster = &EventStore{}

const (
	DummyOrgGUID   = "00000001-0000-0000-0000-000000000000"
	DummyOrgName   = "my-org"
	DummySpaceGUID = "00000001-0001-0000-0000-000000000000"
	DummySpaceName = "my-space"
)

func (s *EventStore) ForecastBillableEventRows(ctx context.Context, events []eventio.UsageEvent, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	rows, err := s.forecastBillableEventRows(tx, events, filter)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return rows, nil
}

func (s *EventStore) forecastBillableEventRows(tx *sql.Tx, events []eventio.UsageEvent, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	eventGUIDs := []string{}
	eventsCTE := `events (event_guid, resource_guid, resource_name, resource_type, org_guid, org_name, space_guid, space_name, duration, plan_guid, plan_name, number_of_nodes, memory_in_mb, storage_in_mb)
	AS (VALUES `
	for i, ev := range events {
		eventGUIDs = append(eventGUIDs, ev.EventGUID)

		if i != 0 {
			eventsCTE += ", "
		}

		eventsCTE += fmt.Sprintf(`(
				%s::uuid,
				%s::uuid,
				%s::text,
				%s::text,
				%s::uuid,
				%s::text,
				%s::uuid,
				%s::text,
				tstzrange(%s::timestamptz, %s::timestamptz),
				%s::uuid,
				'simulated',
				%d::numeric,
				%d::numeric,
				%d::numeric
			)`,
			pq.QuoteLiteral(ev.EventGUID),
			pq.QuoteLiteral(ev.ResourceGUID),
			pq.QuoteLiteral(ev.ResourceName),
			pq.QuoteLiteral(ev.ResourceType),
			pq.QuoteLiteral(ev.OrgGUID),
			pq.QuoteLiteral(ev.OrgName),
			pq.QuoteLiteral(ev.SpaceGUID),
			pq.QuoteLiteral(ev.SpaceName),
			pq.QuoteLiteral(ev.EventStart),
			pq.QuoteLiteral(ev.EventStop),
			pq.QuoteLiteral(ev.PlanGUID),
			ev.NumberOfNodes,
			ev.MemoryInMB,
			ev.StorageInMB,
		)
	}
	eventsCTE += ")"

	billableEventComponentsCTE := `
		valid_pricing_plans as (
        select
            *,
            tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
                partition by plan_guid order by valid_from rows between current row and 1 following
            )) as valid_for
        from
            pricing_plans
    ),
    valid_currency_rates as (
        select
            *,
            tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
                partition by code order by valid_from rows between current row and 1 following
            )) as valid_for
        from
            currency_rates
    ),
    valid_vat_rates as (
        select
            *,
            tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
                partition by code order by valid_from rows between current row and 1 following
            )) as valid_for
        from
            vat_rates
    ),
		billable_event_components AS (
    select
        ev.event_guid,
        ev.resource_guid,
        ev.resource_name,
        ev.resource_type,
        ev.org_guid,
        ev.org_name,
        ev.space_guid,
        ev.space_name,
        ev.duration * vpp.valid_for * vcr.valid_for * vvr.valid_for as duration,
        vpp.plan_guid as plan_guid,
        vpp.valid_from as plan_valid_from,
        vpp.name as plan_name,
        coalesce(ev.number_of_nodes, vpp.number_of_nodes)::integer as number_of_nodes,
        coalesce(ev.memory_in_mb, vpp.memory_in_mb)::numeric as memory_in_mb,
        coalesce(ev.storage_in_mb, vpp.storage_in_mb)::numeric as storage_in_mb,
        ppc.name AS component_name,
        ppc.formula as component_formula,
        vcr.code as currency_code,
        vcr.rate as currency_rate,
        vvr.code as vat_code,
        vvr.rate as vat_rate,
        (eval_formula(
            coalesce(ev.memory_in_mb, vpp.memory_in_mb)::numeric,
            coalesce(ev.storage_in_mb, vpp.storage_in_mb)::numeric,
            coalesce(ev.number_of_nodes, vpp.number_of_nodes)::integer,
            ev.duration * vpp.valid_for * vcr.valid_for * vvr.valid_for,
            ppc.formula
        ) * vcr.rate) as cost_for_duration
    from
        events ev
    left join
				valid_pricing_plans vpp on ev.plan_guid::uuid = vpp.plan_guid::uuid
        and vpp.valid_for && ev.duration
    left join
				pricing_plan_components ppc on ppc.plan_guid::uuid = vpp.plan_guid::uuid
        and ppc.valid_from = vpp.valid_from
    left join
        valid_currency_rates vcr on vcr.code = ppc.currency_code
        and vcr.valid_for && (ev.duration * vpp.valid_for)
    left join
        valid_vat_rates vvr on vvr.code = ppc.vat_code
        and vvr.valid_for && (ev.duration * vpp.valid_for * vcr.valid_for)
		WHERE event_guid IN (`

	for i, eventGUID := range eventGUIDs {
		if i != 0 {
			billableEventComponentsCTE += ", "
		}

		billableEventComponentsCTE += fmt.Sprintf(
			"%s::uuid", pq.QuoteLiteral(eventGUID),
		)
	}
	billableEventComponentsCTE += "))"

	query := eventsCTE + ",\n" + billableEventComponentsCTE + ",\n"

	return s.getBillableEventRows(tx, query, filter)
}

func (s *EventStore) ForecastBillableEvents(input []eventio.UsageEvent, filter eventio.EventFilter) ([]eventio.BillableEvent, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := s.ForecastBillableEventRows(ctx, input, filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []eventio.BillableEvent{}
	for rows.Next() {
		ev, err := rows.Event()
		if err != nil {
			return nil, err
		}
		events = append(events, *ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
