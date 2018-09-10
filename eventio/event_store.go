package eventio

type RawEventWriter interface {
	StoreEvents(events []RawEvent) error
}

type RawEventReader interface {
	GetEvents(filter RawEventFilter) ([]RawEvent, error)
}

type PricingPlanReader interface {
	GetPricingPlans(filter PricingPlanFilter) ([]PricingPlan, error)
}

type EventStore interface {
	Init() error
	Refresh() error
	PricingPlanReader
	RawEventWriter
	RawEventReader
	UsageEventReader
	BillableEventReader
	BillableEventForecaster
	ConsolidatedBillableEventReader
	BillableEventConsolidator
}
