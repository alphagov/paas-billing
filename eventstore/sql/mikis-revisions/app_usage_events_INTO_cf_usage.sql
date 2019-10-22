DO $$
DECLARE
  app_usage_event record;
BEGIN
  FOR app_usage_event IN SELECT * FROM app_usage_events ORDER BY created_at ASC
  LOOP
    CASE
    WHEN app_usage_event.raw_message->>'state' = 'STARTED' AND app_usage_event.raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'
    THEN
      RAISE NOTICE 'STARTED: app_guid=% created_at=%', app_usage_event.raw_message->>'app_guid', app_usage_event.created_at;
      INSERT INTO cf_usage (
        start_event_guid,
        duration,
        resource_type,
        resource_guid,
        resource_name,
        metadata,
        org_guid,
        space_guid,
        plan_guid,
        service_guid
      ) VALUES (
        app_usage_event.guid,
        tstzrange(app_usage_event.created_at, '2099-01-01'::date, '[)'),
        'app-running',
        (app_usage_event.raw_message->>'app_guid')::uuid,
        (app_usage_event.raw_message->>'app_name'),
        jsonb_build_object(
          'number_of_nodes', coalesce(app_usage_event.raw_message->>'instance_count', '1')::numeric,
          'memory_in_mb_per_instance', coalesce(app_usage_event.raw_message->>'memory_in_mb_per_instance', '0')::numeric
        ),
        (app_usage_event.raw_message->>'org_guid')::uuid,
        (app_usage_event.raw_message->>'space_guid')::uuid,
        'f4d4b95a-f55e-4593-8d54-3364c25798c4'::uuid,
        '4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid
      );
    WHEN app_usage_event.raw_message->>'state' = 'STOPPED' AND app_usage_event.raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'
    THEN
      RAISE NOTICE 'STOPPED: app_guid=% created_at=%', app_usage_event.raw_message->>'app_guid', app_usage_event.created_at;
      UPDATE cf_usage
      SET
        stop_event_guid = app_usage_event.guid,
        duration = tstzrange(LOWER(cf_usage.duration), app_usage_event.created_at, '[)')
      WHERE
        resource_type = 'app-running'
        AND resource_guid = (app_usage_event.raw_message->>'app_guid')::uuid
        AND stop_event_guid IS NULL
        AND UPPER(cf_usage.duration) = '2099-01-01'::date;
    ELSE
    END CASE;
  END LOOP;
END;
$$ LANGUAGE plpgsql;
