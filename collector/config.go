package collector

import "os"

// Config is the collector configuration
type Config struct {
	DefaultSchedule string
	MinWaitTime     string
	FetchLimit      string
	RecordMinAge    string
}

// CreateConfigFromEnv creates a collector configuration from environment variables
func CreateConfigFromEnv() *Config {
	return &Config{
		DefaultSchedule: os.Getenv("COLLECTOR_DEFAULT_SCHEDULE"),
		MinWaitTime:     os.Getenv("COLLECTOR_MIN_WAIT_TIME"),
		FetchLimit:      os.Getenv("COLLECTOR_FETCH_LIMIT"),
		RecordMinAge:    os.Getenv("COLLECTOR_RECORD_MIN_AGE"),
	}
}
