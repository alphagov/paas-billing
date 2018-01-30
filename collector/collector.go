package collector

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/db"
	"github.com/pkg/errors"
)

const maxFetchLimit = 100

// Collector is the usage events collector
//
// It periodically calls the Cloud Foundry API for app and service usage events and persists these in the database
type Collector struct {
	sqlClient       db.SQLClient
	appClient       cloudfoundry.UsageEventsAPI
	serviceClient   cloudfoundry.UsageEventsAPI
	logger          lager.Logger
	signalChan      chan os.Signal
	appTimer        *time.Timer
	serviceTimer    *time.Timer
	defaultSchedule time.Duration
	minWaitTime     time.Duration
	fetchLimit      int
	recordMinAge    time.Duration
}

// New creates a new usage events collector
func New(
	appClient cloudfoundry.UsageEventsAPI,
	serviceClient cloudfoundry.UsageEventsAPI,
	sqlClient db.SQLClient,
	config *Config,
	logger lager.Logger,
) (*Collector, error) {
	collector := &Collector{
		sqlClient:       sqlClient,
		appClient:       appClient,
		serviceClient:   serviceClient,
		logger:          logger,
		defaultSchedule: time.Duration(1 * time.Minute),
		minWaitTime:     time.Duration(3 * time.Second),
		fetchLimit:      50,
		recordMinAge:    time.Duration(5 * time.Minute),
	}
	if err := collector.applyConfig(config); err != nil {
		return nil, err
	}
	return collector, nil
}

func (c *Collector) applyConfig(config *Config) error {
	var err error
	if config.DefaultSchedule != "" {
		if c.defaultSchedule, err = time.ParseDuration(config.DefaultSchedule); err != nil {
			return errors.Wrap(err, "DefaultSchedule is invalid")
		}
	}

	if config.MinWaitTime != "" {
		if c.minWaitTime, err = time.ParseDuration(config.MinWaitTime); err != nil {
			return errors.Wrap(err, "MinWaitTime is invalid")
		}
	}

	if config.RecordMinAge != "" {
		if c.recordMinAge, err = time.ParseDuration(config.RecordMinAge); err != nil {
			return errors.Wrap(err, "RecordMinAge is invalid")
		}
	}

	if config.FetchLimit != "" {
		if c.fetchLimit, err = strconv.Atoi(config.FetchLimit); err != nil {
			return errors.Wrap(err, "FetchLimit is invalid")
		}
		if c.fetchLimit <= 0 || c.fetchLimit > maxFetchLimit {
			return errors.New("FetchLimit must be between 1 and 100")
		}
	}

	return nil
}

// Run is the main application loop
func (c *Collector) Run(ctx context.Context) {
	c.logger.Info("start")

	c.appTimer = time.NewTimer(time.Second)
	c.serviceTimer = time.NewTimer(time.Second)

	for {
		select {
		case <-c.appTimer.C:
			c.fetchUsageEvents(c.appClient, c.appTimer)
		case <-c.serviceTimer.C:
			c.fetchUsageEvents(c.serviceClient, c.serviceTimer)
		case <-ctx.Done():
			c.logger.Info("exiting")
			c.appTimer.Stop()
			c.serviceTimer.Stop()
			return
		}
	}
}

func (c *Collector) fetchUsageEvents(client cloudfoundry.UsageEventsAPI, timer *time.Timer) {
	logAction := fmt.Sprintf("fetch-%s-usage-events", client.Type())
	tableName := fmt.Sprintf("%s_usage_events", client.Type())

	guid, err := c.sqlClient.FetchLastGUID(tableName)
	if err != nil {
		c.logger.Error(logAction, err)
		timer.Reset(c.defaultSchedule)
		return
	}

	usageEvents, err := client.Get(guid, c.fetchLimit, c.recordMinAge)
	if err != nil {
		c.logger.Error(logAction, err)
		timer.Reset(c.defaultSchedule)
		return
	}
	cnt := len(usageEvents.Resources)
	c.logger.Info(logAction, lager.Data{"last_guid": guid, "record_count": cnt})

	if cnt > 0 {
		if err := c.sqlClient.InsertUsageEventList(usageEvents, tableName); err != nil {
			c.logger.Error(logAction, err)
			timer.Reset(c.defaultSchedule)
			return
		}
	}

	if cnt == c.fetchLimit {
		timer.Reset(c.minWaitTime)
	} else {
		timer.Reset(c.defaultSchedule)
	}
}
