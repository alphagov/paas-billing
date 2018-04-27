package eventio

type RawEventWriter interface {
	StoreEvents(events []RawEvent) error
}

type RawEventReader interface {
	GetEvents(filter RawEventFilter) ([]RawEvent, error)
}

type EventStore interface {
	Init() error
	Refresh() error
	RawEventWriter
	RawEventReader
	UsageEventReader
	BillableEventReader
	BillableEventForecaster
}
