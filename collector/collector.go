package collector

import (
	"context"
	"time"

	"github.com/alphagov/paas-billing/store"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/fake_event_fetcher.go . EventFetcher
type EventFetcher interface {
	FetchEvents(lastKnownEvent *store.RawEvent) ([]store.RawEvent, error)
	Kind() string
}

// Collector is the usage events collector
//
// It periodically calls the Cloud Foundry API for app and service usage events and persists these in the database
type Collector struct {
	config       *Config
	logger       lager.Logger
	eventFetcher EventFetcher
	store        store.EventStorer
}

// New creates a new usage events collector
func New(
	config *Config,
	logger lager.Logger,
	eventFetcher EventFetcher,
	eventStore store.EventStorer,
) *Collector {
	collector := &Collector{
		config:       config,
		logger:       logger.Session(eventFetcher.Kind()),
		eventFetcher: eventFetcher,
		store:        eventStore,
	}
	return collector
}

// Run is the main application loop
func (c *Collector) Run(ctx context.Context) {
	c.logger.Info("start")

	timer := time.NewTimer(time.Second)

	for {
		select {
		case <-timer.C:
			cnt, err := c.collect()
			if err != nil {
				c.logger.Error("collect", err)
			}
			if cnt == c.config.FetchLimit {
				timer.Reset(c.config.MinWaitTime)
			} else {
				timer.Reset(c.config.DefaultSchedule)
			}
		case <-ctx.Done():
			c.logger.Info("stop")
			timer.Stop()
			return
		}
	}
}

func (c *Collector) getLastEvent() (*store.RawEvent, error) {
	lastEvents, err := c.store.GetEvents(store.RawEventFilter{
		Kind:  c.eventFetcher.Kind(),
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(lastEvents) < 1 {
		return nil, nil
	}
	return &lastEvents[0], nil
}

func (c *Collector) collect() (int, error) {
	lastEvent, err := c.getLastEvent()
	if err != nil {
		return 0, err
	}
	events, err := c.eventFetcher.FetchEvents(lastEvent)
	if err != nil {
		return 0, err
	}
	if err := c.store.StoreEvents(events); err != nil {
		return 0, err
	}
	return len(events), nil
}
