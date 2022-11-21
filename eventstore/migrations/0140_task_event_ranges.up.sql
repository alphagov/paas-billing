-- **do not alter - add new migrations instead**

BEGIN;


CREATE FUNCTION task_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (case
		when (raw_message->>'state') = 'TASK_STARTED' then 'STARTED'
		when (raw_message->>'state') = 'TASK_STOPPED' then 'STOPPED'
	end)::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION task_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT (raw_message->>'state' = 'TASK_STARTED' or raw_message->>'state' = 'TASK_STOPPED')
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION task_event_resource_name(raw_message jsonb) returns text AS $$
	SELECT raw_message->>'task_name';
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION task_event_resource_guid(raw_message jsonb) returns uuid AS $$
	SELECT (raw_message->>'task_guid')::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;


CREATE VIEW task_event_ranges AS SELECT
		id as event_sequence,
		guid::uuid as event_guid,
		'task' as event_type,
		created_at,
		task_event_resource_guid(raw_message) as resource_guid,
		coalesce(
			task_event_resource_name(raw_message),
			(array_remove(
				array_agg(task_event_resource_name(raw_message)) over prev_events
			, NULL))[1]
		) as resource_name,
		'task'::text as resource_type,                              -- resource_type for task resources
		(raw_message->>'org_guid')::uuid as org_guid,
		(raw_message->>'space_guid')::uuid as space_guid,
		'ebfa9453-ef66-450c-8c37-d53dfd931038'::uuid as plan_guid,  -- plan guid for all task resources
		'task'::text as plan_name,                                  -- plan name for all task resources
		'4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid as service_guid,
		'app'::text as service_name,
		coalesce(raw_message->>'instance_count', '1')::numeric as number_of_nodes,
		coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
		'0'::numeric as storage_in_mb,
		task_event_resource_state(raw_message) as state,
		tstzrange(created_at,
			lag(created_at, 1, now()) over resource_states
		) as duration
	FROM
		app_usage_events
	WHERE
		task_event_filter(raw_message)
	WINDOW
		prev_events as (
			partition by task_event_resource_guid(raw_message)
			order by created_at desc, id desc
			rows between current row and unbounded following
		),
		resource_states as (
			partition by task_event_resource_guid(raw_message)
			order by created_at desc, id desc
			rows between 1 preceding and current row
		);

CREATE INDEX task_event_ranges_partial_idx ON app_usage_events((task_event_resource_guid(raw_message)), created_at desc, id desc) WHERE task_event_filter(raw_message);

ANALYZE;


COMMIT;
