package eventio

import (
	"time"
	"context"
)

type RawEventWriter interface {
	StoreEvents(events []RawEvent) error
}

type RawEventReader interface {
	GetEvents(filter RawEventFilter) ([]RawEvent, error)
}

type PricingPlanReader interface {
	GetPricingPlans(filter TimeRangeFilter) ([]PricingPlan, error)
}

type CurrencyRateReader interface {
	GetCurrencyRates(filter TimeRangeFilter) ([]CurrencyRate, error)
}

type VATRateReader interface {
	GetVATRates(filter TimeRangeFilter) ([]VATRate, error)
}

type EventStore interface {
	Init() error
	Refresh() error
	PricingPlanReader
	CurrencyRateReader
	VATRateReader
	RawEventWriter
	RawEventReader
	UsageEventReader
	TotalCostReader
	BillableEventReader
	BillableEventForecaster
	ConsolidatedBillableEventReader
	BillableEventConsolidator
	UpdateResources(context.Context,time.Time) (int,error)
}
