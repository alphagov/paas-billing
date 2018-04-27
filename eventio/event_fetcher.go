package eventio

import "context"

type EventFetcher interface {
	FetchEvents(ctx context.Context, lastKnownEvent *RawEvent) ([]RawEvent, error)
	Kind() string
}
