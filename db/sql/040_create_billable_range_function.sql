-- reset
DROP FUNCTION IF EXISTS billable_range(tstzrange);

-- billable_range queries the billable view for a given range
--
-- example:
--
--    select * from billable_range(tstzrange('2017-01-01', '2017-01-05'));
--

CREATE OR REPLACE FUNCTION billable_range(selected_range tstzrange) RETURNS TABLE(
	id int,
	guid text,
	name text,
	org_guid text,
	space_guid text,
	memory_in_mb numeric,
	duration tstzrange,
	pricing_plan_id int,
	pricing_plan_name text,
	formula text,
	price numeric
) AS $$ BEGIN
	RETURN QUERY WITH
	valid_pricing_plans as (
		select
			pp.*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over plans) as valid_for
		from
			pricing_plans pp
		window
			plans as (partition by plan_guid order by valid_from rows between current row and 1 following)
	)
	select
		b.id,
		b.guid,
		b.name,
		b.org_guid,
		b.space_guid,
		b.memory_in_mb,
		tstzrange(
			greatest(lower(selected_range), lower(vpp.valid_for), lower(b.duration)),
			least(upper(selected_range), upper(vpp.valid_for), upper(b.duration))
		) as duration,
		vpp.id AS pricing_plan_id,
		vpp.name AS pricing_plan_name,
		vpp.formula,
		eval_formula(
			b.memory_in_mb,
			tstzrange(
				greatest(lower(selected_range), lower(vpp.valid_for), lower(b.duration)),
				least(upper(selected_range), upper(vpp.valid_for), upper(b.duration))
			),
			vpp.formula
		) as price
	from
		billable b
		inner join
		valid_pricing_plans vpp 
      on b.plan_guid = vpp.plan_guid
      and vpp.valid_for && b.duration 
      and vpp.valid_for && selected_range
	where
		b.duration && selected_range;
END; $$ LANGUAGE plpgsql;
