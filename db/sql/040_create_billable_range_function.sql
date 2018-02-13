-- reset
DROP FUNCTION IF EXISTS billable_range(tstzrange); -- FIXME: Delete later
DROP FUNCTION IF EXISTS billable_range2(regclass, tstzrange);

-- DROP TYPE IF EXISTS billable_events_dataset;
-- CREATE TYPE billable_events_dataset AS(
--	id integer,
--	guid text,
--	name text,
--	org_guid text,
--	space_guid text,
--	plan_guid text,
--	memory_in_mb numeric,
--	duration tstzrange
--);

-- billable_range queries the billable view for a given range
--
-- example:
--
--    select * from billable_range(tstzrange('2017-01-01', '2017-01-05'));
--
CREATE OR REPLACE FUNCTION billable_range2(
	billable_table regclass,
	selected_range tstzrange)
RETURNS TABLE(
       id int,
       guid text,
       name text,
       org_guid text,
       space_guid text,
       memory_in_mb numeric,
       duration tstzrange,
       pricing_plan_guid text,
       pricing_plan_id int,
       pricing_plan_name text,
       formula text,
       price numeric
) AS $$ BEGIN
	RETURN query
    -- EXECUTE
	-- format('
    WITH
	valid_pricing_plans AS (
		select
		pp.*,
		tstzrange(valid_from, lead(valid_from, 1, 'infinity') over plans) as valid_for
		from
 		    pricing_plans pp
		window
		    plans as (partition by plan_guid order by valid_from rows between current row and 1 following)
	),
	billable_table2 AS (
		EXECUTE IMMEDIATE 'select * from ' || billable_table
	),
    SELECT
		b.id,
		b.guid,
		b.name,
		b.org_guid,
		b.space_guid,
		b.memory_in_mb,
		tstzrange(
			greatest(lower(selected_range), lower(b.duration)),
			least(upper(selected_range), upper(b.duration))
		) as duration,
		pp.plan_guid as pricing_plan_guid,
		pp.id as pricing_plan_id,
		pp.name as pricing_plan_name,
		pp.formula,
		eval_formula(
			b.memory_in_mb,
			tstzrange(
				greatest(lower(selected_range), lower(b.duration)),
				least(upper(selected_range), upper(b.duration))
			),
			pp.formula
		) as price
    from
		-- %s ba
		billable_table2 b
    inner join
		valid_pricing_plans pp on b.plan_guid = pp.plan_guid and pp.valid_for && b.duration
    where
		b.duration && selected_range
    --', billable_table)
;
END; $$ LANGUAGE plpgsql;


