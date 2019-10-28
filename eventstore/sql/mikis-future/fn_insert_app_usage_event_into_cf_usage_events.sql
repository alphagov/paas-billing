CREATE OR REPLACE FUNCTION insert_app_usage_event_into_cf_usage_events(
  guid uuid,
  created_at TIMESTAMPTZ,
  raw_message JSONB
) RETURNS SETOF cf_usage_events LANGUAGE plpgsql AS $$
DECLARE
  in_use BOOLEAN;
  pricing_plan_id TEXT;
  pricing_metadata JSONB;
  resource_guid uuid;
  resource_name TEXT;
BEGIN
  CASE raw_message->>'state'
  WHEN 'STARTED', 'TASK_STARTED', 'STAGING_STARTED' THEN
    in_use := TRUE;
  WHEN 'STOPPED', 'TASK_STOPPED', 'STAGING_STOPPED' THEN
    in_use := FALSE;
  ELSE
    RAISE EXCEPTION 'Unknown state % for event GUID %', raw_message->>'state', guid;
  END CASE;

  CASE raw_message->>'state'
  WHEN 'STARTED', 'STOPPED' THEN
    pricing_plan_id := 'app-run';
    pricing_metadata := jsonb_build_object(
      'number_of_nodes', (raw_message->>'instance_count')::numeric,
      'memory_in_mb_per_instance', (raw_message->>'memory_in_mb_per_instance')::numeric
    );
    resource_guid := (raw_message->>'app_guid')::uuid;
    resource_name := (raw_message->>'app_name');
  WHEN 'TASK_STARTED', 'TASK_STOPPED' THEN
    pricing_plan_id := 'task-run';
    pricing_metadata := jsonb_build_object(
      'number_of_nodes', (raw_message->>'instance_count')::numeric,
      'memory_in_mb_per_instance', (raw_message->>'memory_in_mb_per_instance')::numeric
    );
    resource_guid := (raw_message->>'task_guid')::uuid;
    resource_name := (raw_message->>'task_name');
  WHEN 'STAGING_STARTED', 'STAGING_STOPPED' THEN
    pricing_plan_id := 'staging-run';
    pricing_metadata := jsonb_build_object(
      'memory_in_mb_per_instance', (raw_message->>'memory_in_mb_per_instance')::numeric
    );
    resource_guid := (raw_message->>'parent_app_guid')::uuid;
    resource_name := (raw_message->>'parent_app_name');
  END CASE;

  RETURN QUERY INSERT INTO
    cf_usage_events (
      guid,
      created_at,
      in_use,
      resource_type,
      resource_guid,
      resource_name,
      pricing_plan_id,
      pricing_metadata,
      org_guid,
      space_guid
    ) VALUES (
      guid,
      created_at,
      in_use,
      'app',
      resource_guid,
      resource_name,
      pricing_plan_id,
      pricing_metadata,
      (raw_message->>'org_guid')::uuid,
      (raw_message->>'space_guid')::uuid
    )
    RETURNING *
    ;
END $$;
