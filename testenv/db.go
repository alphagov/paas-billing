package testenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-billing/eventio"
	"github.com/alphagov/paas-billing/eventstore"
	uuid "github.com/satori/go.uuid"
)

type TempDB struct {
	MasterConnectionString string
	TempConnectionString   string
	Schema                 eventio.EventStore
	Conn                   *sql.DB
	masterConn             *sql.DB
	name                   string
}

func DropAllDatabases() error {
	fmt.Println("drop")
	masterConnectionString, err := getDatabaseURL()
	if err != nil {
		return err
	}
	conn, err := sql.Open("postgres", masterConnectionString)
	if err != nil {
		return err
	}
	defer conn.Close()
	rows, err := conn.Query(`SELECT datname from pg_database WHERE datname LIKE 'test_%'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var errors []string
	for rows.Next() {
		var (
			dbName string
		)
		if err := rows.Scan(&dbName); err != nil {
			fmt.Println(err)
		}
		_, err = conn.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName))
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("there were errors: %s were not deleted", strings.Join(errors, ", "))
	}
	return nil
}

// Opens creates a new database named test_<uuid> and runs Init() with the given config
func OpenWithContext(cfg eventstore.Config, ctx context.Context) (*TempDB, error) {
	tdb, err := New()
	if err != nil {
		return nil, err
	}
	logger := lager.NewLogger("test")
	s := eventstore.New(ctx, tdb.Conn, logger, cfg)
	if err := s.Init(); err != nil {
		return nil, err
	}
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	tdb.Schema = s
	go func() {
		err := tdb.monitor(ctx)
		if err != nil {
			fmt.Println(err)
		}
	}()
	return tdb, nil
}
func Open(cfg eventstore.Config) (*TempDB, error) {
	return OpenWithContext(cfg, context.Background())
}

func New() (*TempDB, error) {
	masterConnectionString, err := getDatabaseURL()
	if err != nil {
		return nil, err
	}
	master, err := sql.Open("postgres", masterConnectionString)
	if err != nil {
		return nil, err
	}

	dbName := "test_" + strings.Replace(uuid.NewV4().String(), "-", "_", -1)
	_, err = master.Exec(fmt.Sprintf(`CREATE DATABASE %s`, dbName))
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(masterConnectionString)
	if err != nil {
		return nil, err
	}
	u.Path = "/" + dbName
	conn, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, err
	}
	tdb := &TempDB{
		TempConnectionString:   u.String(),
		MasterConnectionString: masterConnectionString,
		Conn:                   conn,
		masterConn:             master,
		name:                   dbName,
	}
	return tdb, nil
}

// MustOpen is a panicy version of Open
func MustOpen(cfg eventstore.Config) *TempDB {
	tdb, err := Open(cfg)
	if err != nil {
		panic(err)
	}
	return tdb
}

func (db *TempDB) monitor(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return db.Cleanup()
		}
	}
}

// Close closes the database connection.
// This needs to gracefully handle the database, and the connection being 'nil' - some specs deliberately cause this state
func (db *TempDB) Close() error {
	if db == nil {
		return errors.New("db was nil")
	}
	if db.Conn == nil {
		return errors.New("db connection was nil")
	}
	return db.Conn.Close()
}
func (db *TempDB) Cleanup() error {
	time.Sleep(10 * time.Second)
	db.Close()
	_, deleteDBError := db.masterConn.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, db.name))
	masterCloseError := db.masterConn.Close()
	return errors.Join(deleteDBError, masterCloseError)
}

// Perform a query that returns a single row single column and return whatever it is
// This is useful for testing
func (db *TempDB) Get(q string, args ...interface{}) interface{} {
	var b []byte
	err := db.Conn.QueryRow(`select coalesce(to_json((`+q+`)), 'null'::json)`, args...).Scan(&b)
	if err != nil {
		return err
	}
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		panic(err)
	}
	return v
}

// Perform a query and return the result as thinly typed Rows
func (db *TempDB) Query(q string, args ...interface{}) Rows {
	var b []byte
	err := db.Conn.QueryRow(`
		with q as (`+q+`)
		select json_agg(row_to_json(q.*)) from q
	`, args...).Scan(&b)
	if err != nil {
		panic(err)
	}
	if b == nil || string(b) == "" {
		return Rows([]Row{})
	}
	var v []Row
	if err := json.Unmarshal(b, &v); err != nil {
		panic(err)
	}
	return Rows(v)
}

// Insert is way of inserting Row objects (generic json representations of
// data) into the database it makes it a bit easier to read the intention of
// the insert in tests than raw sql
func (db *TempDB) Insert(tableName string, rows ...Row) error {
	tx, err := db.Conn.Begin()
	if err != nil {
		return err
	}
	for _, row := range rows {
		cols := []string{}
		vars := []string{}
		vals := []interface{}{}
		for k, v := range row {
			cols = append(cols, k)
			vars = append(vars, fmt.Sprintf("$%d", len(cols)))
			vals = append(vals, v)
		}
		sql := fmt.Sprintf(
			`insert into `+tableName+` (%s) values (%s)`,
			strings.Join(cols, ", "),
			strings.Join(vars, ", "),
		)
		if _, err := tx.Exec(sql, vals...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

const ISO8601 = "2006-01-02T15:04:05-07:00"

// T parses an ISO8601 or RFC3339 or YYYY-MM-DD formated string and returns a time.Time or panics
func Time(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse(ISO8601, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	panic(fmt.Errorf("failed to convert RFC3339 string to time: %v", s))
}

type Row map[string]interface{}

func (row Row) String() string {
	b, err := json.Marshal(row)
	if err != nil {
		panic(fmt.Errorf("cannot JSON stringify row %v", row))
	}
	return string(b)
}

type Rows []Row

func (rows Rows) String() string {
	b, err := json.Marshal(rows)
	if err != nil {
		panic(fmt.Errorf("cannot JSON stringify row %v", rows))
	}
	return string(b)
}

// getDatabaseURL returns a sensible URL for connecting to the database:
// - if $TEST_DATABASE_URL is set, return that
// - if GOOS is 'darwin' (ie. MacOS), assume we are on a developer laptop and return the value from the makefile
// - otherwise, bail out with an error
func getDatabaseURL() (string, error) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url != "" {
		return url, nil
	}
	// This may appear to be an 'always true' situation at a glance, but it is defined differently per-os
	if runtime.GOOS == "darwin" {
		return "postgres://postgres:@localhost:15432/?sslmode=disable", nil
	}
	return "", errors.New("$TEST_DATABASE_URL environment variable is required")
}
