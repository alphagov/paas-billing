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

DROP MATERIALIZED VIEW IF EXISTS billable2;
CREATE MATERIALIZED VIEW IF NOT EXISTS billable2 AS
select
	id,
	guid,
	name,
	org_guid,
	space_guid,
	pricing_plan_id as plan_id,
	memory_in_mb,
	duration,
from billable;

