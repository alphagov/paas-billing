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

func ParseDate (dateString string) (time.Time, error){
	var dateFormats [6]string
	dateFormats = [6]string {
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02"}
	var date time.Time
	for _, dateFormat := range dateFormats{
		date , _ = time.Parse(dateFormat, dateString)
		if ! date.IsZero() {
			break
		}
	}

	if date.IsZero() {
		return date , fmt.Errorf("Could not parse date %s", dateString )
	} else {
		return date, nil
	}
}


func (filter *EventFilter) SplitByMonth() ([]EventFilter, error) {

	start, err := ParseDate(filter.RangeStart)
	if err != nil {
		return nil, err
	}
	end, err := ParseDate(filter.RangeStop)
	if err != nil {
		return nil, err
	}

	return filter.recursiveSplitByMonth(start, end), nil
}

func (filter *EventFilter) recursiveSplitByMonth(t1, t2 time.Time) []EventFilter {
	dateFormat := "2006-01-02T15:04:05Z"

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
	start, err := ParseDate(filter.RangeStart)
	if err != nil {
		return *filter, err
	}
	stop, err := ParseDate(filter.RangeStop)
	if err != nil {
		return *filter, err
	}

	return EventFilter{
		RangeStart: truncateMonth(start).Format("2006-01-02T15:04:05Z"),
		RangeStop:  truncateMonth(stop).Format("2006-01-02T15:04:05Z"),
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

type TimeRangeFilter struct {
	RangeStart string
	RangeStop  string
}

func (filter *TimeRangeFilter) Validate() error {
	if err := validateDateString("start", filter.RangeStart); err != nil {
		return err
	}
	if err := validateDateString("end", filter.RangeStop); err != nil {
		return err
	}
	return nil
}

func validateDateString(name string, value string) error {
	if _, err := ParseDate(value); err != nil {
		return fmt.Errorf(`a valid range %s filter value is required - expected format 2006-01-02 [15:04] [:05] [Z] - got %s`,
			name, value,
		)
	}
	return nil
}
