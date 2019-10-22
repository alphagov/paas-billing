-- HEAVILY WORK-IN-PROGRESS IDEAS OF HOW TO REFINE EVENTS.
--
-- *** The ideas from this led to my `cf_usage` table. See that instead. ***
--
-- This is a modification of how we already refine events from the raw `app_usage_events` and
-- `service_usage_events` tables. Instead of the hardcoded pricing metadata fields such as
-- `number_of_nodes`, we have a JSONB `metadata` field for arbitrary pricing metadata. This makes
-- things a bit saner. I've stopped joining the org and space names here, to reduce the amount of
-- SQL.

with
raw_events as (
  (
    -- Process raw usage events for CF apps starting and stopping
    select
      id as event_sequence,
      guid::uuid as event_guid,

      created_at,
      (raw_message->>'state')::resource_state as state,

      'app-running' as resource_type,
      (raw_message->>'app_guid')::uuid as resource_guid,
      (raw_message->>'app_name') as resource_name,
      jsonb_build_object(
        'number_of_nodes', coalesce(raw_message->>'instance_count', '1')::numeric,
        'memory_in_mb_per_instance', coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric
      ) as metadata,

      (raw_message->>'org_guid')::uuid as org_guid,
      (raw_message->>'space_guid')::uuid as space_guid,
      'f4d4b95a-f55e-4593-8d54-3364c25798c4'::uuid as plan_guid, -- plan guid for all compute resources
      '4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid as service_guid
    from
      app_usage_events
    where
      (raw_message->>'state' = 'STARTED' or raw_message->>'state' = 'STOPPED')
      and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
  ) union all (
    -- Process raw usage events for CF tasks starting and stopping
    select
      id as event_sequence,
      guid::uuid as event_guid,

      created_at,
      (case
        when (raw_message->>'state') = 'TASK_STARTED' then 'STARTED'
        when (raw_message->>'state') = 'TASK_STOPPED' then 'STOPPED'
      end)::resource_state as state,

      'app-task'::text as resource_type,
      (raw_message->>'task_guid')::uuid as resource_guid,
      (raw_message->>'task_name') as resource_name,
      jsonb_build_object(
        'number_of_nodes', coalesce(raw_message->>'instance_count', '1')::numeric,
        'memory_in_mb_per_instance', coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric
      ) as metadata,

      (raw_message->>'org_guid')::uuid as org_guid,
      (raw_message->>'space_guid')::uuid as space_guid,
      'ebfa9453-ef66-450c-8c37-d53dfd931038'::uuid as plan_guid,  -- plan guid for all task resources
      '4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid as service_guid
    from
      app_usage_events
    where
      (raw_message->>'state' = 'TASK_STARTED' or raw_message->>'state' = 'TASK_STOPPED')
      and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
  ) union all (
    -- Process raw usage events for CF staging starting and stopping
    select
      id as event_sequence,
      guid::uuid as event_guid,

      created_at,
      (case
        when (raw_message->>'state') = 'STAGING_STARTED' then 'STARTED'
        when (raw_message->>'state') = 'STAGING_STOPPED' then 'STOPPED'
      end)::resource_state as state,

      'app-staging' as resource_type,
      (raw_message->>'parent_app_guid')::uuid as resource_guid,
      (raw_message->>'parent_app_name') as resource_name,
      jsonb_build_object(
        'memory_in_mb_per_instance', coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric
      ) as metadata,

      (raw_message->>'org_guid')::uuid as org_guid,
      (raw_message->>'space_guid')::uuid as space_guid,
      '9d071c77-7a68-4346-9981-e8dafac95b6f'::uuid as plan_guid,  -- plan guid for all staging of resources
      '4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid as service_guid
    from
      app_usage_events
    where
      (raw_message->>'state' = 'STAGING_STARTED' or raw_message->>'state' = 'STAGING_STOPPED')
      and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
  ) union all (
    -- Process raw usage events for CF services
    select
      id as event_sequence,
      guid::uuid as event_guid,

      created_at,
      (case
        when (raw_message->>'state') = 'CREATED' then 'STARTED'
        when (raw_message->>'state') = 'DELETED' then 'STOPPED'
        when (raw_message->>'state') = 'UPDATED' then 'STARTED'
      end)::resource_state as state,

      'service' as resource_type,
      (raw_message->>'service_instance_guid')::uuid as resource_guid,
      (raw_message->>'service_instance_name') as resource_name,
      jsonb_build_object() as metadata,

      (raw_message->>'org_guid')::uuid as org_guid,
      (raw_message->>'space_guid')::uuid as space_guid,
      (raw_message->>'service_plan_guid')::uuid as plan_guid,
      (raw_message->>'service_guid')::uuid as service_guid
    from
      service_usage_events
    where
      raw_message->>'service_instance_type' = 'managed_service_instance'
      and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
  )
)
