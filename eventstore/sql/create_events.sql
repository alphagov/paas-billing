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

-- FIXME: should be dropped once we'd upgrade postgres version.
CREATE FUNCTION pg_size_bytes (size text)
	RETURNS bigint
AS $$
  if (size[-1] == 'B'):
    size = size[:-1]
  value = size[:-1].strip()
  unit = size[-1]
  if (value.isdigit()):
    b = int(value)
    if (unit == 'G'):
      b *= 1073741824
    elif (unit == 'M'):
      b *= 1048576
    elif (unit == 'K'):
      b *= 1024
  else:
    b = 0
  return b
$$ LANGUAGE plpythonu;

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
				created_at::timestamptz as created_at,
				substring(
					raw_message->'data'->>'deployment' 
					from '[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}$'
				)::uuid as resource_guid,
				(sue.raw_message->>'service_instance_name') as resource_name,
				'compose'::text as resource_type,
				(sue.raw_message->>'org_guid')::uuid as org_guid,
				(sue.raw_message->>'space_guid')::uuid as space_guid,
				'8d3383cf-9477-46cc-a219-ec0c23c020dd'::uuid as plan_guid,
				'compose'::text as plan_name,
				'1'::numeric as number_of_nodes,
				(pg_size_bytes(raw_message->'data'->>'memory') / 1024 / 1024)::numeric as memory_in_mb,
				(pg_size_bytes(raw_message->'data'->>'storage') / 1024 / 1024)::numeric as storage_in_mb,
				(case
					when (sue.raw_message->>'state') = 'CREATED' then 'STARTED'
					when (sue.raw_message->>'state') = 'DELETED' then 'STOPPED'
				end)::resource_state as state
			from
				compose_audit_events
			left join
				service_usage_events sue
			on
				resource_guid = sue.raw_message->>'service_instance_guid'
			where
				sue.raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-' -- FIXME: this is open to abuse
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
