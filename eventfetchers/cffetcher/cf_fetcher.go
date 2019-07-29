package cffetcher

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

const (
	DefaultRecordMinAge = 5 * time.Minute
	DefaultFetchLimit   = 50
)

type Kind string

const (
	App     Kind = "app"
	Service Kind = "service"
)

var _ eventio.EventFetcher = &CFEventFetcher{}

// CFEventFetcher is an EventFetcher that fetches cloudfoundry App or Service usage events
type CFEventFetcher struct {
	client       UsageEventsAPI
	logger       lager.Logger
	recordMinAge time.Duration
	fetchLimit   int
}

// FetchEvents requests
func (e *CFEventFetcher) FetchEvents(ctx context.Context, lastEvent *eventio.RawEvent) ([]eventio.RawEvent, error) {
	guid := GUIDNil
	if lastEvent != nil {
		if lastEvent.GUID == "" {
			return nil, fmt.Errorf("invalid GUID for lastEvent")
		}
		guid = lastEvent.GUID
	}

	fetchLimit := e.fetchLimit
	if fetchLimit < 1 {
		fetchLimit = DefaultFetchLimit
	}
	if fetchLimit < 1 || fetchLimit > 100 {
		return nil, fmt.Errorf("FetchLimit must be between 1 and 100")
	}
	recordMinAge := e.recordMinAge
	if e.recordMinAge == 0 {
		recordMinAge = DefaultRecordMinAge
	}
	if recordMinAge < DefaultRecordMinAge {
		return nil, fmt.Errorf("RecordMinAge should be at least 5m to reduce the risk of late arriving events being skipped")
	}

	e.logger.Info("fetching", lager.Data{
		"after_guid": guid,
		"limit":      fetchLimit,
	})
	startTime := time.Now()
	usageEvents, err := e.client.Get(guid, fetchLimit, recordMinAge)
	if err != nil {
		return nil, err
	}
	events := []eventio.RawEvent{}
	if usageEvents != nil {
		for _, usageEvent := range usageEvents.Resources {
			events = append(events, eventio.RawEvent{
				GUID:       usageEvent.MetaData.GUID,
				Kind:       e.client.Type(),
				CreatedAt:  usageEvent.MetaData.CreatedAt,
				RawMessage: usageEvent.EntityRaw,
			})
		}
	}
	elapsed := time.Since(startTime)
	e.logger.Info("fetched", lager.Data{
		"last_guid":   guid,
		"event_count": len(events),
		"elapsed":     int64(elapsed),
	})

	return events, nil
}

// Kind returns the type of event this fetcher returns
func (e *CFEventFetcher) Kind() string {
	return e.client.Type()
}

// FetcherConfig allows tuning of the fetcher. You must set a Type and ClientConfig
type Config struct {
	// Type sets the Kind of event to collect, must be App or Service
	Type Kind
	// ClientConfig allows configuration of connection to CF API
	ClientConfig *cfclient.Config
	// Client overrides the default client used to query API
	Client UsageEventsAPI
	// Logger overrides the default logger
	Logger lager.Logger
	// RecordMinAge sets the age at which events are mature enough for collection
	RecordMinAge time.Duration
	// FetchLimit dictates the max number of events returned in each FetchEvents call
	FetchLimit int
}

// New creates a new CFEventFetcher for the given config
func New(cfg Config) (*CFEventFetcher, error) {
	if cfg.Logger == nil {
		cfg.Logger = lager.NewLogger("cf-fetcher")
	}
	if cfg.Client == nil {
		if cfg.ClientConfig == nil {
			return nil, fmt.Errorf("cffetcher.New: must supply cfclient.Config")
		}
		cf, err := cfclient.NewClient(cfg.ClientConfig)
		if err != nil {
			return nil, err
		}
		apiEngine := &client{cf}
		switch cfg.Type {
		case App:
			cfg.Client = NewAppUsageEventsAPI(apiEngine, cfg.Logger)
		case Service:
			cfg.Client = NewServiceUsageEventsAPI(apiEngine, cfg.Logger)
		default:
			return nil, fmt.Errorf("missing or unknown FetcherConfig.Type")
		}
	}
	fetcher := &CFEventFetcher{
		client:       cfg.Client,
		logger:       cfg.Logger.Session(fmt.Sprintf("%s-event-fetcher", cfg.Client.Type())),
		fetchLimit:   cfg.FetchLimit,
		recordMinAge: cfg.RecordMinAge,
	}
	return fetcher, nil
}
