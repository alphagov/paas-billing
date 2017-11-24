package db

import (
	"fmt"
	"sort"

	"github.com/lib/pq"
)

// InitSchema initialises the database tables and functions
//go:generate go-bindata -pkg db -o bindata.go sql/...
func (pc *PostgresClient) InitSchema() (err error) {
	names := AssetNames()
	sort.Strings(names)

	tx, txErr := pc.Conn.Begin()
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

	for _, name := range names {
		sql := string(MustAsset(name))
		_, err := tx.Exec(sql)
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
			return fmt.Errorf("Error applying %s: %s", name, msg)
		}
	}

	return nil
}
