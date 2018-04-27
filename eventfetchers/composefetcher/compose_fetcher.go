package composefetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/eventio"
	composeapi "github.com/compose/gocomposeapi"
)

var _ eventio.EventFetcher = &ComposeEventFetcher{}

const (
	TimePrecisionOffset = -1 * time.Second
	DefaultFetchLimit   = 50
	Kind                = "compose"
)

const (
	DeploymentScaleMembersEvent = "deployment.scale.members"
)

type ComposeEventFetcher struct {
	logger     lager.Logger
	client     ComposeClient
	fetchLimit int
}

func (e *ComposeEventFetcher) fetchScaleEvents(ctx context.Context, newerThan *time.Time) ([]eventio.RawEvent, error) {
	events := make([]eventio.RawEvent, 0)
	limit := DefaultFetchLimit
	if e.fetchLimit != 0 {
		limit = e.fetchLimit
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("FetchLimit must be between 1 and 100")
	}
	params := composeapi.AuditEventsParams{
		NewerThan: newerThan,
		Limit:     limit,
	}
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("interupted by context cancelation")
		default:
		}
		e.logger.Info("fetching", lager.Data{
			"newer_than": params.NewerThan,
			"cursor":     params.Cursor,
			"limit":      params.Limit,
		})
		auditEvents, errs := e.client.GetAuditEvents(params)
		if errs != nil {
			return nil, squashErrors(errs)
		}
		if auditEvents == nil {
			break
		}
		for _, auditEvent := range *auditEvents {
			params.Cursor = auditEvent.ID
			if auditEvent.Event != DeploymentScaleMembersEvent {
				continue
			}
			eventJSON, err := json.Marshal(auditEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to convert Compose audit event to JSON: %s", err.Error())
			}
			events = append([]eventio.RawEvent{
				{
					GUID:       auditEvent.ID,
					Kind:       e.Kind(),
					CreatedAt:  auditEvent.CreatedAt,
					RawMessage: json.RawMessage(eventJSON),
				},
			}, events...)
		}
		if len(*auditEvents) < params.Limit {
			break
		}
	}
	return events, nil
}

func (e *ComposeEventFetcher) FetchEvents(ctx context.Context, lastEvent *eventio.RawEvent) ([]eventio.RawEvent, error) {
	var lastEventTime *time.Time = nil
	if lastEvent != nil {
		t := lastEvent.CreatedAt.Add(TimePrecisionOffset)
		lastEventTime = &t
	}

	scaleEvents, err := e.fetchScaleEvents(ctx, lastEventTime)
	if err != nil {
		return nil, err
	}
	scaleEvents = sliceToLatest(scaleEvents, lastEvent)

	return scaleEvents, nil
}

func (e *ComposeEventFetcher) Kind() string {
	return Kind
}

func sliceToLatest(events []eventio.RawEvent, lastEvent *eventio.RawEvent) []eventio.RawEvent {
	if lastEvent == nil {
		return events
	}
	for i, event := range events {
		if event.GUID == lastEvent.GUID {
			return events[i+1:]
		}
	}
	return events
}

type Config struct {
	// Logger overrides the default logger
	Logger lager.Logger
	// APIKey sets the API for querying events
	APIKey string
	// Client overrides the default compose client
	Client ComposeClient
	// FetchLimit sets the batch size for each http request
	FetchLimit int
}

// New creates a new EventFetcher for fetching compose events
func New(cfg Config) (*ComposeEventFetcher, error) {
	if cfg.Logger == nil {
		cfg.Logger = lager.NewLogger("compose-event-fetcher")
	}
	if cfg.Client == nil {
		composeClient, err := newClient(cfg.APIKey)
		if err != nil {
			return nil, err
		}
		cfg.Client = composeClient
	}
	fetcher := &ComposeEventFetcher{
		client:     cfg.Client,
		logger:     cfg.Logger,
		fetchLimit: cfg.FetchLimit,
	}
	return fetcher, nil
}
