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

CREATE OR REPLACE FUNCTION cf_usage_events_still_active_from_previous_days(
  day date
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
BEGIN
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
  RETURN QUERY SELECT
      events.*
    FROM
      spans
    INNER JOIN events ON
      spans.from_seq = events.seq
    WHERE
      UPPER(timespan) = day::TIMESTAMPTZ
      AND running -- we filter out non-running spans elsewhere, but this makes sure
    ;
END $$;

CREATE OR REPLACE FUNCTION cf_usage_events_on_day(
  day date
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY SELECT * FROM events WHERE created_at::date = day;
END $$;

CREATE OR REPLACE FUNCTION cf_usage_events_relevant_to_day(
  day date
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY
    SELECT cf_usage_events_still_active_from_previous_days(day)
    UNION ALL
    SELECT events_on_this_day(day);
END $$;

CREATE OR REPLACE FUNCTION cf_usage_events_in_sequence(
  relevant_cf_usage_events TABLE(...),
) RETURNS SETOF cf_usage_periods LANGUAGE plpgsql AS $$
BEGIN
  RETURN QUERY SELECT
      seq AS from_seq,
      created_at AS from_time,
      LEAD(seq) OVER next_resource_event AS to_seq,
      LEAD(created_at) OVER next_resource_event AS to_time,
      running,
    FROM
      relevant_cf_usage_events
    WINDOW
      next_resource_event AS (
        PARTITION BY thing
        ORDER BY created_at ASC
        ROWS BETWEEN CURRENT ROW AND 1 FOLLOWING
      )
    ORDER BY
      created_at ASC;
END $$;

CREATE OR REPLACE FUNCTION get_cf_usage_periods_for_date(
  day date,
) RETURNS SETOF cf_usage_periods LANGUAGE plpgsql AS $$
DECLARE
  end_of_day TIMESTAMPTZ := LOWER(TSTZRANGE(NOW(), day::TIMESTAMPTZ + INTERVAL '1 DAY'))
BEGIN
  RETURN QUERY
    WITH events_in_sequence AS (
      SELECT * FROM cf_usage_events_in_sequence(cf_usage_events_relevant_to_day(day));
    )
    SELECT
        from_seq,
        to_seq,
        TSTZRANGE(
          from_time,
          COALESCE(to_time, end_of_day),
          '[)'
        ) AS timespan
      FROM
        events_in_sequence
      WHERE
        running
      ORDER BY
        from_seq,
        timespan
      ;
END $$;
