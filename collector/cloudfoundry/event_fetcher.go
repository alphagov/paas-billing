package cloudfoundry

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/collector"
	"github.com/alphagov/paas-billing/store"
)

const (
	DefaultFetchLimit   = 50
	DefaultRecordMinAge = 5 * time.Minute
)

var _ collector.EventFetcher = &EventFetcher{}

type EventFetcher struct {
	Client       cloudfoundry.UsageEventsAPI
	Logger       lager.Logger
	RecordMinAge time.Duration
	FetchLimit   int
}

func (e *EventFetcher) FetchEvents(lastEvent *store.RawEvent) ([]store.RawEvent, error) {
	guid := cloudfoundry.GUIDNil
	if lastEvent != nil {
		if lastEvent.GUID == "" {
			return nil, fmt.Errorf("invalid GUID for lastEvent")
		}
		guid = lastEvent.GUID
	}

	fetchLimit := e.FetchLimit
	if fetchLimit < 1 {
		fetchLimit = DefaultFetchLimit
	}
	if fetchLimit < 1 || fetchLimit > 100 {
		return nil, fmt.Errorf("FetchLimit must be between 1 and 100")
	}
	recordMinAge := e.RecordMinAge
	if e.RecordMinAge == 0 {
		recordMinAge = DefaultRecordMinAge
	}
	if recordMinAge < DefaultRecordMinAge {
		return nil, fmt.Errorf("RecordMinAge should be at least 5m to reduce the risk of late arriving events being skipped")
	}

	usageEvents, err := e.Client.Get(guid, fetchLimit, recordMinAge)
	if err != nil {
		return nil, err
	}
	events := []store.RawEvent{}
	if usageEvents != nil {
		for _, usageEvent := range usageEvents.Resources {
			events = append(events, store.RawEvent{
				GUID:       usageEvent.MetaData.GUID,
				Kind:       e.Client.Type(),
				CreatedAt:  usageEvent.MetaData.CreatedAt,
				RawMessage: usageEvent.EntityRaw,
			})
		}
	}
	e.Logger.Info("fetched", lager.Data{
		"last_guid":   guid,
		"event_count": len(events),
	})

	return events, nil
}

func (e *EventFetcher) Kind() string {
	return e.Client.Type()
}
