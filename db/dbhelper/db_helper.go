package dbhelper

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	uuid "github.com/satori/go.uuid"
)

var (
	TestDatabaseURL = os.Getenv("TEST_DATABASE_URL")
)

// CreateDB creates a new database named test_<uuid> by connecting to
// TEST_DATABASE_URL and returning the connection string
func CreateDB() (string, error) {
	if TestDatabaseURL == "" {
		return "", errors.New("TEST_DATABASE_URL environment variable is required")
	}
	conn, err := sql.Open("postgres", TestDatabaseURL)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	dbName := "test_" + strings.Replace(uuid.NewV4().String(), "-", "_", -1)
	_, err = conn.Exec(fmt.Sprintf(`CREATE DATABASE %s`, dbName))
	if err != nil {
		return "", err
	}
	u, err := url.Parse(TestDatabaseURL)
	if err != nil {
		return "", err
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

// DropDB drops the database at connst using the connection at TEST_DATABASE_URL
func DropDB(connstr string) error {
	if TestDatabaseURL == "" {
		return errors.New("TEST_DATABASE_URL environment variable is required")
	}
	conn, err := sql.Open("postgres", TestDatabaseURL)
	if err != nil {
		return err
	}
	defer conn.Close()
	u, err := url.Parse(connstr)
	if err != nil {
		return err
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	_, err = conn.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName))
	return err
}
