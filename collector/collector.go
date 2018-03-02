package collector

import (
	"context"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter -o fakes/fake_event_fetcher.go . EventFetcher
type EventFetcher interface {
	Name() string
	FetchEvents(logger lager.Logger, fetchLimit int, recordMinAge time.Duration) (int, error)
}

// Collector is the usage events collector
//
// It periodically calls the Cloud Foundry API for app and service usage events and persists these in the database
type Collector struct {
	config       *Config
	logger       lager.Logger
	eventFetcher EventFetcher
	signalChan   chan os.Signal
}

// New creates a new usage events collector
func New(
	config *Config,
	logger lager.Logger,
	eventFetcher EventFetcher,
) *Collector {
	collector := &Collector{
		config:       config,
		logger:       logger.Session(eventFetcher.Name()),
		eventFetcher: eventFetcher,
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
			cnt, err := c.eventFetcher.FetchEvents(c.logger, c.config.FetchLimit, c.config.RecordMinAge)
			if err != nil {
				c.logger.Error("fetch-events", err)
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
