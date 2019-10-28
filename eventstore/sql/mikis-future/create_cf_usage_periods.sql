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

CREATE INDEX IF NOT EXISTS cf_usage_periods_duration_idx ON cf_usage_periods USING gist (duration);

-- If provided the current date it only computes up to the present moment.
CREATE OR REPLACE FUNCTION infer_cf_usage_periods(day DATE)
RETURNS SETOF cf_usage_periods LANGUAGE plpgsql AS $$
BEGIN
  ASSERT day <= NOW()::date, 'Provided date was in the future';

  RETURN QUERY SELECT
    guid AS start_guid,
    LEAD(guid, 1, null) OVER next_usage_event_of_this_resource AS stop_guid,
    tstzrange(
      created_at,
      LEAD(
        created_at,
        1,
        NOW()
      ) OVER next_usage_event_of_this_resource,
      '[)'
    ) AS duration,
    in_use
  FROM
    cf_usage_events
  WHERE
    created_at::date = $1 --- NOOOOO! Multi-day events are ignored on all but the start day.
  WINDOW
    next_usage_event_of_this_resource AS (
      PARTITION BY resource_type, resource_guid, pricing_plan_id
      ORDER BY created_at ASC
      ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
    )
  ORDER BY
    created_at ASC
  ;
END $$;
