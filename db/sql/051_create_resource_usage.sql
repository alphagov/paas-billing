DROP MATERIALIZED VIEW IF EXISTS resource_usage;

-- fix timestamp fields to ensure they use UTC
alter table app_usage_events alter column created_at type timestamp with time zone USING created_at at time zone 'UTC';
alter table service_usage_events alter column created_at type timestamp with time zone USING created_at at time zone 'UTC';

CREATE MATERIALIZED VIEW IF NOT EXISTS resource_usage AS with

-- extract useful stuff from usage events
-- we treat both apps and services as "resources" so normalize the fields
-- we normalize states to just STARTED/STOPPED because we treat consecutive STARTED to mean "update"
events as (
	(
		select
			id,
			guid as event_guid,
			created_at,
			(raw_message->>'app_guid') as resource_guid,
			(raw_message->>'app_name') as resource_name,
			'app'::text as resource_type,                              -- resource_type for compute resources
			(raw_message->>'org_guid') as org_guid,
			(raw_message->>'space_guid') as space_guid,
			'f4d4b95a-f55e-4593-8d54-3364c25798c4'::text as plan_guid, -- plan guid for all compute resources
			'default'::text as plan_name,                              -- plan name for all compute resources
			coalesce(raw_message->>'instance_count', '1')::numeric as number_of_nodes,
			coalesce(raw_message->>'memory_in_mb_per_instance', '0')::numeric as memory_in_mb,
			'0'::numeric as storage_in_mb,
			raw_message->>'state' as state
		from
			app_usage_events
		where
			raw_message->>'state' = 'STARTED'
			or raw_message->>'state' = 'STOPPED'
	) union all (
		select
			id,
			guid as event_guid,
			created_at,
			(raw_message->>'service_instance_guid') as resource_guid,
			(raw_message->>'service_instance_name') as resource_name,
			(raw_message->>'service_plan_label') as resource_type,
			(raw_message->>'org_guid') as org_guid,
			(raw_message->>'space_guid') as space_guid,
			(raw_message->>'service_plan_guid') as plan_guid,
			(raw_message->>'service_plan_name') as plan_name,
			NULL::numeric as number_of_nodes,
			NULL::numeric as memory_in_mb,
			NULL::numeric as storage_in_mb,
			case
				when (raw_message->>'state') = 'CREATED' then 'STARTED'
				when (raw_message->>'state') = 'DELETED' then 'STOPPED'
				when (raw_message->>'state') = 'UPDATED' then 'STARTED'
			end as state
		from
			service_usage_events
		where
			raw_message->>'service_instance_type' = 'managed_service_instance'
	)
),

-- combine and convert to rows for each STARTED-STARTED or STARTED-STOPPED pair with duration range
-- if a resources does not have a STOPPED event yet, then the resource's duration range will end at the time the view is refreshed
event_ranges as (
	select
		*,
		tstzrange(created_at, lead(created_at, 1, now()) over resource_states) as duration
	from
		events
	window
		resource_states as (partition by resource_guid order by id rows between current row and 1 following)
	order by
		id
)

-- keep only the STARTED rows
-- discard empty durations due to low precision of event timestamps (1s)
select
	event_guid,
	resource_guid,
	resource_name,
	resource_type,
	duration,
	org_guid,
	space_guid,
	plan_guid,
	plan_name,
	number_of_nodes,
	memory_in_mb,
	storage_in_mb
from
	event_ranges
where
	state = 'STARTED' and not isempty(duration)
;

CREATE UNIQUE INDEX resource_usage_uniq ON resource_usage (event_guid);
CREATE INDEX resource_usage_org_idx ON resource_usage (org_guid);
CREATE INDEX resource_usage_space_idx ON resource_usage (space_guid);
CREATE INDEX resource_usage_resource_idx ON resource_usage (resource_guid);
CREATE INDEX resouce_usage_duration_idx ON resource_usage using gist (duration);
CREATE INDEX resource_usage_plan_idx ON resource_usage (plan_guid);
