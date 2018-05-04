package eventio

import (
	"fmt"
)

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

type PricingPlanFilter struct {
	RangeStart string
	RangeStop  string
}

func (filter *PricingPlanFilter) Validate() error {
	if filter.RangeStart == "" {
		return fmt.Errorf(`a range start filter value is required`)
	}
	if filter.RangeStop == "" {
		return fmt.Errorf(`a range stop filter value is required`)
	}
	return nil
}
