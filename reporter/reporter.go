package reporter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type PriceComponent struct {
	Name         string `json:"name"`
	PlanName     string `json:"plan_name"`
	Start        string `json:"start"`
	Stop         string `json:"stop"`
	VatRate      string `json:"vat_rate"`
	VatCode      string `json:"vat_code"`
	CurrencyCode string `json:"currency_code"`
	CurrencyRate string `json:"currency_rate"`
	IncVAT       string `json:"inc_vat"`
	ExVAT        string `json:"ex_vat"`
}

type Price struct {
	IncVAT  string           `json:"inc_vat"`
	ExVAT   string           `json:"ex_vat"`
	Details []PriceComponent `json:"details"`
}

type BillableEvent struct {
	EventGUID     string `json:"event_guid"`
	EventStart    string `json:"event_start"`
	EventStop     string `json:"event_stop"`
	ResourceGUID  string `json:"resource_guid"`
	ResourceName  string `json:"resource_name"`
	ResourceType  string `json:"resource_type"`
	OrgGUID       string `json:"org_guid"`
	SpaceGUID     string `json:"space_guid"`
	PlanGUID      string `json:"plan_guid"`
	NumberOfNodes int64  `json:"number_of_nodes"`
	MemoryInMB    int64  `json:"memory_in_mb"`
	StorageInMB   int64  `json:"storage_in_mb"`
	Price         Price  `json:"price"`
}

func (e *BillableEvent) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("cannot Scan into BillableEvent with: %T", src)
	}
	if err := json.Unmarshal(source, e); err != nil {
		return err
	}
	return nil
}

type BillableEventRows struct {
	rows *sql.Rows
}

// Next moves the row cursor to the next iteration. Returns false if no more
// rows.
func (ber *BillableEventRows) Next() bool {
	return ber.rows.Next()
}

// Err returns any errors that occured behind the scenes during processing.
// Call this at the end of your iteration.
func (ber *BillableEventRows) Err() error {
	return ber.rows.Err()
}

// Close ends the query connection. You must call this. So stick it in a defer.
func (ber *BillableEventRows) Close() error {
	return ber.rows.Close()
}

// EventJSON returns the JSON representation of the event directly from the db.
// If you are just going to marshel the object to JSON immediately, then this
// is probably more effcient.
func (ber *BillableEventRows) EventJSON() ([]byte, error) {
	var b []byte
	if err := ber.rows.Scan(&b); err != nil {
		return nil, err
	}
	return b, nil
}

// Event returns the current row's BillableEvent. Call Next() to get the next
// row. You must call Next _before_ calling this method
func (ber *BillableEventRows) Event() (*BillableEvent, error) {
	b, err := ber.EventJSON()
	if err != nil {
		return nil, err
	}
	var event BillableEvent
	if err := json.Unmarshal(b, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// Reporter enables fetching BillableEvents for building reports/bills
type Reporter struct {
	db *sql.DB
}

func New(db *sql.DB) *Reporter {
	return &Reporter{db: db}
}

func (r *Reporter) queryJSON(sql string, args ...interface{}) (*sql.Rows, error) {
	return r.db.Query(fmt.Sprintf(`
		with q as ( %s )
		select row_to_json(q.*) from q;
	`, sql), args...)
}

// GetBillableEventRows returns a handle to a resultset of BillableEvents. Use
// this to iterate over rows without buffering all into memory
func (r *Reporter) GetBillableEventRows(filter EventFilter) (*BillableEventRows, error) {
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
	rows, err := r.queryJSON(fmt.Sprintf(`
		with
		components_with_price as (
			select
				event_guid,
				resource_guid,
				resource_name,
				resource_type,
				org_guid,
				space_guid,
				plan_guid,
				plan_name,
				duration * $1::tstzrange as duration,
				number_of_nodes,
				memory_in_mb,
				storage_in_mb,
				component_name,
				component_formula, 
				currency_code,
				currency_rate,
				vat_code,
				vat_rate,
				greatest(0.01, eval_formula(
					memory_in_mb,
					storage_in_mb,
					number_of_nodes,
					duration * $1::tstzrange,
					component_formula
				) * currency_rate) as price_ex_vat
			from
				billable_event_components
			where
				duration && $1::tstzrange
				%s
			order by
				lower(duration * $1::tstzrange) asc
		)
		select
			event_guid,
			to_json(min(lower(duration))) as event_start,
			to_json(max(upper(duration))) as event_stop,
			resource_guid,
			resource_name,
			resource_type,
			org_guid,
			space_guid,
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
			space_guid,
			plan_guid,
			number_of_nodes,
			memory_in_mb,
			storage_in_mb
	`, filterQuery), args...)
	if err != nil {
		return nil, err
	}
	return &BillableEventRows{rows}, nil
}

// GetBillableEvents returns a slice of billable events for the given filter.
// Due to the large number of results that can be returned it is recormended
// you use the GetBillableEventRows version to avoid buffering everything into
// memory
func (r *Reporter) GetBillableEvents(filter EventFilter) ([]*BillableEvent, error) {
	rows, err := r.GetBillableEventRows(filter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []*BillableEvent{}
	for rows.Next() {
		ev, err := rows.Event()
		if err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

type EventFilter struct {
	RangeStart string
	RangeStop  string
	OrgGUIDs   []string
}

func (filter *EventFilter) Validate() error {
	if filter.RangeStart == "" {
		return fmt.Errorf(`a range start filter value is required`)
	}
	if filter.RangeStop == "" {
		return fmt.Errorf(`a range stop filter value is required`)
	}
	return nil
}
