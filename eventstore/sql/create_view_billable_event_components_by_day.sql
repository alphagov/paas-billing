-- When creating or recreating a usage and adoption spreadsheet
-- this file can be used to create the tables in an idempotent fashion

CREATE MATERIALIZED VIEW IF NOT EXISTS billable_event_components_by_day AS
WITH
  billable_event_components_we_want AS (
    SELECT *
    FROM billable_event_components
  ),

  billable_event_component_series AS (
    SELECT
      event_guid,

      resource_guid, resource_name, resource_type,

      org_guid, org_name,

      space_guid, space_name,

      plan_guid, plan_valid_from, plan_name,

      number_of_nodes, memory_in_mb, storage_in_mb,

      component_name, component_formula,

      currency_code, currency_rate,

      vat_code, vat_rate,

      cost_for_duration, duration,

      -- unroll event duration (start -> end) into rows where each row is a day
      -- so we can group and query by day and by month using a regular index
      GENERATE_SERIES(
        LOWER(duration),
        UPPER(duration),
        '1 day'::interval
      ) AS day

    FROM billable_event_components_we_want
  ),

  costed_billable_event_component_series AS (
    SELECT
      event_guid,

      resource_guid, resource_name, resource_type,

      org_guid, org_name,

      space_guid, space_name,

      plan_guid, plan_valid_from, plan_name,

      number_of_nodes, memory_in_mb, storage_in_mb,

      component_name, component_formula,

      currency_code, currency_rate,

      vat_code, vat_rate,

      cost_for_duration, duration,

      day::date as day,

      -- intersect whole day and duration to get minimal complete event
      -- duration for day
      TSTZRANGE(
        DATE_TRUNC('day', day),
        DATE_TRUNC('day', day) + INTERVAL '1 day' - INTERVAL '1 second',
        '[]'
      ) * duration AS day_duration

    FROM billable_event_component_series
  ),

  daily_costed_billable_event_components AS (
    SELECT
      day,

      event_guid,

      resource_guid, resource_name, resource_type,

      org_guid, org_name,

      space_guid, space_name,

      plan_guid, plan_valid_from, plan_name,

      number_of_nodes, memory_in_mb, storage_in_mb,

      component_name, component_formula,

      currency_code, currency_rate,

      vat_code, vat_rate,

      cost_for_duration, duration, day_duration,

      -- compute cost for this event for this day
      -- $duration_seconds_of_day_event / $duration_seconds_of_event
      -- we are losing millisecond precision here
      (
        EXTRACT(EPOCH FROM (UPPER(day_duration) - LOWER(day_duration)))
      ) / (
        EXTRACT(EPOCH FROM (UPPER(duration) - LOWER(duration)))
      ) * cost_for_duration AS cost

    FROM costed_billable_event_component_series
  )

SELECT *
FROM daily_costed_billable_event_components;

CREATE INDEX IF NOT EXISTS
    billable_event_components_by_day_day
  ON
    billable_event_components_by_day (day)
;

CREATE INDEX IF NOT EXISTS
    billable_event_components_by_day_plan_guid
  ON
    billable_event_components_by_day (plan_guid)
;

CREATE INDEX IF NOT EXISTS
    billable_event_components_by_day_org_guid
  ON
    billable_event_components_by_day (org_guid)
;

CREATE INDEX IF NOT EXISTS
    billable_event_components_by_day_space_guid
  ON
    billable_event_components_by_day (space_guid)
;

REFRESH MATERIALIZED VIEW billable_event_components_by_day;

