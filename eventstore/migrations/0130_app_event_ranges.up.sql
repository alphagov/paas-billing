-- **do not alter - add new migrations instead**

BEGIN;


CREATE FUNCTION app_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (raw_message->>'state')::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION app_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT (raw_message->>'state' = 'STARTED' or raw_message->>'state' = 'STOPPED')
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION app_event_resource_name(raw_message jsonb) returns text AS $$
	SELECT raw_message->>'app_name';
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION app_event_resource_guid(raw_message jsonb) returns uuid AS $$
	SELECT (raw_message->>'app_guid')::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;


CREATE VIEW app_event_ranges AS SELECT
		event_sequence,
		event_guid,
		event_type,
		created_at,
		resource_guid,
		coalesce(
			resource_name,
			(array_remove(
				array_agg(resource_name) over prev_events
			, NULL))[1]
		) as resource_name,
		resource_type,
		org_guid,
		space_guid,
		plan_guid,
		plan_name,
		service_guid,
		service_name,
		number_of_nodes,
		memory_in_mb,
		storage_in_mb,
		state,
		tstzrange(created_at,
			lag(created_at, 1, now()) over resource_states
		) as duration
	FROM (
		SELECT
			id as event_sequence,
			guid::uuid as event_guid,
			'app' as event_type,
			created_at,
			app_event_resource_guid(raw_message) as resource_guid,
			app_event_resource_name(raw_message) as resource_name,
			'app'::text as resource_type,                              -- resource_type for compute resources
			(raw_message->>'org_guid')::uuid as org_guid,
			(raw_message->>'space_guid')::uuid as space_guid,
			'f4d4b95a-f55e-4593-8d54-3364c25798c4'::uuid as plan_guid, -- plan guid for all compute resources
			'app'::text as plan_name,                                  -- plan name for all compute resources
			'4f6f0a18-cdd4-4e51-8b6b-dc39b696e61b'::uuid as service_guid,
			'app'::text as service_name,
			coalesce(raw_message->>'instance_count', '1')::numeric as number_of_nodes,
			coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
			'0'::numeric as storage_in_mb,
			app_event_resource_state(raw_message) as state
		FROM
			app_usage_events
		WHERE
			app_event_filter(raw_message)
	) as sq
	WINDOW
		prev_events as (
			partition by resource_guid
			order by created_at desc, event_sequence desc
			rows between current row and unbounded following
		),
		resource_states as (
			partition by resource_guid
			order by created_at desc, event_sequence desc
			rows between 1 preceding and current row
		);


CREATE INDEX app_event_ranges_partial_idx ON app_usage_events((app_event_resource_guid(raw_message)), created_at desc, id desc) WHERE app_event_filter(raw_message);

ANALYZE;


COMMIT;
