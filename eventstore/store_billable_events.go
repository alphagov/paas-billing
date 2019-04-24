package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.BillableEventReader = &EventStore{}

// GetBillableEventRows returns a handle to a resultset of BillableEvents. Use
// this to iterate over rows without buffering all into memory. You must call
// rows.Close when you are done to release the connection. Use GetBillableEvents
// if you intend on buffering everything into memory.
func (s *EventStore) GetBillableEventRows(ctx context.Context, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	rows, err := s.getBillableEventRows(tx, filter)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return rows, nil
}

func (s *EventStore) getBillableEventRows(tx *sql.Tx, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}

	query, args, err := WithBillableEvents(
		`select * from billable_events`,
		filter,
	)
	if err != nil {
		return nil, err
	}

	rows, err := queryJSON(tx, query, args...)
	if err != nil {
		return nil, err
	}

	return &BillableEventRows{rows}, nil
}

// GetBillableEvents returns a slice of billable events for the given filter.
// Due to the large number of results that can be returned it is recormended
// you use the GetBillableEventRows version to avoid buffering everything into
// memory
func (s *EventStore) GetBillableEvents(filter eventio.EventFilter) ([]eventio.BillableEvent, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := s.GetBillableEventRows(ctx, filter)
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

type BillableEventRows struct {
	rows *sql.Rows
}

// Next moves the row cursor to the next iteration. Returns false if no more
// rows.
func (ber *BillableEventRows) Next() bool {
	return ber.rows.Next()
}

// Err returns any errors that occurred behind the scenes during processing.
// Call this at the end of your iteration.
func (ber *BillableEventRows) Err() error {
	return ber.rows.Err()
}

// Close ends the query connection. You must call this. So stick it in a defer.
func (ber *BillableEventRows) Close() error {
	return ber.rows.Close()
}

// EventJSON returns the JSON representation of the event directly from the db.
// If you are just going to marshal the object to JSON immediately, then this
// is probably more efficient.
func (ber *BillableEventRows) EventJSON() ([]byte, error) {
	var b []byte
	if err := ber.rows.Scan(&b); err != nil {
		return nil, err
	}
	return b, nil
}

// Event returns the current row's BillableEvent. Call Next() to get the next
// row. You must call Next _before_ calling this method
func (ber *BillableEventRows) Event() (*eventio.BillableEvent, error) {
	b, err := ber.EventJSON()
	if err != nil {
		return nil, err
	}
	var event eventio.BillableEvent
	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// WithBillableEvents wraps a given query with a subquery called
// billable_events, containing the result of applying the given pricing
// formula to the events for the given filter.
//
// Other included tables are:
//  - components_with_price: Components and formulas selected for this filter
//  - filtered range: time range of the filter
func WithBillableEvents(query string, filter eventio.EventFilter, args ...interface{}) (string, []interface{}, error) {
	if err := filter.Validate(); err != nil {
		return query, args, err
	}
	args = append(args, fmt.Sprintf("[%s, %s)", filter.RangeStart, filter.RangeStop)) // $1
	durationArgPosition := len(args)

	filterConditions := []string{}
	orgPlaceholders := []string{}
	for _, orgGUID := range filter.OrgGUIDs {
		args = append(args, orgGUID)
		orgPlaceholders = append(orgPlaceholders, fmt.Sprintf("($%d::uuid)", len(args))) // $N
	}
	if len(orgPlaceholders) > 0 {
		filterConditions = append(filterConditions, fmt.Sprintf("org_guid = any (values %s)", strings.Join(orgPlaceholders, ",")))
	}
	filterQuery := ""
	if len(filterConditions) > 0 {
		filterQuery = " and " + strings.Join(filterConditions, " and ")
	}

	wrappedQuery := fmt.Sprintf(`
		with
		filtered_range as (
			select $%d::tstzrange as filtered_range
		),
		components_with_price as (
			select
				b.event_guid,
				b.resource_guid,
				b.resource_name,
				b.resource_type,
				b.org_guid,
				b.org_name,
				o.quota_definition_guid,
				b.space_guid,
				b.space_name,
				b.plan_guid,
				b.plan_name,
				b.duration * filtered_range as duration,
				b.number_of_nodes,
				b.memory_in_mb,
				b.storage_in_mb,
				b.component_name,
				b.component_formula,
				b.vat_code,
				b.vat_rate,
				'GBP' as currency_code,
				b.currency_rate,
				(eval_formula(
					b.memory_in_mb,
					b.storage_in_mb,
					b.number_of_nodes,
					b.duration * filtered_range,
					b.component_formula
				) * b.currency_rate) as price_ex_vat
			from
			    filtered_range,
				billable_event_components b
			left join
				orgs o on b.org_guid = o.guid
			where
				duration && filtered_range
				%s
			order by
				lower(duration) asc
		),
		billable_events as (
			select
				event_guid,
				min(lower(duration)) as event_start,
				max(upper(duration)) as event_stop,
				resource_guid,
				resource_name,
				resource_type,
				org_guid,
				org_name,
				quota_definition_guid,
				space_guid,
				space_name,
				plan_guid,
				number_of_nodes,
				memory_in_mb,
				storage_in_mb,
				json_build_object(
					'ex_vat', (sum(price_ex_vat))::text,
					'inc_vat', (sum(price_ex_vat * (1 + vat_rate)))::text,
					'details', json_agg(json_build_object(
						'name', component_name,
						'start', lower(duration),
						'stop', upper(duration),
						'plan_name', plan_name,
						'ex_vat', (price_ex_vat)::text,
						'inc_vat', (price_ex_vat * (1 + vat_rate))::text,
						'vat_rate', (vat_rate)::text,
						'vat_code', vat_code,
						'currency_code', currency_code,
						'currency_rate', (currency_rate)::text
					))
				) as price
			from
				components_with_price
			group by
				event_guid,
				resource_guid,
				resource_name,
				resource_type,
				org_guid,
				org_name,
				quota_definition_guid,
				space_guid,
				space_name,
				plan_guid,
				number_of_nodes,
				memory_in_mb,
				storage_in_mb
			order by
				event_guid
	  )
	  %s
	  `,
		durationArgPosition,
		filterQuery,
		query,
	)

	return wrappedQuery, args, nil
}
