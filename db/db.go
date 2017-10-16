package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	cf "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	_ "github.com/lib/pq"
)

const (
	AppUsageTableName     = "app_usage_events"
	ServiceUsageTableName = "service_usage_events"
	PostgresTimeFormat    = "2006-01-02T15:04:05.000000Z"
)

type SQLClient struct {
	ConnectionString string
}

func NewSQLClient() SQLClient {
	var connectionString string
	databaseUrl, ok := os.LookupEnv("DATABASE_URL")
	if ok {
		connectionString = databaseUrl
	} else {
		connectionString = "postgres://postgres:@localhost:5432/?sslmode=disable"
	}
	return SQLClient{ConnectionString: connectionString}
}

func (sc SQLClient) InsertUsageEventList(data cf.UsageEventList, tableName string) error {
	db, err := sql.Open("postgres", sc.ConnectionString)
	defer db.Close()

	valueStrings := make([]string, 0, len(data.Resources))
	valueArgs := make([]interface{}, 0, len(data.Resources)*3)
	i := 1
	for _, event := range data.Resources {
		p1, p2, p3 := i, i+1, i+2
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", p1, p2, p3))
		valueArgs = append(valueArgs, event.MetaData.GUID)
		valueArgs = append(valueArgs, event.MetaData.CreatedAt)
		valueArgs = append(valueArgs, event.EntityRaw)
		i += 3
	}
	stmt := fmt.Sprintf("INSERT INTO %s (guid, created_at, raw_message) VALUES %s", tableName, strings.Join(valueStrings, ","))
	_, err = db.Exec(stmt, valueArgs...)
	if err != nil {
		return err
	}

	return nil
}

func (sc SQLClient) SelectUsageEvents(tableName string) (cf.UsageEventList, error) {
	var usageEvents cf.UsageEventList

	db, err := sql.Open("postgres", sc.ConnectionString)
	defer db.Close()

	rows, err := db.Query("SELECT guid, created_at, raw_message FROM " + tableName)
	if err != nil {
		return cf.UsageEventList{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var guid string
		var createdAt string
		var rawMessage json.RawMessage
		err = rows.Scan(&guid, &createdAt, &rawMessage)
		if err != nil {
			return cf.UsageEventList{}, err
		}
		usageEvents.Resources = append(usageEvents.Resources, cf.UsageEvent{
			MetaData:  cf.MetaData{GUID: guid, CreatedAt: decodePostgresTimestamp(createdAt)},
			EntityRaw: rawMessage,
		})
	}
	err = rows.Err()
	if err != nil {
		return cf.UsageEventList{}, err
	}
	return usageEvents, nil
}

func decodePostgresTimestamp(timestamp string) time.Time {
	t, _ := time.Parse(PostgresTimeFormat, timestamp)
	return t
}

func (sc SQLClient) FetchLastGUID(tableName string) (string, error) {
	db, err := sql.Open("postgres", sc.ConnectionString)
	defer db.Close()

	var guid string
	err = db.QueryRow("SELECT guid FROM " + tableName + " ORDER BY id DESC LIMIT 1").Scan(&guid)

	switch {
	case err == sql.ErrNoRows:
		return cf.GUIDNil, nil
	case err != nil:
		return "", err
	default:
		return guid, nil
	}
}
