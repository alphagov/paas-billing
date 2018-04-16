package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

//go:generate counterfeiter -o fakes/fake_event_storer.go . EventStorer
type EventStorer interface {
	StoreEvents(events []RawEvent) error
	GetEvents(filter RawEventFilter) ([]RawEvent, error)
}

type RawEventFilter struct {
	Reverse bool
	Limit   int
	Kind    string
}

type RawEvent struct {
	GUID       string          `json:"guid"`
	Kind       string          `json:"kind"`
	RawMessage json.RawMessage `json:"raw_message"`
	CreatedAt  time.Time       `json:"created_at"`
}

func (e *RawEvent) Validate() error {
	if e.GUID == "" {
		return fmt.Errorf("events must have a GUID")
	}
	if e.Kind == "" {
		return fmt.Errorf("events must have a Kind")
	}
	if e.CreatedAt.IsZero() {
		return fmt.Errorf("events must have a CreatedAt time")
	}
	if string(e.RawMessage) == "" {
		return fmt.Errorf("events must have a RawMessage payload")
	}
	return nil
}

type Beginer interface {
	Begin() (*sql.Tx, error)
}
