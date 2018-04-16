package testenv

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alphagov/paas-billing/schema"
	uuid "github.com/satori/go.uuid"
)

type TempDB struct {
	masterConnectionString string
	tempConnectionString   string
	Schema                 *schema.Schema
	Conn                   *sql.DB
}

// Close drops the database
func (db *TempDB) Close() error {
	db.Conn.Close()
	conn, err := sql.Open("postgres", db.masterConnectionString)
	if err != nil {
		return err
	}
	defer conn.Close()
	u, err := url.Parse(db.tempConnectionString)
	if err != nil {
		return err
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	try := 0
	for {
		_, err = conn.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName))
		if err != nil {
			if try > 3 {
				return err
			}
			fmt.Println(err)
			try++
			time.Sleep(1 * time.Second)
			continue
		}
		return nil
	}
}

// Opens creates a new database named test_<uuid> and runs schema.Init() with the given config
func Open(cfg schema.Config) (*TempDB, error) {
	masterConnectionString := os.Getenv("TEST_DATABASE_URL")
	if masterConnectionString == "" {
		return nil, errors.New("TEST_DATABASE_URL environment variable is required")
	}
	master, err := sql.Open("postgres", masterConnectionString)
	if err != nil {
		return nil, err
	}
	defer master.Close()
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
	s := schema.New(context.Background(), conn, cfg)
	if err := s.Init(); err != nil {
		return nil, err
	}
	tdb := &TempDB{
		tempConnectionString:   u.String(),
		masterConnectionString: masterConnectionString,
		Conn:   conn,
		Schema: s,
	}
	return tdb, nil
}

// MustOpen is a panicy version of Open
func MustOpen(cfg schema.Config) *TempDB {
	tdb, err := Open(cfg)
	if err != nil {
		panic(err)
	}
	return tdb
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
	var v []Row
	if err := json.Unmarshal(b, &v); err != nil {
		panic(err)
	}
	return Rows(v)
}

// Insert is way of inserting Row objects (generic json representations of
// data) into the database it makes it a bit easier to read the intension of
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
