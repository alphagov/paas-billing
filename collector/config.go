package collector

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const maxFetchLimit = 100

// Config is the collector configuration
type Config struct {
	DefaultSchedule time.Duration
	MinWaitTime     time.Duration
	FetchLimit      int
	RecordMinAge    time.Duration
}

func CreateDefaultConfig() *Config {
	return &Config{
		DefaultSchedule: time.Duration(15 * time.Minute),
		MinWaitTime:     time.Duration(3 * time.Second),
		FetchLimit:      50,
		RecordMinAge:    time.Duration(5 * time.Minute),
	}
}

// CreateConfigFromEnv creates a collector configuration from environment variables
func CreateConfigFromEnv() (*Config, error) {
	c := CreateDefaultConfig()
	var err error

	defaultSchedule := os.Getenv("COLLECTOR_DEFAULT_SCHEDULE")
	if defaultSchedule != "" {
		if c.DefaultSchedule, err = time.ParseDuration(defaultSchedule); err != nil {
			return nil, fmt.Errorf("COLLECTOR_DEFAULT_SCHEDULE is invalid")
		}
	}

	minWaitTime := os.Getenv("COLLECTOR_MIN_WAIT_TIME")
	if minWaitTime != "" {
		if c.MinWaitTime, err = time.ParseDuration(minWaitTime); err != nil {
			return nil, fmt.Errorf("COLLECTOR_MIN_WAIT_TIME is invalid")
		}
	}

	recordMinAge := os.Getenv("COLLECTOR_RECORD_MIN_AGE")
	if recordMinAge != "" {
		if c.RecordMinAge, err = time.ParseDuration(recordMinAge); err != nil {
			return nil, fmt.Errorf("COLLECTOR_RECORD_MIN_AGE is invalid")
		}
	}

	fetchLimit := os.Getenv("COLLECTOR_FETCH_LIMIT")
	if fetchLimit != "" {
		if c.FetchLimit, err = strconv.Atoi(fetchLimit); err != nil {
			return nil, fmt.Errorf("COLLECTOR_FETCH_LIMIT is invalid")
		}
		if c.FetchLimit <= 0 || c.FetchLimit > maxFetchLimit {
			return nil, fmt.Errorf("COLLECTOR_FETCH_LIMIT must be between 1 and 100")
		}
	}

	return c, nil
}
