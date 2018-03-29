package cloudfoundry

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-billing/cloudfoundry"
	"github.com/alphagov/paas-billing/collector"
)

var _ collector.EventFetcher = &EventFetcher{}

type EventFetcher struct {
	db        *sql.DB
	client    cloudfoundry.UsageEventsAPI
	tableName string
}

func NewEventFetcher(db *sql.DB, client cloudfoundry.UsageEventsAPI) *EventFetcher {
	return &EventFetcher{
		db:        db,
		client:    client,
		tableName: fmt.Sprintf("%s_usage_events", client.Type()),
	}
}

func (e *EventFetcher) FetchEvents(logger lager.Logger, fetchLimit int, recordMinAge time.Duration) (int, error) {
	tx, err := e.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	guid, err := e.fetchLastGUID(tx)
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
		if err := e.InsertEvents(tx, usageEvents); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return cnt, nil
}

func (e *EventFetcher) Name() string {
	return fmt.Sprintf("cf-%s-usage-event-collector", e.client.Type())
}

func (e *EventFetcher) InsertEvents(tx *sql.Tx, batch *cloudfoundry.UsageEventList) error {
	valueStrings := make([]string, 0, len(batch.Resources))
	valueArgs := make([]interface{}, 0, len(batch.Resources)*3)
	i := 1
	for _, event := range batch.Resources {
		// if event.MetaData.GUID == "" {
		// 	return fmt.Errorf("cannot insert event without an event GUID")
		// }
		// if event.MetaData.CreatedAt == "" {
		// 	return fmt.Errorf("cannot insert event without a created_at timestamp")
		// }
		p1, p2, p3 := i, i+1, i+2
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", p1, p2, p3))
		valueArgs = append(valueArgs, event.MetaData.GUID)
		valueArgs = append(valueArgs, event.MetaData.CreatedAt)
		valueArgs = append(valueArgs, event.EntityRaw)
		i += 3
	}
	stmt := fmt.Sprintf("INSERT INTO %s (guid, created_at, raw_message) VALUES %s", e.tableName, strings.Join(valueStrings, ","))
	_, execErr := tx.Exec(stmt, valueArgs...)
	return execErr
}

// FetchLastGUID returns with the last inserted GUID
// If the table is empty it will return with cloudfoundry.GUIDNil
func (e *EventFetcher) fetchLastGUID(tx *sql.Tx) (string, error) {
	var guid string
	queryErr := tx.QueryRow("SELECT guid FROM " + e.tableName + " ORDER BY id DESC LIMIT 1").Scan(&guid)

	switch {
	case queryErr == sql.ErrNoRows:
		return cloudfoundry.GUIDNil, nil
	case queryErr != nil:
		return "", queryErr
	default:
		return guid, nil
	}
}

func (e *EventFetcher) LastGUID() (string, error) {
	tx, err := e.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	return e.fetchLastGUID(tx)
}
