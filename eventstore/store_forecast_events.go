package eventstore

import (
	"database/sql"

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

func (s *EventStore) ForecastBillableEventRows(events []eventio.UsageEvent, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	tx, err := s.db.Begin()
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
	for _, ev := range events {
		_, err := tx.Exec(`
			insert into events (
				event_guid,
				resource_guid, resource_name, resource_type,
				org_guid, org_name, space_guid, space_name,
				duration,
				plan_guid, plan_name,
				number_of_nodes, memory_in_mb, storage_in_mb
			) values (
				$1::uuid,
				$2::uuid, $3::text, $4::text,
				$5::uuid, $6::text, $7::uuid, $8::text,
				tstzrange($9::timestamptz, $10::timestamptz),
				$11::uuid, 'simulated',
				$12::numeric, $13::numeric, $14::numeric
			)
		`,
			ev.EventGUID,
			ev.ResourceGUID, ev.ResourceName, ev.ResourceType,
			ev.OrgGUID, ev.OrgName, ev.SpaceGUID, ev.SpaceName,
			ev.EventStart, ev.EventStop,
			ev.PlanGUID,
			ev.NumberOfNodes, ev.MemoryInMB, ev.StorageInMB,
		)
		if err != nil {
			return nil, err
		}
		eventGUIDs = append(eventGUIDs, ev.EventGUID)
	}
	_, err := tx.Exec(`
		insert into billable_event_components (
			select * from generate_billable_event_components()
			where event_guid = any($1)
		)
	`, pq.Array(eventGUIDs))
	if err != nil {
		return nil, err
	}

	return s.getBillableEventRows(tx, filter)
}

func (s *EventStore) ForecastBillableEvents(input []eventio.UsageEvent, filter eventio.EventFilter) ([]eventio.BillableEvent, error) {
	rows, err := s.ForecastBillableEventRows(input, filter)
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
