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
	schemaDir := schemaDataDir()
	files, err := filepath.Glob(filepath.Join(schemaDir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) < 1 {
		return fmt.Errorf("failed to initialize sql schema: no .sql files found in '%s'", schemaDir)
	}
	sort.Strings(files)
	tx, txErr := pc.db.Begin()
	if txErr != nil {
		return txErr
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	for _, filename := range files {
		sql, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		_, err = tx.Exec(string(sql))
		if err != nil {
			msg := err.Error()
			if err, ok := err.(*pq.Error); ok {
				msg = err.Message
				if err.Hint != "" {
					msg += ": " + err.Hint
				}
				if err.Where != "" {
					msg += ": " + err.Where
				}
			}
			return fmt.Errorf("Error applying %s: %s", filename, msg)
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
