package db

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	cf "github.com/alphagov/paas-billing/cloudfoundry"
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

// SQLClient is a general interface for handling usage event queries
type SQLClient interface {
	InitSchema() error
	InsertUsageEventList(data *cf.UsageEventList, tableName string) error
	FetchLastGUID(tableName string) (string, error)
	QueryJSON(q string, args ...interface{}) io.Reader
	QueryRowJSON(q string, args ...interface{}) io.Reader
	UpdateViews() error
	Close() error
}

// PostgresClient is the Postgres DB client for handling usage event queries
type PostgresClient struct {
	Conn *sql.DB
}

// NewPostgresClient creates a new Postgres client
func NewPostgresClient(connectionString string) (*PostgresClient, error) {
	conn, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	pc := &PostgresClient{
		Conn: conn,
	}
	return pc, nil
}

// Close the connection
func (pc *PostgresClient) Close() error {
	return pc.Conn.Close()
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
	_, err := pc.Conn.Exec("REFRESH MATERIALIZED VIEW billable")
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
