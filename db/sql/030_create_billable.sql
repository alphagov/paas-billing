CREATE TYPE billable_events_dataset AS(
	id integer,
	guid text,
	name text,
	org_guid text,
	space_guid text,
	plan_id text,
	memory_in_mb numeric,
	duration tstzrange
);



DROP MATERIALIZED VIEW IF EXISTS billable;
CREATE MATERIALIZED VIEW IF NOT EXISTS billable AS with

-- extract useful stuff from usage events
-- we treat both apps and services as billable "resources" so normalize the fields
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
			'0'::numeric as memory_in_mb,
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
		tstzrange(created_at, lead(created_at, 1, now()) over resource_states) as duration,
		state
	from
		events
	window
		resource_states as (partition by guid order by id rows between current row and 1 following)
	order by
		guid, id
),

-- generate rows for every "instance" of an app
-- this results in a row per-instance
resources as (
	select
		t.*,
		generate_series(1, t.inst_count)
	from
		event_ranges t
),

-- join the pricing plans
-- this results in ONLY the resources that can be billed for being listed
select
	r.id,
	r.guid,
	r.name,
	r.org_guid,
	r.space_guid,
	r.plan_guid,
	r.memory_in_mb,
	(
		greatest(lower(pp.valid_for),lower(r.duration)),
		least(upper(pp.valid_for),upper(r.duration))
	) as duration,
from
	resources r
where
	r.state = 'STARTED'
order by
	r.id
;

CREATE INDEX IF NOT EXISTS idx_id ON billable (id);
CREATE INDEX IF NOT EXISTS idx_org ON billable (org_guid);
CREATE INDEX IF NOT EXISTS idx_space ON billable (space_guid);
CREATE INDEX IF NOT EXISTS idx_pricing_plan ON billable (pricing_plan_id);
CREATE INDEX IF NOT EXISTS idx_duration ON billable USING gist (duration);
