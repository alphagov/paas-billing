package eventio

import (
	"fmt"
	"time"
)

type EventFilter struct {
	RangeStart string
	RangeStop  string
	OrgGUIDs   []string
}

func (filter *EventFilter) SplitByMonth() ([]EventFilter, error) {
	dateFormat := "2006-01-02"

	start, err := time.Parse(dateFormat, filter.RangeStart)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(dateFormat, filter.RangeStop)
	if err != nil {
		return nil, err
	}

	return filter.recursiveSplitByMonth(start, end), nil
}

func (filter *EventFilter) recursiveSplitByMonth(t1, t2 time.Time) []EventFilter {
	dateFormat := "2006-01-02"

	if !t1.Before(t2) {
		return []EventFilter{}
	} else {
		next := truncateMonth(t1.AddDate(0, 1, 0))
		return append(
			[]EventFilter{
				{
					RangeStart: t1.Format(dateFormat),
					RangeStop:  minDate(t2, next).Format(dateFormat),
					OrgGUIDs:   filter.OrgGUIDs,
				},
			},
			filter.recursiveSplitByMonth(next, t2)...,
		)
	}
}

func truncateMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func minDate(t1 time.Time, t2 time.Time) time.Time {
	if t1.Before(t2) {
		return t1
	} else {
		return t2
	}
}

func (filter *EventFilter) TruncateMonth() (EventFilter, error) {
	start, err := time.Parse("2006-01-02", filter.RangeStart)
	if err != nil {
		return *filter, err
	}
	stop, err := time.Parse("2006-01-02", filter.RangeStop)
	if err != nil {
		return *filter, err
	}

	return EventFilter{
		RangeStart: truncateMonth(start).Format("2006-01-02"),
		RangeStop:  truncateMonth(stop).Format("2006-01-02"),
		OrgGUIDs:   filter.OrgGUIDs,
	}, nil
}

func (filter *EventFilter) Validate() error {
	if err := validateDateString("start", filter.RangeStart); err != nil {
		return err
	}
	if err := validateDateString("end", filter.RangeStop); err != nil {
		return err
	}
	return nil
}

func (filter *PricingPlanFilter) Validate() error {
	if err := validateDateString("start", filter.RangeStart); err != nil {
		return err
	}
	if err := validateDateString("end", filter.RangeStop); err != nil {
		return err
	}
	return nil
}

func validateDateString(name string, value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf(
			`a valid range %s filter value is required - expected format 2006-01-02 - got %s`,
			name, value,
		)
	}
	return nil
}

type PricingPlanFilter struct {
	RangeStart string
	RangeStop  string
}
