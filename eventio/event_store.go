package eventio

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

//counterfeiter:generate . EventStore
type EventStore interface {
	Init() error
	Refresh() error
	Ping() error
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
}
