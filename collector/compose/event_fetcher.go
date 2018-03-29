package compose

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/collector"
	composeclient "github.com/alphagov/paas-billing/compose"
	composeapi "github.com/compose/gocomposeapi"
)

var _ collector.EventFetcher = &EventFetcher{}

const (
	latestEventCursor = "latest_event_id"
	positionCursor    = "cursor"
)

type EventFetcher struct {
	logger lager.Logger
	db     *sql.DB
	client composeclient.Client
}

func NewEventFetcher(db *sql.DB, client composeclient.Client) *EventFetcher {
	return &EventFetcher{
		db:     db,
		client: client,
	}
}

func (e *EventFetcher) FetchEvents(logger lager.Logger, fetchLimit int, recordMinAge time.Duration) (int, error) {
	tx, err := e.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	latestEventID, err := e.getNamedCursor(tx, latestEventCursor)
	if err != nil {
		return 0, err
	}

	cursor, err := e.getNamedCursor(tx, positionCursor)
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
	elapsedTime := time.Since(startTime)

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
		if len(filteredEvents) > 0 {
			if err := e.insertComposeAuditEvents(tx, filteredEvents); err != nil {
				return 0, err
			}
		}

		if cursor == nil {
			latestEventID = &(*auditEvents)[0].ID
			if err := e.setNamedCursor(tx, latestEventCursor, latestEventID); err != nil {
				return 0, err
			}
		}

		if cnt == fetchLimit {
			cursor = &(*auditEvents)[len(*auditEvents)-1].ID
		} else {
			cursor = nil
		}

		if err := e.setNamedCursor(tx, positionCursor, cursor); err != nil {
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

func (e *EventFetcher) setNamedCursor(tx *sql.Tx, name string, value *string) error {
	_, err := tx.Exec("UPDATE compose_audit_events_cursor SET value = $1 WHERE name = $2", value, name)
	return err
}

func (e *EventFetcher) getNamedCursor(tx *sql.Tx, name string) (*string, error) {
	var value *string
	queryErr := e.db.QueryRow(
		"SELECT value FROM compose_audit_events_cursor WHERE name = $1", name,
	).Scan(&value)

	switch {
	case queryErr == sql.ErrNoRows:
		return nil, nil
	case queryErr != nil:
		return nil, queryErr
	default:
		return value, nil
	}
}

func (e *EventFetcher) insertComposeAuditEvents(tx *sql.Tx, events []composeapi.AuditEvent) error {
	valueStrings := make([]string, 0, len(events))
	valueArgs := make([]interface{}, 0, len(events)*3)
	i := 1
	for _, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to convert Compose audit event to JSON: %s", err.Error())
		}
		p1, p2, p3 := i, i+1, i+2
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, ($%d)::jsonb - '_links')", p1, p2, p3))
		valueArgs = append(valueArgs, event.ID)
		valueArgs = append(valueArgs, event.CreatedAt)
		valueArgs = append(valueArgs, string(eventJSON))
		i += 3
	}
	stmt := fmt.Sprintf("INSERT INTO compose_audit_events (event_id, created_at, raw_message) VALUES %s", strings.Join(valueStrings, ","))
	_, err := tx.Exec(stmt, valueArgs...)
	return err
}
