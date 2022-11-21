-- **do not alter - add new migrations instead**

BEGIN;


CREATE FUNCTION service_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (case
		when (raw_message->>'state') = 'CREATED' then 'STARTED'
		when (raw_message->>'state') = 'DELETED' then 'STOPPED'
		when (raw_message->>'state') = 'UPDATED' then 'STARTED'
	end)::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION service_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT raw_message->>'service_instance_type' = 'managed_service_instance'
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;


CREATE FUNCTION compose_service_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-';
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION service_instance_guid_if_created(raw_message jsonb) returns uuid AS $$
	SELECT (case
		when (raw_message->>'state') = 'CREATED' then raw_message->>'service_instance_guid'
		-- else NULL, which won't match against anything when used in a join
	end)::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION uuid_from_data_deployment(raw_message jsonb) returns uuid AS $$
	SELECT substring(
		raw_message->'data'->>'deployment'
		from '[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}$'
	)::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;


CREATE VIEW service_event_ranges AS WITH
	raw_events as (
		(
			select
				id as event_sequence,
				guid::uuid as event_guid,
				'service' as event_type,
				created_at,
				(raw_message->>'service_instance_guid')::uuid as resource_guid,
				(raw_message->>'service_instance_name') as resource_name,
				'service' as resource_type,
				(raw_message->>'org_guid')::uuid as org_guid,
				(raw_message->>'space_guid')::uuid as space_guid,
				(raw_message->>'service_plan_guid')::uuid as plan_guid,
				(raw_message->>'service_plan_name') as plan_name,
				(raw_message->>'service_guid')::uuid as service_guid,
				(raw_message->>'service_label') as service_name,
				NULL::numeric as number_of_nodes,
				NULL::numeric as memory_in_mb,
				NULL::numeric as storage_in_mb,
				service_event_resource_state(raw_message) as state
			from
				service_usage_events
			where
				service_event_filter(raw_message)
		) union all (
			select
				s.id as event_sequence,
				uuid_generate_v4() as event_guid,
				'service' as event_type,
				c.created_at::timestamptz as created_at,
				uuid_from_data_deployment(c.raw_message) as resource_guid,
				(case
					when s.created_at > c.created_at then (s.raw_message->>'service_instance_name')
					else NULL::text
				end) as resource_name,
				'service'::text as resource_type,
				(s.raw_message->>'org_guid')::uuid as org_guid,
				(s.raw_message->>'space_guid')::uuid as space_guid,
				(s.raw_message->>'service_plan_guid')::uuid as plan_guid,
				(s.raw_message->>'service_plan_name') as plan_name,
				(s.raw_message->>'service_guid')::uuid as service_guid,
				(s.raw_message->>'service_label') as service_name,
				NULL::numeric as number_of_nodes,
				(pg_size_bytes(c.raw_message->'data'->>'memory') / 1024 / 1024)::numeric as memory_in_mb,
				(pg_size_bytes(c.raw_message->'data'->>'storage') / 1024 / 1024)::numeric as storage_in_mb,
				'STARTED'::resource_state as state
			from
				compose_audit_events c
			left join
				service_usage_events s
			on
				service_instance_guid_if_created(s.raw_message) = uuid_from_data_deployment(c.raw_message)
			where
				compose_service_event_filter(s.raw_message)
		)
	)
	SELECT
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
		coalesce(
			memory_in_mb,
			(array_remove(
				array_agg(memory_in_mb) over prev_events
			, NULL))[1]
		) as memory_in_mb,
		coalesce(
			storage_in_mb,
			(array_remove(
				array_agg(storage_in_mb) over prev_events
			, NULL))[1]
		) as storage_in_mb,
		state,
		tstzrange(created_at,
			lag(created_at, 1, now()) over resource_states
		) as duration
	FROM
		raw_events
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


CREATE INDEX service_usage_events_service_resource_state_part_idx ON service_usage_events ((service_event_resource_state(raw_message))) WHERE service_event_filter(raw_message);

CREATE INDEX service_usage_events_svc_inst_guid_if_crtd_part_idx ON service_usage_events ((service_instance_guid_if_created(raw_message))) WHERE compose_service_event_filter(raw_message);

CREATE INDEX compose_audit_events_uuid_frm_data_dpmt_idx ON compose_audit_events ((uuid_from_data_deployment(raw_message)));


COMMIT;
