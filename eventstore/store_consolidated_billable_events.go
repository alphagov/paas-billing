package eventstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
)

const (
	DefaultConsolidationStartDate = "2017-07-01"
)

func (s *EventStore) GetConsolidatedBillableEventRows(ctx context.Context, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	rows, err := s.getConsolidatedBillableEventRows(tx, filter)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return rows, nil
}

func (s *EventStore) getConsolidatedBillableEventRows(tx *sql.Tx, filter eventio.EventFilter) (eventio.BillableEventRows, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	err := checkMonthBoundary(filter.RangeStart)
	if err != nil {
		return nil, err
	}
	err = checkMonthBoundary(filter.RangeStop)
	if err != nil {
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

	startTime := time.Now()
	rows, err := queryJSON(tx, fmt.Sprintf(`
		select
			event_guid,
			lower(duration) as event_start,
			upper(duration) as event_stop,
			resource_guid,
			resource_name,
			resource_type,
			org_guid,
			org_name,
			space_guid,
			space_name,
			plan_guid,
			quota_definition_guid,
			number_of_nodes,
			memory_in_mb,
			storage_in_mb,
			price
		from
			consolidated_billable_events
 		where
			consolidated_range && $1::tstzrange
			%s
		order by event_guid
	`, filterQuery), args...)
	elapsed := time.Since(startTime)
	eventStorePerformanceGauge.WithLabelValues("getConsolidatedBillableEventRows", "").Set(elapsed.Seconds())
	if err != nil {
		s.logger.Error("get-consolidated-billable-event-rows-query", err, lager.Data{
			"filter":  filter,
			"elapsed": int64(elapsed),
		})
		return nil, err
	}
	s.logger.Info("get-consolidated-billable-event-rows-query", lager.Data{
		"filter":  filter,
		"elapsed": int64(elapsed),
	})
	return &BillableEventRows{rows}, nil
}

func (s *EventStore) GetConsolidatedBillableEvents(filter eventio.EventFilter) ([]eventio.BillableEvent, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := s.GetConsolidatedBillableEventRows(ctx, filter)
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

func (s *EventStore) IsRangeConsolidated(filter eventio.EventFilter) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	result, err := s.isRangeConsolidated(tx, filter)
	if err != nil {
		tx.Rollback()
		return false, err
	}
	return result, tx.Commit()
}

func (s *EventStore) isRangeConsolidated(tx *sql.Tx, filter eventio.EventFilter) (bool, error) {
	if err := filter.Validate(); err != nil {
		return false, err
	}
	startTime := time.Now()
	rows, err := tx.Query(
		"SELECT 1 FROM consolidation_history where consolidated_range=$1::tstzrange",
		fmt.Sprintf("[%s, %s)", filter.RangeStart, filter.RangeStop),
	)
	elapsed := time.Since(startTime)
	if err != nil {
		eventStorePerformanceGauge.WithLabelValues("isRangeConsolidated", err.Error()).Set(elapsed.Seconds())
		s.logger.Error("is-range-consolidated-query", err, lager.Data{
			"filter":  filter,
			"elapsed": int64(elapsed),
		})
		return false, err
	}
	eventStorePerformanceGauge.WithLabelValues("isRangeConsolidated", "").Set(elapsed.Seconds())
	s.logger.Info("is-range-consolidated-query", lager.Data{
		"filter":  filter,
		"elapsed": int64(elapsed),
	})
	defer rows.Close()
	return rows.Next(), nil
}

func (s *EventStore) ConsolidateAll() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	err = s.consolidateAll(tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *EventStore) consolidateAll(tx *sql.Tx) error {
	startAt := os.Getenv("CONSOLIDATION_START_DATE")
	if startAt == "" {
		startAt = DefaultConsolidationStartDate
	}
	endAt := os.Getenv("CONSOLIDATION_END_DATE")
	if endAt == "" {
		endAt = time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	}
	return s.consolidateFullMonths(tx, startAt, endAt)
}

func (s *EventStore) ConsolidateFullMonths(startAt string, endAt string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	err = s.consolidateFullMonths(tx, startAt, endAt)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *EventStore) consolidateFullMonths(tx *sql.Tx, startAt string, endAt string) error {
	eventFilter := eventio.EventFilter{
		RangeStart: startAt,
		RangeStop:  endAt,
	}
	truncatedEventFilter, err := eventFilter.TruncateMonth()
	if err != nil {
		return err
	}

	s.logger.Info("consolidating-full-months", lager.Data{
		"start": truncatedEventFilter.RangeStart,
		"stop":  truncatedEventFilter.RangeStop,
	})

	monthFilters, err := truncatedEventFilter.SplitByMonth()
	if err != nil {
		return err
	}
	for _, filter := range monthFilters {
		isConsolidated, err := s.isRangeConsolidated(tx, filter)
		if err != nil {
			return err
		}
		if !isConsolidated {
			s.logger.Info("consolidating-months", lager.Data{
				"start": filter.RangeStart,
				"stop":  filter.RangeStop,
			})
			err = s.consolidate(tx, filter)
			if err != nil {
				return err
			}
		}
	}

	s.logger.Info("consolidated-full-months", lager.Data{
		"start": truncatedEventFilter.RangeStart,
		"stop":  truncatedEventFilter.RangeStop,
	})

	return nil
}

func (s *EventStore) Consolidate(filter eventio.EventFilter) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	err = s.consolidate(tx, filter)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *EventStore) consolidate(tx *sql.Tx, filter eventio.EventFilter) error {
	if len(filter.OrgGUIDs) != 0 {
		return fmt.Errorf("consolidate must be called without an organisations filter (i.e. for all orgs)")
	}

	startTime := time.Now()
	_, err := tx.Exec(`
				insert into consolidation_history (
					consolidated_range,
					created_at
				) values (
					$1::tstzrange,
					$2::timestamptz
				)`,
		fmt.Sprintf("[%s, %s)", filter.RangeStart, filter.RangeStop),
		time.Now())
	elapsed := time.Since(startTime)
	eventStorePerformanceGauge.WithLabelValues("consolidate:history", "").Set(elapsed.Seconds())
	if err != nil {
		s.logger.Error("consolidation-history-query", err, lager.Data{
			"filter":  filter,
			"elapsed": int64(elapsed),
		})
		return err
	}
	s.logger.Info("consolidation-history-query", lager.Data{
		"filter":  filter,
		"elapsed": int64(elapsed),
	})

	query, args, err := WithBillableEvents(`
			insert into consolidated_billable_events (
				consolidated_range,

				event_guid,
				duration,
				resource_guid,
				resource_name,
				resource_type,
				org_guid,
				org_name,
				space_guid,
				space_name,
				plan_guid,
				quota_definition_guid,
				number_of_nodes,
				memory_in_mb,
				storage_in_mb,
				price
			)
			select
				filtered_range,

				billable_events.event_guid,
				tstzrange(billable_events.event_start, billable_events.event_stop),
				billable_events.resource_guid,
				billable_events.resource_name,
				billable_events.resource_type,
				billable_events.org_guid,
				billable_events.org_name,
				billable_events.space_guid,
				billable_events.space_name,
				billable_events.plan_guid,
				billable_events.quota_definition_guid,
				billable_events.number_of_nodes,
				billable_events.memory_in_mb,
				billable_events.storage_in_mb,
				billable_events.price
			from
				billable_events,
				filtered_range
		`,
		filter,
	)
	if err != nil {
		return err
	}

	startTime = time.Now()
	_, err = tx.Exec(query, args...)
	elapsed = time.Since(startTime)
	eventStorePerformanceGauge.WithLabelValues("consolidate:insert", "").Set(elapsed.Seconds())
	if err != nil {
		s.logger.Error("consolidation-insert-query", err, lager.Data{
			"filter":  filter,
			"elapsed": int64(elapsed),
		})
		return err
	}
	s.logger.Info("consolidation-insert-query", lager.Data{
		"filter":  filter,
		"elapsed": int64(elapsed),
	})

	return nil
}

func checkMonthBoundary(value string) error {
	rangeStart, err := time.Parse("2006-01-02", value)
	if err != nil {
		return err
	}
	rangeStartMonth := time.Date(rangeStart.Year(), rangeStart.Month(), 1, 0, 0, 0, 0, rangeStart.Location())
	if rangeStartMonth != rangeStart {
		return fmt.Errorf("consolidation only works with ranges starting and ending on month boundaries")
	}
	return nil
}

type CachedBillableEventRows struct {
}
