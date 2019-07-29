package eventcollector

import (
	"context"
	"sync"
	"time"

	"github.com/alphagov/paas-billing/eventio"

	"code.cloudfoundry.org/lager"
)

const (
	DefaultSchedule = time.Duration(15 * time.Minute)
)

type state string

const (
	// Syncing state means that the collector has just started and need to immediately do a fetch to catchup
	Syncing state = "sync"
	// Scheduled means that we have caught up with latest events and are scheduled to run again later in Schedule time
	Scheduled state = "waiting"
	// Collecting means the collector thinks it probably has more to collect but is rate limited by MinWaitTime
	Collecting state = "collecting"
)

// EventCollector periodically fetches events via the given EventFetcher and
// stores them to the given EventStore
type EventCollector struct {
	state           state
	schedule        time.Duration
	minWaitTime     time.Duration
	initialWaitTime time.Duration
	logger          lager.Logger
	fetcher         eventio.EventFetcher
	store           eventio.EventStore
	mu              sync.Mutex
	eventsCollected int
}

// Run executes collect periodically the rate is dictated by Schedule and MinWaitTime
func (c *EventCollector) Run(ctx context.Context) error {
	c.logger.Info("started")
	defer c.logger.Info("stopping")
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		c.logger.Info("status", lager.Data{
			"state":            c.state,
			"kind":             c.fetcher.Kind(),
			"next_collection":  c.waitDuration().String(),
			"events_collected": c.eventsCollected,
		})
		select {
		case <-time.After(c.waitDuration()):
			startTime := time.Now()
			collectedEvents, err := c.collect(ctx)
			if err != nil {
				c.state = Scheduled
				c.logger.Error("collect-error", err)
				continue
			}
			c.eventsCollected += len(collectedEvents)
			elapsed := time.Since(startTime)
			c.logger.Info("collected", lager.Data{
				"count":          len(collectedEvents),
				"kind":           c.fetcher.Kind(),
				"elapsed":        elapsed.String(),
				"elapsed_millis": string(int64(elapsed / time.Millisecond)),
			})
		case <-ctx.Done():
			return nil
		}
	}
}

// collect reads a batch of RawEvents from the EventFetcher and writes them to the EventStore
func (c *EventCollector) collect(ctx context.Context) ([]eventio.RawEvent, error) {
	lastEvent, err := c.getLastEvent()
	if err != nil {
		return nil, err
	}
	events, err := c.fetcher.FetchEvents(ctx, lastEvent)
	if err != nil {
		return nil, err
	}
	c.logger.Info("collecting", lager.Data{
		"kind": c.fetcher.Kind(),
		"after_guid": func() string {
			if lastEvent == nil {
				return ""
			}
			return lastEvent.GUID
		}(),
	})
	if err := c.store.StoreEvents(events); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		c.state = Scheduled
	} else if len(events) > 0 && lastEvent != nil && events[len(events)-1].GUID == lastEvent.GUID {
		c.state = Scheduled
	} else {
		c.state = Collecting
	}
	return events, nil
}

// getLastEvent returns the latest event of the same kind as the fetcher or nil if no events
func (c *EventCollector) getLastEvent() (*eventio.RawEvent, error) {
	lastEvents, err := c.store.GetEvents(eventio.RawEventFilter{
		Kind:  c.fetcher.Kind(),
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

// wait returns a channel that closes after the collection schedule time has elapsed
func (c *EventCollector) waitDuration() time.Duration {
	delay := c.schedule
	if c.state == Syncing {
		delay = c.initialWaitTime
	}
	if c.state == Collecting {
		delay = c.minWaitTime
	}
	return delay
}

type Config struct {
	Schedule        time.Duration
	MinWaitTime     time.Duration
	InitialWaitTime time.Duration
	Logger          lager.Logger
	Fetcher         eventio.EventFetcher
	Store           eventio.EventStore
}

func New(cfg Config) *EventCollector {
	if cfg.Logger == nil {
		cfg.Logger = lager.NewLogger("collector")
	}
	return &EventCollector{
		schedule:    cfg.Schedule,
		minWaitTime: cfg.MinWaitTime,
		logger:      cfg.Logger,
		fetcher:     cfg.Fetcher,
		store:       cfg.Store,
		state:       Syncing,
	}
}
