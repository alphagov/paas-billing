package db

import (
	"fmt"
)

// InitSchema initialises the database tables
func (pc PostgresClient) InitSchema() error {
	db, openErr := pc.Open()
	if openErr != nil {
		return openErr
	}
	defer db.Close()

	createAppUsageEventTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL,
			guid CHAR(36) UNIQUE,
			created_at TIMESTAMP,
			raw_message JSONB
		)`,
		AppUsageTableName,
	)
	if _, err := db.Exec(createAppUsageEventTable); err != nil {
		return err
	}

	createServiceUsageEventTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL,
			guid CHAR(36) UNIQUE,
			created_at TIMESTAMP,
			raw_message JSONB
		)`,
		ServiceUsageTableName,
	)
	if _, err := db.Exec(createServiceUsageEventTable); err != nil {
		return err
	}

	return nil
}
