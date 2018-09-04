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
