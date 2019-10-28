-- When inserting new app or service usage events:
-- If it's a START or CREATE event, we'll insert a new row and set the duration to be unbounded (e.g., to year 9999.)
-- If it's a STOP or DELETE event, we'll update the existing previous row to set the end of its duration.
-- This assumes that all details used for computing prices (plan, service, org, space GUID, etc) match. T
-- If it's an UPDATE event, those details change. We'll end the existing row to end it and create a new one.
CREATE TABLE cf_usage_periods (
  start_guid uuid NOT NULL,
  stop_guid uuid,
  duration tstzrange NOT NULL,
  in_use boolean NOT NULL, -- whether this resource is being in-use (i.e., billable) as of this event

  PRIMARY KEY (start_guid, stop_guid),
  FOREIGN KEY (start_guid) REFERENCES cf_usage_events (guid),
  FOREIGN KEY (stop_guid) REFERENCES cf_usage_events (guid),
  CONSTRAINT duration_must_not_be_empty CHECK (not isempty(duration))
);

CREATE INDEX IF NOT EXISTS cf_usage_periods_duration_idx ON cf_usage USING gist (duration);

-- Prepared statement for getting usage on a specific day.
--
-- Call it with a historic date:
--   EXECUTE cf_usage_on_day ('2019-10-01'::date);
--
-- Call it with today's date:
--   EXECUTE cf_usage_on_day (now()::date);
-- Durations will be trimmed down to the present time. DURATIONS WON'T INCLUDE THE FUTURE.
PREPARE cf_usage_on_day(date) AS (
  WITH excluded_end_time AS (
    SELECT LEAST($1 + INTERVAL '1 DAY', now()) AS excluded_end_time
  )
  SELECT
    *,
    $1 as day,
    duration * tstzrange($1, excluded_end_time, '[)') AS duration_on_day
  FROM
    cf_usage,
    excluded_end_time
  WHERE
    duration && tstzrange($1, excluded_end_time, '[)')
);
