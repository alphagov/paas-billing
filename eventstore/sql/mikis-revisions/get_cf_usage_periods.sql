-- Takes one parameter: the date to compute events for.
-- If provided the current date it only computes up to the present moment.
-- Do not provide future dates. That results in undefined behaviour.

CREATE OR REPLACE FUNCTION get_cf_usage_periods(day DATE)
-- RETURNS SETOF cf_usage_periods
RETURNS TABLE(
  start_guid uuid,
  end_guid uuid,
  in_use boolean,
  duration tstzrange
)
AS $$
BEGIN
  ASSERT day <= NOW()::date, 'Provided date was in the future'

  RETURN QUERY SELECT
    guid AS start_guid,
    in_use,
    LEAD(guid, 1, null) OVER next_usage_event_of_this_resource AS stop_guid,
    tstzrange(
      created_at,
      LEAD(
        created_at,
        1,
        LOWER(NOW(), day)::date + INTERVAL '1 day'
      ) OVER next_usage_event_of_this_resource,
      '[)'
    ) AS duration
  FROM
    cf_usage_events
  WHERE
    created_at::date = $1
  WINDOW
    next_usage_event_of_this_resource AS (
      PARTITION BY resource_type, resource_guid
      ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
    )
  ORDER BY
    created_at ASC
  ;
END;
$$ LANGUAGE 'plpgsql'
