-- Alternate idea:
--
--     Times are stored in the events table. We don't actually need to
--     copy that data.
--
--     TO FIGURE OUT WHAT WAS RUNNING ON A DAY, WE ONLY NEED TO KNOW:
--
--     - which things are still running at the start of the next day,
--       doable with a simple boolean
--       provided by `cf_usage_events_still_active_from_previous_days`
--
--     - what changes to things starting and stopping happened on that
--       day
--       provided by `cf_usage_events_on_day`
--
--     These are both direct from `cf_usage_events`, and we combine
--     them in the output of `cf_usage_events_relevant_to_day`.
--
--     Commonplace postgres lets us order the events by time and then
--     get for every event get the next event for that resource.

-- IDEAS FOR PROPERTIES WE CAN TEST:
-- (Where a "span" is generically a period of time, e.g., a row in `cf_usage_periods`.)
--
-- No spans should start from an event which isn't running
-- Spans should always be created between following events
-- Precisely one span should start from each running event
-- Multi-day running events should have one span per day (kinda clashes with above when described simply)
-- One span per day should start from each multi-day running event
-- Span timeranges should not include their upper bound
--
-- Running these functions 700 times for every day we've billed so far will
-- surely(?) be slower than how we currently do things once for all days.
-- However, this approach should scale well. Stopped resources aren't considered
-- at all for future days. We only figure out the sequence of events for today.
-- Question is–where is the crossover point and will we reach it within two years?

CREATE OR REPLACE FUNCTION cf_usage_events_relevant_to_day_in_sequence(
  day date
) RETURNS TABLE(
  start_event_guid UUID,
  from_time TIMESTAMPTZ,
  stop_event_guid UUID,
  to_time TIMESTAMPTZ,
  in_use BOOLEAN
) LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY SELECT
      guid AS start_event_guid,
      max_time(created_at, day::TIMESTAMPTZ) AS from_time,
      LEAD(guid) OVER next_resource_event AS stop_event_guid,
      LEAD(created_at) OVER next_resource_event AS to_time,
      cf_usage_events_relevant_to_day.in_use
    FROM
      cf_usage_events_relevant_to_day(day)
    WINDOW
      next_resource_event AS (
        PARTITION BY resource_type, resource_guid, pricing_plan_id
        ORDER BY created_at ASC
        ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
      )
    ORDER BY
      created_at ASC
    ;
END $$;

CREATE OR REPLACE FUNCTION get_cf_usage_periods_for_date(
  day date
) RETURNS SETOF cf_usage_periods LANGUAGE plpgsql AS $$
DECLARE
  end_of_day_or_now TIMESTAMPTZ := min_time(TSTZRANGE(NOW(), day::TIMESTAMPTZ + INTERVAL '1 DAY'));
BEGIN
  RETURN QUERY
    WITH
      events_in_sequence AS (
        SELECT * FROM cf_usage_events_relevant_to_day_in_sequence(day)
      )
    SELECT
        start_event_guid,
        stop_event_guid,
        TSTZRANGE(
          from_time,
          COALESCE(to_time, end_of_day_or_now),
          '[)'
        ) AS timerange
      FROM
        events_in_sequence
      WHERE
        --to_time > from_time
        --AND in_use
        in_use
      ORDER BY
        start_event_guid,
        timerange
      ;
END $$;
