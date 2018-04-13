package compose

import (
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/collector"
	composeclient "github.com/alphagov/paas-billing/compose"
	"github.com/alphagov/paas-billing/db"
	composeapi "github.com/compose/gocomposeapi"
)

var _ collector.EventFetcher = &EventFetcher{}

type EventFetcher struct {
	logger    lager.Logger
	sqlClient db.SQLClient
	client    composeclient.Client
}

func NewEventFetcher(sqlClient db.SQLClient, client composeclient.Client) *EventFetcher {
	return &EventFetcher{
		sqlClient: sqlClient,
		client:    client,
	}
}

func (e *EventFetcher) FetchEvents(logger lager.Logger, fetchLimit int, recordMinAge time.Duration) (int, error) {
	latestEventID, err := e.sqlClient.FetchComposeLatestEventID()
	if err != nil {
		return 0, err
	}

	cursor, err := e.sqlClient.FetchComposeCursor()
	if err != nil {
		return 0, err
	}

	var cursorStr string
	if cursor != nil {
		cursorStr = *cursor
	}

	startTime := time.Now()
	auditEvents, errs := e.client.GetAuditEvents(composeapi.AuditEventsParams{
		Cursor: cursorStr,
		Limit:  fetchLimit,
	})
	if errs != nil {
		return 0, composeclient.SquashErrors(errs)
	}
	elapsedTime := time.Now().Sub(startTime)

	cnt := 0
	filteredEvents := make([]composeapi.AuditEvent, 0)
	for _, event := range *auditEvents {
		if latestEventID != nil && event.ID == *latestEventID {
			break
		}
		if event.Event == "deployment.scale.members" {
			filteredEvents = append(filteredEvents, event)
		}
		cnt++
	}

	logger.Info("fetch", lager.Data{
		"cursor":          cursor,
		"latest_event_id": latestEventID,
		"event_cnt":       cnt,
		"response_time":   elapsedTime.String(),
	})

	if cnt > 0 || cursor != nil {
		tx, err := e.sqlClient.BeginTx()
		if err != nil {
			return 0, err
		}

		if len(filteredEvents) > 0 {
			if err := tx.InsertComposeAuditEvents(filteredEvents); err != nil {
				tx.Rollback()
				return 0, err
			}
		}

		if cursor == nil {
			latestEventID = &(*auditEvents)[0].ID
			if err := tx.InsertComposeLatestEventID(*latestEventID); err != nil {
				tx.Rollback()
				return 0, err
			}
		}

		if cnt == fetchLimit {
			cursor = &(*auditEvents)[len(*auditEvents)-1].ID
		} else {
			cursor = nil
		}

		if err := tx.InsertComposeCursor(cursor); err != nil {
			tx.Rollback()
			return 0, err
		}

		if err := tx.Commit(); err != nil {
			return 0, err
		}

		logger.Info("store", lager.Data{
			"cursor":            cursor,
			"latest_event_id":   latestEventID,
			"billing_event_cnt": len(filteredEvents),
		})
	}

	return cnt, nil
}

func (e *EventFetcher) Name() string {
	return "compose-audit-events"
}
