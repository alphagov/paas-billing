-- exec a formula with variables substituted
CREATE OR REPLACE FUNCTION eval_formula(
	memory_in_mb numeric,
	storage_in_mb numeric,
	number_of_nodes integer,
	duration tstzrange,
	formula text
) returns numeric AS $$
DECLARE
	out numeric;
BEGIN
	execute compile_formula(formula) into out using
		coalesce(memory_in_mb, 0),
		coalesce(storage_in_mb, 0),
		coalesce(number_of_nodes, 0),
		extract(epoch from (upper(duration) - lower(duration)));
	return out;
END; $$ LANGUAGE plpgsql IMMUTABLE;



-- compile formula into sql
CREATE OR REPLACE FUNCTION compile_formula( formula text ) RETURNS text AS $$
DECLARE
	out text;
BEGIN
	out := coalesce(lower(formula), '0');
	out := regexp_replace(out, '\$memory_in_mb', '($1::numeric)', 'g');
	out := regexp_replace(out, '\$storage_in_mb', '($2::numeric)', 'g');
	out := regexp_replace(out, '\$number_of_nodes', '($3::numeric)', 'g');
	out := regexp_replace(out, '\$time_in_seconds', '($4::numeric)', 'g');
	out := (select 'select (' || out || ')::numeric;');
	return out;
END; $$ LANGUAGE plpgsql;
