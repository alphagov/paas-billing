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
	return query select
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
		b.pricing_plan_id,
		b.pricing_plan_name,
		b.formula,
		eval_formula(
			b.memory_in_mb,
			tstzrange(
				greatest(lower(selected_range), lower(b.duration)),
				least(upper(selected_range), upper(b.duration))
			),
			b.formula
		) as price
	from
		billable b
	where
		b.duration && selected_range;
END; $$ LANGUAGE plpgsql;
