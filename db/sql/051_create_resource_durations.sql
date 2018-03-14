DROP MATERIALIZED VIEW IF EXISTS resource_durations;
CREATE MATERIALIZED VIEW IF NOT EXISTS resource_durations AS with

-- Transform deployment name to just a GUID, and human-readable size values in megabytes
compose_events_transformed as (
	select
			id,
			created_at,
			substring(raw_message->'data'->>'deployment' from '[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}$') as guid,
			pg_size_bytes(raw_message->'data'->>'memory') / 1024 / 1024 as memory_in_mb,
			pg_size_bytes(raw_message->'data'->>'storage') / 1024 / 1024 as storage_in_mb,
			raw_message
	from
			compose_audit_events
	order by id
),

-- Enhance Compose event data by adding org, space, and plan data from the 'CREATED' event for that service instance
compose_events_enhanced as (
	select
		cet.id,
		cet.created_at,
		cet.guid,
		sue.raw_message->>'service_instance_name' as name,
		sue.raw_message->>'org_guid' as org_guid,
		sue.raw_message->>'space_guid' as space_guid,
		sue.raw_message->>'service_plan_guid' as plan_guid,
		sue.raw_message->>'service_plan_name' as org_guid,
		'1'::numeric as inst_count,
		cet.memory_in_mb,
		cet.storage_in_mb,
		'STARTED' as state
	from
		compose_events_transformed cet
	left join
		service_usage_events sue
	on
		cet.guid = sue.raw_message->>'service_instance_guid'
	where
		sue.raw_message->>'state' = 'CREATED'
	order by
		cet.id
),

-- Combine Compose events with CloudFoundry service usage events
-- we normalize states to just STARTED/STOPPED because we treat consecutive STARTED to mean "update"
combined_service_events as (
	(
		select * from compose_events_enhanced
	) union all (
		select
			id,
			created_at,
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

-- extract useful stuff from usage events
-- we treat both apps and services as "resources" so normalize the fields
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
		select * from combined_service_events
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
