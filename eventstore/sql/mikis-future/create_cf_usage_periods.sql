CREATE TABLE cf_usage_periods (
  start_event_guid uuid NOT NULL,
  stop_event_guid uuid,
  timerange tstzrange NOT NULL,

  PRIMARY KEY (start_event_guid, timerange),
  FOREIGN KEY (start_event_guid) REFERENCES cf_usage_events (guid),
  FOREIGN KEY (stop_event_guid) REFERENCES cf_usage_events (guid)
);

CREATE INDEX IF NOT EXISTS cf_usage_periods_duration_idx ON cf_usage_periods USING gist (timerange);

CREATE INDEX IF NOT EXISTS cf_usage_periods_fn_safpd_idx ON cf_usage_periods (UPPER(timerange));
