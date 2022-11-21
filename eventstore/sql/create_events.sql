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
	event_ranges as (
			(SELECT * FROM app_event_ranges)
		union all
			(SELECT * FROM task_event_ranges)
		union all
			(SELECT * FROM staging_event_ranges)
		union all
			(SELECT * FROM service_event_ranges)
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
