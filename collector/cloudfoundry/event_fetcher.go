package cloudfoundry

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/collector"
	"github.com/alphagov/paas-billing/db"
)

var _ collector.EventFetcher = &EventFetcher{}

type EventFetcher struct {
	sqlClient db.SQLClient
	client    cloudfoundry.UsageEventsAPI
}

func NewEventFetcher(sqlClient db.SQLClient, client cloudfoundry.UsageEventsAPI) *EventFetcher {
	return &EventFetcher{
		sqlClient: sqlClient,
		client:    client,
	}
}

func (e *EventFetcher) FetchEvents(logger lager.Logger, fetchLimit int, recordMinAge time.Duration) (int, error) {
	tableName := fmt.Sprintf("%s_usage_events", e.client.Type())

	guid, err := e.sqlClient.FetchLastGUID(tableName)
	if err != nil {
		return 0, err
	}

	usageEvents, err := e.client.Get(guid, fetchLimit, recordMinAge)
	if err != nil {
		return 0, err
	}
	cnt := len(usageEvents.Resources)
	logger.Info("fetch", lager.Data{"last_guid": guid, "record_count": cnt})

	if cnt > 0 {
		if err := e.sqlClient.InsertUsageEventList(usageEvents, tableName); err != nil {
			return 0, err
		}
	}

	return cnt, nil
}

func (e *EventFetcher) Name() string {
	return fmt.Sprintf("cf-%s-usage-event-collector", e.client.Type())
}
