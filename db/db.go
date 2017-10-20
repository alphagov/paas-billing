package db

import (
	"database/sql"
	"fmt"
	"strings"

	cf "github.com/alphagov/paas-usage-events-collector/cloudfoundry"
	_ "github.com/lib/pq"
)

// Database table names
const (
	AppUsageTableName     = "app_usage_events"
	ServiceUsageTableName = "service_usage_events"
)

// SQLClient is a general interface for handling usage event queries
type SQLClient interface {
	InitSchema() error
	InsertUsageEventList(data *cf.UsageEventList, tableName string) error
	FetchLastGUID(tableName string) (string, error)
}

// PostgresClient is the Postgres DB client for handling usage event queries
type PostgresClient struct {
	connectionString string
}

// NewPostgresClient creates a new Postgres client
func NewPostgresClient(connectionString string) PostgresClient {
	return PostgresClient{connectionString: connectionString}
}

// Open opens the database connection
func (pc PostgresClient) Open() (*sql.DB, error) {
	return sql.Open("postgres", pc.connectionString)
}

// InsertUsageEventList saves the usage event records in the database
func (pc PostgresClient) InsertUsageEventList(data *cf.UsageEventList, tableName string) error {
	db, openErr := pc.Open()
	if openErr != nil {
		return openErr
	}
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
	_, execErr := db.Exec(stmt, valueArgs...)
	if execErr != nil {
		return execErr
	}

	return nil
}

// FetchLastGUID returns with the last inserted GUID
//
// If the table is empty it will return with cloudfoundry.GUIDNil
func (pc PostgresClient) FetchLastGUID(tableName string) (string, error) {
	db, openErr := pc.Open()
	if openErr != nil {
		return "", openErr
	}
	defer db.Close()

	var guid string
	queryErr := db.QueryRow("SELECT guid FROM " + tableName + " ORDER BY id DESC LIMIT 1").Scan(&guid)

	switch {
	case queryErr == sql.ErrNoRows:
		return cf.GUIDNil, nil
	case queryErr != nil:
		return "", queryErr
	default:
		return guid, nil
	}
}
