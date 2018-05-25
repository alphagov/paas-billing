CREATE TABLE events (
	event_guid uuid PRIMARY KEY NOT NULL,
	resource_guid uuid NOT NULL,
	resource_name text NOT NULL,
	resource_type text NOT NULL,
	org_guid uuid NOT NULL,
	space_guid uuid NOT NULL,
	duration tstzrange NOT NULL,
	plan_guid uuid NOT NULL,
	plan_name text NOT NULL,
	number_of_nodes integer,
	memory_in_mb integer,
	storage_in_mb integer,

	CONSTRAINT duration_must_not_be_empty CHECK (not isempty(duration))
);

-- extract useful stuff from usage events
-- we treat both apps and services as "resources" so normalize the fields
-- we normalize states to just STARTED/STOPPED because we treat consecutive STARTED to mean "update"
INSERT INTO events with
	raw_events as (
		(
			select
				id as event_sequence,
				guid::uuid as event_guid,
				created_at,
				(raw_message->>'app_guid')::uuid as resource_guid,
				(raw_message->>'app_name') as resource_name,
				'app'::text as resource_type,                              -- resource_type for compute resources
				(raw_message->>'org_guid')::uuid as org_guid,
				(raw_message->>'space_guid')::uuid as space_guid,
				'f4d4b95a-f55e-4593-8d54-3364c25798c4'::uuid as plan_guid, -- plan guid for all compute resources
				'app'::text as plan_name,                                  -- plan name for all compute resources
				coalesce(raw_message->>'instance_count', '1')::numeric as number_of_nodes,
				coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
				'0'::numeric as storage_in_mb,
				(raw_message->>'state')::resource_state as state
			from
				app_usage_events
			where
				(raw_message->>'state' = 'STARTED' or raw_message->>'state' = 'STOPPED')
				and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
		) union all (
			select
				id as event_sequence,
				guid::uuid as event_guid,
				created_at,
				(raw_message->>'service_instance_guid')::uuid as resource_guid,
				(raw_message->>'service_instance_name') as resource_name,
				(raw_message->>'service_label') as resource_type,
				(raw_message->>'org_guid')::uuid as org_guid,
				(raw_message->>'space_guid')::uuid as space_guid,
				(raw_message->>'service_plan_guid')::uuid as plan_guid,
				(raw_message->>'service_plan_name') as plan_name,
				NULL::numeric as number_of_nodes,
				NULL::numeric as memory_in_mb,
				NULL::numeric as storage_in_mb,
				(case
					when (raw_message->>'state') = 'CREATED' then 'STARTED'
					when (raw_message->>'state') = 'DELETED' then 'STOPPED'
					when (raw_message->>'state') = 'UPDATED' then 'STARTED'
				end)::resource_state as state
			from
				service_usage_events
			where
				raw_message->>'service_instance_type' = 'managed_service_instance'
				and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
		) union all (
			select
				id as event_sequence,
				guid::uuid as event_guid,
				created_at,
				(raw_message->>'task_guid')::uuid as resource_guid,
				(raw_message->>'task_name') as resource_name,
				'task'::text as resource_type,                              -- resource_type for task resources
				(raw_message->>'org_guid')::uuid as org_guid,
				(raw_message->>'space_guid')::uuid as space_guid,
				'ebfa9453-ef66-450c-8c37-d53dfd931038'::uuid as plan_guid,  -- plan guid for all task resources
				'task'::text as plan_name,                                  -- plan name for all task resources
				coalesce(raw_message->>'instance_count', '1')::numeric as number_of_nodes,
				coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
				'0'::numeric as storage_in_mb,
				(case
					when (raw_message->>'state') = 'TASK_STARTED' then 'STARTED'
					when (raw_message->>'state') = 'TASK_STOPPED' then 'STOPPED'
				end)::resource_state as state
			from
				app_usage_events
			where
				(raw_message->>'state' = 'TASK_STARTED' or raw_message->>'state' = 'TASK_STOPPED')
				and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
		) union all (
			select
				id as event_sequence,
				guid::uuid as event_guid,
				created_at,
				(raw_message->>'parent_app_guid')::uuid as resource_guid,
				(raw_message->>'parent_app_name') as resource_name,
				'app'::text as resource_type,                              -- resource_type for staging of resources
				(raw_message->>'org_guid')::uuid as org_guid,
				(raw_message->>'space_guid')::uuid as space_guid,
				'f4d4b95a-f55e-4593-8d54-3364c25798c4'::uuid as plan_guid,  -- plan guid for all staging of resources
				'app'::text as plan_name,                                  -- plan name for all staging of resources
				coalesce(raw_message->>'instance_count', '1')::numeric as number_of_nodes,
				coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
				'0'::numeric as storage_in_mb,
				(case
				 when (raw_message->>'state') = 'STAGING_STARTED' then 'STARTED'
				 when (raw_message->>'state') = 'STAGING_STOPPED' then 'STOPPED'
				 end)::resource_state as state
			from
				app_usage_events
			where
				(raw_message->>'state' = 'STAGING_STARTED' or raw_message->>'state' = 'STAGING_STOPPED')
				and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
		)
	),
	event_ranges as (
		select
			*,
			tstzrange(created_at, lead(created_at, 1, now()) over resource_states) as duration
		from
			raw_events
		window
			resource_states as (partition by resource_guid order by event_sequence rows between current row and 1 following)
		order by
			event_sequence
	)
	select
		event_guid,
		resource_guid,
		resource_name,
		resource_type,
		org_guid,
		space_guid,
		duration,
		plan_guid,
		plan_name,
		number_of_nodes,
		memory_in_mb,
		storage_in_mb
	from
		event_ranges
	where
		state = 'STARTED'
		and not isempty(duration)
;

CREATE INDEX events_org_idx ON events (org_guid);
CREATE INDEX events_space_idx ON events (space_guid);
CREATE INDEX events_resource_idx ON events (resource_guid);
CREATE INDEX events_duration_idx ON events using gist (duration);
CREATE INDEX events_plan_idx ON events (plan_guid);
