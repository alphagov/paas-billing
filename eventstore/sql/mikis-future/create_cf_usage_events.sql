CREATE TABLE cf_usage_events (
  guid uuid NOT NULL,
  created_at timestamptz NOT NULL,
  in_use boolean NOT NULL, -- whether this resource is being in-use (i.e., billable) as of this event

  resource_type text NOT NULL, -- "app" or "service" (or name of service?)
  resource_guid uuid NOT NULL,
  resource_name text NOT NULL,

  pricing_plan_id text NOT NULL, -- "app-run", "app-task", "app-staging", or GUID of the service plan
  pricing_metadata JSONB,

  org_guid uuid NOT NULL,
  space_guid uuid NOT NULL,
  service_guid uuid,

  PRIMARY KEY (guid),
  CONSTRAINT created_at_not_zero_value CHECK (created_at > 'epoch'::timestamptz)
);

CREATE INDEX IF NOT EXISTS cf_usage_events_created_at_idx ON cf_usage_events (created_at);

CREATE INDEX IF NOT EXISTS cf_usage_events_fn_safpd_idx ON cf_usage_events (guid, in_use);

CREATE OR REPLACE FUNCTION cf_usage_events_on_day(
  day date
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY SELECT * FROM cf_usage_events WHERE created_at::date = day;
END $$;

-- The principle here:
--   Resources that stopped being used partway through yesterday
--   won't have a timespan that goes right up unto the end of the day.
--   Resources that stopped exactly at midnight will extend onto this
--   new day and do still need to be considered.
--   Resources that haven't stopped are given a timespan right up to
--   midnight, and the time range non-inclusively includes midnight.
--   So we can find non-stopped things that way.
--   Alternately we can add a boolean for timespans that only ended
--   because of the end of the day, but that's less neat.
CREATE OR REPLACE FUNCTION cf_usage_events_still_active_from_previous_days(
  day date
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY SELECT
      cf_usage_events.*
    FROM
      cf_usage_periods
    INNER JOIN cf_usage_events ON
      cf_usage_periods.start_event_guid = cf_usage_events.guid
    WHERE
      UPPER(timerange) = day::TIMESTAMPTZ
      AND cf_usage_events.in_use -- we filter out non-running spans elsewhere, but this makes sure
    ;
END $$;

CREATE OR REPLACE FUNCTION cf_usage_events_relevant_to_day(
  day date
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY
    SELECT * FROM cf_usage_events_still_active_from_previous_days(day)
    UNION ALL
    SELECT * FROM cf_usage_events_on_day(day);
END $$;
