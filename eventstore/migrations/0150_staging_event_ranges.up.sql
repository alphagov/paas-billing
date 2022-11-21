-- **do not alter - add new migrations instead**

BEGIN;


CREATE FUNCTION staging_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (case
		when (raw_message->>'state') = 'CREATED' then 'STARTED'
		when (raw_message->>'state') = 'DELETED' then 'STOPPED'
		when (raw_message->>'state') = 'UPDATED' then 'STARTED'
	end)::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION staging_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT (raw_message->>'state' = 'STAGING_STARTED' or raw_message->>'state' = 'STAGING_STOPPED')
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION staging_event_resource_name(raw_message jsonb) returns text AS $$
	SELECT raw_message->>'parent_app_name';
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION staging_event_resource_guid(raw_message jsonb) returns uuid AS $$
	SELECT (raw_message->>'parent_app_guid')::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;


CREATE VIEW staging_event_ranges AS SELECT
		id as event_sequence,
		guid::uuid as event_guid,
		'staging' as event_type,
		created_at,
		staging_event_resource_guid(raw_message) as resource_guid,
		coalesce(
			staging_event_resource_name(raw_message),
			(array_remove(
				array_agg(staging_event_resource_name(raw_message)) over prev_events
			, NULL))[1]
		) as resource_name,
		'app'::text as resource_type,                              -- resource_type for staging of resources
		(raw_message->>'org_guid')::uuid as org_guid,
		(raw_message->>'space_guid')::uuid as space_guid,
		'9d071c77-7a68-4346-9981-e8dafac95b6f'::uuid as plan_guid,  -- plan guid for all staging of resources
		'staging'::text as plan_name,                                  -- plan name for all staging of resources
		'4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid as service_guid,
		'app'::text as service_name,
		'1'::numeric as number_of_nodes,
		coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
		'0'::numeric as storage_in_mb,
		staging_event_resource_state(raw_message) as state,
		tstzrange(created_at,
			lag(created_at, 1, created_at) over resource_states
		) as duration
	FROM
		app_usage_events
	WHERE
		staging_event_filter(raw_message)
	WINDOW
		prev_events as (
			partition by staging_event_resource_guid(raw_message)
			order by created_at desc, id desc
			rows between current row and unbounded following
		),
		resource_states as (
			partition by staging_event_resource_guid(raw_message)
			order by created_at desc, id desc
			rows between 1 preceding and current row
		);

CREATE INDEX staging_event_ranges_partial_idx ON app_usage_events((staging_event_resource_guid(raw_message)), created_at desc, id desc) WHERE staging_event_filter(raw_message);

ANALYZE;


COMMIT;
