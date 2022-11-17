CREATE TABLE events_temp (
	event_guid uuid PRIMARY KEY NOT NULL,
	resource_guid uuid NOT NULL,
	resource_name text NOT NULL,
	resource_type text NOT NULL,
	org_guid uuid NOT NULL,
	org_name text NOT NULL,
	space_guid uuid NOT NULL,
	space_name text NOT NULL,
	duration tstzrange NOT NULL,
	plan_guid uuid NOT NULL,
	plan_name text NOT NULL,
	service_guid uuid,
	service_name text,
	number_of_nodes integer,
	memory_in_mb integer,
	storage_in_mb integer,

	CONSTRAINT duration_must_not_be_empty CHECK (not isempty(duration))
);

-- extract useful stuff from usage events
-- we treat both apps and services as "resources" so normalize the fields
-- we normalize states to just STARTED/STOPPED because we treat consecutive STARTED to mean "update"
INSERT INTO events_temp with
	raw_events as (
		(
			select
				id as event_sequence,
				guid::uuid as event_guid,
				'app' as event_type,
				created_at,
				(raw_message->>'app_guid')::uuid as resource_guid,
				(raw_message->>'app_name') as resource_name,
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
			from
				app_usage_events
			where
				app_event_filter(raw_message)
		) union all (
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
				id as event_sequence,
				guid::uuid as event_guid,
				'task' as event_type,
				created_at,
				(raw_message->>'task_guid')::uuid as resource_guid,
				(raw_message->>'task_name') as resource_name,
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
				task_event_resource_state(raw_message) as state
			from
				app_usage_events
			where
				task_event_filter(raw_message)
		) union all (
			select
				id as event_sequence,
				guid::uuid as event_guid,
				'staging' as event_type,
				created_at,
				(raw_message->>'parent_app_guid')::uuid as resource_guid,
				(raw_message->>'parent_app_name') as resource_name,
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
				staging_event_resource_state(raw_message) as state
			from
				app_usage_events
			where
				staging_event_filter(raw_message)
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
	),
	raw_events_with_injected_values as (
		select
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
			state
		from
			raw_events
		window
			prev_events as (
				partition by resource_guid, event_type
				order by created_at desc, event_sequence desc
				rows between current row and unbounded following
			)
		order by
			created_at, event_sequence
	),
	event_ranges as (
		select
			*,
			tstzrange(created_at,
				lead(created_at, 1,
					case when event_type = 'staging' then created_at
					else now() end
				) over resource_states
			) as duration
		from
			raw_events_with_injected_values
		window
			resource_states as (
				partition by resource_guid, event_type
				order by created_at, event_sequence
				rows between current row and 1 following
			)
	),
	valid_service_plans as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by guid order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			service_plans
	),
	valid_services as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by guid order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			services
	),
	valid_orgs as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by guid order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			orgs
	),
	valid_spaces as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by guid order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			spaces
	)

	select
		event_guid,
		resource_guid,
		resource_name,
		resource_type,
		org_guid,
		coalesce(vo.name, org_guid::text) as org_name,
		space_guid,
		coalesce(vspace.name, space_guid::text) as space_name,
		duration,
		(case
			when resource_type = 'service'
			then coalesce(uuid_or_placeholder(vsp.unique_id), 'd5091c33-2f9d-4b15-82dc-4ad69717fc03')::uuid
			else plan_guid
		end) as plan_guid,
		coalesce(vsp.name, plan_name) as plan_name,
		coalesce(vs.guid, ev.service_guid) as service_guid,
		coalesce(vs.label, ev.service_name) as service_name,
		number_of_nodes,
		memory_in_mb,
		storage_in_mb
	from
		event_ranges ev
	left join
		valid_service_plans vsp on ev.plan_guid = vsp.guid
		and upper(ev.duration) <@ vsp.valid_for
	left join
		valid_services vs on vsp.service_guid = vs.guid
		and upper(ev.duration) <@ vs.valid_for
	left join
		valid_orgs vo on ev.org_guid = vo.guid
		and upper(ev.duration) <@ vo.valid_for
	left join
		valid_spaces vspace on ev.space_guid = vspace.guid
		and upper(ev.duration) <@ vspace.valid_for
	where
		state = 'STARTED'
		and not isempty(duration)
	order by
		event_sequence, event_guid
;

CREATE INDEX events_org_temp_idx ON events_temp (org_guid);
CREATE INDEX events_space_temp_idx ON events_temp (space_guid);
CREATE INDEX events_resource_temp_idx ON events_temp (resource_guid);
CREATE INDEX events_duration_temp_idx ON events_temp using gist (duration);
CREATE INDEX events_plan_temp_idx ON events_temp (plan_guid);

DROP TABLE IF EXISTS events;
ALTER TABLE events_temp RENAME TO events;
ALTER INDEX events_temp_pkey RENAME TO events_pkey;
ALTER INDEX events_org_temp_idx RENAME TO events_org_idx;
ALTER INDEX events_space_temp_idx RENAME TO events_space_idx;
ALTER INDEX events_resource_temp_idx RENAME TO events_resource_idx;
ALTER INDEX events_duration_temp_idx RENAME TO events_duration_idx;
ALTER INDEX events_plan_temp_idx RENAME TO events_plan_idx;
