
CREATE TEMPORARY TABLE events_temp AS WITH
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
				(case
					when (raw_message->>'state') = 'STAGING_STARTED' then 'STARTED'
					when (raw_message->>'state') = 'STAGING_STOPPED' then 'STOPPED'
				end)::resource_state as state
			from
				app_usage_events
			where
				(raw_message->>'state' = 'STAGING_STARTED' or raw_message->>'state' = 'STAGING_STOPPED')
				and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
		) union all (
			select
				s.id as event_sequence,
				uuid_generate_v4() as event_guid,
				'service' as event_type,
				c.created_at::timestamptz as created_at,
				substring(
					c.raw_message->'data'->>'deployment'
					from '[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}$'
				)::uuid as resource_guid,
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
				s.raw_message->>'service_instance_guid' = substring(
					c.raw_message->'data'->>'deployment'
					from '[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}$'
				) AND s.raw_message->>'state' = 'CREATED'
			where
				s.raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
		)
	),
	raw_events_with_injected_values as (
		select
			event_sequence,
			event_guid,
			event_type,
			created_at,
			resource_guid,
			propagate_nonnull_agg(resource_name) over prev_events as resource_name,
			resource_type,
			org_guid,
			space_guid,
			plan_guid,
			plan_name,
			service_guid,
			service_name,
			number_of_nodes,
			propagate_nonnull_agg(memory_in_mb) over prev_events as memory_in_mb,
			propagate_nonnull_agg(storage_in_mb) over prev_events as storage_in_mb,
			state
		from
			raw_events
		window
			prev_events as (
				partition by resource_guid, event_type
				order by created_at, event_sequence
				rows between unbounded preceding and current row
			)
		order by
			created_at, event_sequence
	)
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
		);
