package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/lib/pq"
)

// InitSchema initialises the database tables and functions
func (pc *PostgresClient) InitSchema() (err error) {
	migrations, err := MigrationSequence()
	if err != nil {
		return err
	}
	return pc.ApplyMigrations(migrations)
}

func MigrationSequence() ([]string, error) {
	schemaDir := schemaDataDir()
	filepaths, err := filepath.Glob(filepath.Join(schemaDir, "*.sql"))
	if err != nil {
		return nil, err
	}
	if len(filepaths) < 1 {
		return nil, fmt.Errorf("failed to initialize sql schema: no .sql files found in '%s'", schemaDir)
	}
	filenames := make([]string, len(filepaths))
	for i, filepath_ := range filepaths {
		filenames[i] = filepath.Base(filepath_)
	}
	sort.Strings(filenames)
	return filenames, nil
}

func (pc *PostgresClient) ApplyMigrations(sortedMigrationFilenames []string) (err error) {
	tx, err := pc.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	for _, migrationFilename := range sortedMigrationFilenames {
		migrationFilepath := filepath.Join(schemaDataDir(), migrationFilename)
		sql, err := ioutil.ReadFile(migrationFilepath)
		if err != nil {
			return fmt.Errorf("Error reading %s from %s: %s", migrationFilename, migrationFilepath, err)
		}

		_, err = tx.Exec(string(sql))
		if err != nil {
			msg := err.Error()
			if err, ok := err.(*pq.Error); ok {
				msg = err.Message
				if err.Detail != "" {
					msg += ": " + err.Detail
				}
				if err.Hint != "" {
					msg += ": " + err.Hint
				}
				if err.Where != "" {
					msg += ": " + err.Where
				}
			}
			return fmt.Errorf("Error applying %s: %s", migrationFilename, msg)
		}
	}

	return nil
}

func schemaDataDir() string {
	p := os.Getenv("DATABASE_SCHEMA_DIR")
	if p != "" {
		return p
	}
	pwd := os.Getenv("PWD")
	if pwd == "" {
		pwd, _ = os.Getwd()
	}
	return filepath.Join(pwd, "db", "sql")
}
