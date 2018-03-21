package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	cf "github.com/alphagov/paas-billing/cloudfoundry"
	composeapi "github.com/compose/gocomposeapi"
	_ "github.com/lib/pq"
)

const (
	AppUsageTableName     = "app_usage_events"
	ServiceUsageTableName = "service_usage_events"
	ComputePlanGuid       = "f4d4b95a-f55e-4593-8d54-3364c25798c4"
)

const (
	StateStarted = "STARTED"
	StateStopped = "STOPPED"
)

const (
	composeLatestEventID = "latest_event_id"
	composeCursor        = "cursor"
)

// SQLClient is a general interface for handling usage event queries
//go:generate counterfeiter -o fakes/fake_sqlclient.go . SQLClient
type SQLClient interface {
	InitSchema() error
	InsertUsageEventList(data *cf.UsageEventList, tableName string) error
	FetchLastGUID(tableName string) (string, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	QueryJSON(q string, args ...interface{}) io.Reader
	QueryRowJSON(q string, args ...interface{}) io.Reader
	UpdateViews() error
	BeginTx() (SQLClient, error)
	Rollback() error
	Commit() error
	InsertComposeCursor(*string) error
	FetchComposeCursor() (*string, error)
	InsertComposeLatestEventID(string) error
	FetchComposeLatestEventID() (*string, error)
	InsertComposeAuditEvents([]composeapi.AuditEvent) error
}

type SQLConn interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
}

// PostgresClient is the Postgres DB client for handling usage event queries
type PostgresClient struct {
	db   *sql.DB
	tx   *sql.Tx
	Conn SQLConn
}

// NewPostgresClient creates a new Postgres client
func NewPostgresClient(connectionString string) (*PostgresClient, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	pc := &PostgresClient{
		db:   db,
		tx:   nil,
		Conn: db,
	}
	return pc, nil
}

func (pc *PostgresClient) BeginTx() (SQLClient, error) {
	if pc.tx != nil {
		return nil, errors.New("cannot create a transaction within a transaction")
	}
	tx, err := pc.db.Begin()
	if err != nil {
		return nil, err
	}
	newPc := &PostgresClient{
		db:   pc.db,
		tx:   tx,
		Conn: tx,
	}
	return newPc, nil
}

func (pc *PostgresClient) Rollback() error {
	if pc.tx == nil {
		return errors.New("cannot rollback unless in a transaction")
	}
	return pc.tx.Rollback()
}

func (pc *PostgresClient) Commit() error {
	if pc.tx == nil {
		return errors.New("cannot commit unless in a transaction")
	}
	return pc.tx.Commit()
}

// Close the connection
func (pc *PostgresClient) Close() error {
	if pc.tx != nil {
		pc.tx.Commit()
	}
	return pc.db.Close()
}
func (pc *PostgresClient) Exec(query string, args ...interface{}) (sql.Result, error) {
	return pc.Conn.Exec(query, args...)
}
func (pc *PostgresClient) Prepare(query string) (*sql.Stmt, error) {
	return pc.Conn.Prepare(query)
}

// InsertUsageEventList saves the usage event records in the database
func (pc *PostgresClient) InsertUsageEventList(data *cf.UsageEventList, tableName string) error {
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
	_, execErr := pc.Conn.Exec(stmt, valueArgs...)
	return execErr
}

// FetchLastGUID returns with the last inserted GUID
//
// If the table is empty it will return with cloudfoundry.GUIDNil
func (pc *PostgresClient) FetchLastGUID(tableName string) (string, error) {
	var guid string
	queryErr := pc.Conn.QueryRow("SELECT guid FROM " + tableName + " ORDER BY id DESC LIMIT 1").Scan(&guid)

	switch {
	case queryErr == sql.ErrNoRows:
		return cf.GUIDNil, nil
	case queryErr != nil:
		return "", queryErr
	default:
		return guid, nil
	}
}

// UpdateViews updates the indexed materialized views used to generate reports
func (pc *PostgresClient) UpdateViews() error {
	_, err := pc.Conn.Exec("REFRESH MATERIALIZED VIEW CONCURRENTLY resource_usage")
	return err
}

// QueryJSON executes SQL query q with args and writes the result as JSON to w
func (pc *PostgresClient) QueryJSON(q string, args ...interface{}) io.Reader {
	return pc.doQueryJSON(true, q, args...)
}

// QueryRowJSON is the same as QueryJSON but for a single row
func (pc *PostgresClient) QueryRowJSON(q string, args ...interface{}) io.Reader {
	return pc.doQueryJSON(false, q, args...)
}

func (pc *PostgresClient) doQueryJSON(many bool, q string, args ...interface{}) io.Reader {
	r, w := io.Pipe()
	go func() {
		var closeErr error
		defer func() {
			w.CloseWithError(closeErr)
		}()
		rows, err := pc.Conn.Query(fmt.Sprintf(`
			with
			q as (
				%s
			)
			select row_to_json(q.*) from q;
		`, q), args...)
		if err != nil {
			closeErr = err
			return
		}
		defer rows.Close()
		if many {
			fmt.Fprint(w, "[\n")
		}
		for i := 0; rows.Next(); i++ {
			var result string
			if err := rows.Scan(&result); err != nil {
				closeErr = err
				return
			}
			if i > 0 {
				fmt.Fprint(w, ",\n")
			}
			fmt.Fprint(w, result)
		}
		if err := rows.Err(); err != nil {
			closeErr = err
			return
		}
		if many {
			fmt.Fprint(w, "\n]")
		}
	}()
	return r
}

func (pc *PostgresClient) InsertComposeCursor(value *string) error {
	return pc.insertComposeCursor(composeCursor, value)
}

func (pc *PostgresClient) InsertComposeLatestEventID(value string) error {
	return pc.insertComposeCursor(composeLatestEventID, &value)
}

func (pc *PostgresClient) FetchComposeCursor() (*string, error) {
	return pc.fetchComposeCursor(composeCursor)
}

func (pc *PostgresClient) FetchComposeLatestEventID() (*string, error) {
	return pc.fetchComposeCursor(composeLatestEventID)
}

func (pc *PostgresClient) insertComposeCursor(name string, value *string) error {
	_, err := pc.Conn.Exec("UPDATE compose_audit_events_cursor SET value = $1 WHERE name = $2", value, name)
	return err
}

func (pc *PostgresClient) fetchComposeCursor(name string) (*string, error) {
	var value *string
	queryErr := pc.Conn.QueryRow(
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

func (pc *PostgresClient) InsertComposeAuditEvents(events []composeapi.AuditEvent) error {
	valueStrings := make([]string, 0, len(events))
	valueArgs := make([]interface{}, 0, len(events)*3)
	i := 1
	for _, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to convert Compose audit event to JSON: %s", err.Error())
		}
		p1, p2, p3 := i, i+1, i+2
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", p1, p2, p3))
		valueArgs = append(valueArgs, event.ID)
		valueArgs = append(valueArgs, event.CreatedAt)
		valueArgs = append(valueArgs, string(eventJSON))
		i += 3
	}
	stmt := fmt.Sprintf("INSERT INTO compose_audit_events (event_id, created_at, raw_message) VALUES %s", strings.Join(valueStrings, ","))
	_, err := pc.Conn.Exec(stmt, valueArgs...)
	return err
}
