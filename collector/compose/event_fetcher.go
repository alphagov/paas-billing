package compose

import (
	"encoding/json"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/collector"
	composeclient "github.com/alphagov/paas-billing/compose"
	"github.com/alphagov/paas-billing/store"
	composeapi "github.com/compose/gocomposeapi"
)

var _ collector.EventFetcher = &EventFetcher{}

const (
	TimePrecisionOffset = -1 * time.Second
	DefaultFetchLimit   = 50
	Kind                = "compose"
)

var (
	DefaultEventEpoch = time.Time{}
)

const (
	DeploymentScaleMembersEvent = "deployment.scale.members"
)

type EventFetcher struct {
	Logger     lager.Logger
	Store      store.EventStorer
	Compose    composeclient.Client
	FetchLimit int
}

func (e *EventFetcher) fetchScaleEvents(newerThan *time.Time) ([]store.RawEvent, error) {
	events := make([]store.RawEvent, 0)
	limit := DefaultFetchLimit
	if e.FetchLimit != 0 {
		limit = e.FetchLimit
	}
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("FetchLimit must be between 1 and 100")
	}
	params := composeapi.AuditEventsParams{
		NewerThan: newerThan,
		Limit:     limit,
	}
	for {
		e.Logger.Info("fetching", lager.Data{
			"newer_than": fmt.Sprintf("%v", params.NewerThan),
			"cursor":     params.Cursor,
			"limit":      params.Limit,
		})
		auditEvents, errs := e.Compose.GetAuditEvents(params)
		if errs != nil {
			return nil, composeclient.SquashErrors(errs)
		}
		if auditEvents == nil {
			break
		}
		for _, auditEvent := range *auditEvents {
			if auditEvent.Event != DeploymentScaleMembersEvent {
				continue
			}
			eventJSON, err := json.Marshal(auditEvent)
			if err != nil {
				return nil, fmt.Errorf("failed to convert Compose audit event to JSON: %s", err.Error())
			}
			events = append([]store.RawEvent{
				{
					GUID:       auditEvent.ID,
					Kind:       e.Kind(),
					CreatedAt:  auditEvent.CreatedAt,
					RawMessage: json.RawMessage(eventJSON),
				},
			}, events...)
			params.Cursor = auditEvent.ID
		}
		if len(*auditEvents) < params.Limit {
			break
		}
	}
	return events, nil
}

func (e *EventFetcher) FetchEvents(lastEvent *store.RawEvent) ([]store.RawEvent, error) {
	var lastEventTime *time.Time = nil
	if lastEvent != nil {
		t := lastEvent.CreatedAt.Add(TimePrecisionOffset)
		lastEventTime = &t
	}

	startTime := time.Now()
	scaleEvents, err := e.fetchScaleEvents(lastEventTime)
	if err != nil {
		return nil, err
	}
	scaleEvents = sliceToLatest(scaleEvents, lastEvent)
	elapsedTime := time.Since(startTime)

	e.Logger.Info("fetched", lager.Data{
		"newer_than":    fmt.Sprintf("%v", lastEventTime),
		"event_cnt":     len(scaleEvents),
		"response_time": elapsedTime.String(),
	})

	return scaleEvents, nil
}

func (e *EventFetcher) Kind() string {
	return Kind
}

func sliceToLatest(events []store.RawEvent, lastEvent *store.RawEvent) []store.RawEvent {
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
