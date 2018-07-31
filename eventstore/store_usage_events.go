package eventstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alphagov/paas-billing/eventio"
)

var _ eventio.UsageEventReader = &EventStore{}

// GetUsageEventRows returns a handle to a resultset of UsageEvents. Use
// this to iterate over rows without buffering all into memory. You must call
// rows.Close when you are done to release the connection. Use GetUsageEvents
// if you intend on buffering everything into memory.
func (s *EventStore) GetUsageEventRows(filter eventio.EventFilter) (eventio.UsageEventRows, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	rows, err := s.getUsageEventRows(tx, filter)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return rows, nil
}

func (s *EventStore) getUsageEventRows(tx *sql.Tx, filter eventio.EventFilter) (eventio.UsageEventRows, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	args := []interface{}{
		fmt.Sprintf("[%s, %s)", filter.RangeStart, filter.RangeStop), // $1
	}
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
	rows, err := s.queryJSON(tx, fmt.Sprintf(`
		select
			event_guid,
			to_json(lower(duration * $1::tstzrange)) as event_start,
			to_json(upper(duration * $1::tstzrange)) as event_stop,
			resource_guid,
			resource_name,
			resource_type,
			org_guid,
			org_name,
			space_guid,
			space_name,
			plan_guid,
			plan_name,
			service_guid,
			service_name,
			number_of_nodes,
			memory_in_mb,
			storage_in_mb
		from
			events
		where
			duration && $1::tstzrange
			%s
		order by
			lower(duration), event_guid
	`, filterQuery), args...)
	if err != nil {
		return nil, err
	}
	return &UsageEventRows{rows, tx}, nil
}

// GetUsageEvents returns a slice of usage events for the given filter.
// Due to the large number of results that can be returned it is recormended
// you use the GetUsageEventRows version to avoid buffering everything into
// memory
func (s *EventStore) GetUsageEvents(filter eventio.EventFilter) ([]eventio.UsageEvent, error) {
	rows, err := s.GetUsageEventRows(filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []eventio.UsageEvent{}
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

type UsageEventRows struct {
	rows *sql.Rows
	tx   *sql.Tx
}

// Next moves the row cursor to the next iteration. Returns false if no more
// rows.
func (ber *UsageEventRows) Next() bool {
	return ber.rows.Next()
}

// Err returns any errors that occured behind the scenes during processing.
// Call this at the end of your iteration.
func (ber *UsageEventRows) Err() error {
	return ber.rows.Err()
}

// Close ends the query connection. You must call this. So stick it in a defer.
func (ber *UsageEventRows) Close() error {
	ber.tx.Rollback()
	return ber.rows.Close()
}

// EventJSON returns the JSON representation of the event directly from the db.
// If you are just going to marshel the object to JSON immediately, then this
// is probably more effcient.
func (ber *UsageEventRows) EventJSON() ([]byte, error) {
	var b []byte
	if err := ber.rows.Scan(&b); err != nil {
		return nil, err
	}
	return b, nil
}

// Event returns the current row's UsageEvent. Call Next() to get the next
// row. You must call Next _before_ calling this method
func (ber *UsageEventRows) Event() (*eventio.UsageEvent, error) {
	b, err := ber.EventJSON()
	if err != nil {
		return nil, err
	}
	var event eventio.UsageEvent
	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
