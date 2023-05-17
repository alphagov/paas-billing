package eventio

import "context"

//counterfeiter:generate . EventFetcher
type EventFetcher interface {
	FetchEvents(ctx context.Context, lastKnownEvent *RawEvent) ([]RawEvent, error)
	Kind() string
}
