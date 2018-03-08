DROP MATERIALIZED VIEW IF EXISTS resource_durations;
CREATE MATERIALIZED VIEW IF NOT EXISTS resource_durations AS with

-- extract useful stuff from usage events
-- we treat both apps and services as "resources" so normalize the fields
-- we normalize states to just STARTED/STOPPED because we treat consecutive STARTED to mean "update"
events as (
	(
		select
			id,
			created_at::timestamptz as created_at,
			(raw_message->>'app_guid') as guid,
			(raw_message->>'app_name') as name,
			(raw_message->>'org_guid') as org_guid,
			(raw_message->>'space_guid') as space_guid,
			'f4d4b95a-f55e-4593-8d54-3364c25798c4'::text as plan_guid, -- fake plan id for compute plans
			'default-compute'::text as plan_name,                      -- fake plan name for compute plans
			coalesce(raw_message->>'instance_count', '0')::numeric as inst_count,
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
			created_at::timestamptz as created_at,
			(raw_message->>'service_instance_guid') as guid,
			(raw_message->>'service_instance_name') as name,
			(raw_message->>'org_guid') as org_guid,
			(raw_message->>'space_guid') as space_guid,
			(raw_message->>'service_plan_guid') as plan_guid,
			(raw_message->>'service_plan_name') as plan_name,
			'1'::numeric as inst_count,
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
		id,
		guid,
		org_guid,
		space_guid,
		name,
		plan_guid,
		plan_name,
		inst_count,
		memory_in_mb,
		storage_in_mb,
		tstzrange(created_at, lead(created_at, 1, now()) over resource_states) as duration,
		state
	from
		events
	window
		resource_states as (partition by guid order by id rows between current row and 1 following)
	order by
		guid, id
)

-- generate rows for every "instance" of an app
-- this results in a row per-instance
select
	t.id || '-' || t.guid || '-' || generate_series(1, t.inst_count) AS id,
	t.guid,
	t.name,
	t.org_guid,
	t.space_guid,
	t.plan_guid,
	t.memory_in_mb,
	t.storage_in_mb,
	t.duration,
	t.state
from
	event_ranges t
where
	t.state = 'STARTED'
order by
	t.id
;

CREATE UNIQUE INDEX IF NOT EXISTS idx_id ON resource_durations (id);
CREATE INDEX IF NOT EXISTS idx_org ON resource_durations (org_guid);
CREATE INDEX IF NOT EXISTS idx_space ON resource_durations (space_guid);
CREATE INDEX IF NOT EXISTS idx_duration ON resource_durations USING gist (duration);
CREATE INDEX IF NOT EXISTS idx_plan ON resource_durations (plan_guid);
