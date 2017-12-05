-- exec a formula with variables substituted
CREATE OR REPLACE FUNCTION eval_formula(
	mb numeric,
	duration tstzrange,
	formula text
) returns numeric AS $$
DECLARE
	out numeric;
BEGIN
	execute compile_formula(formula) into out using
		coalesce(mb, 0),
		extract(epoch from (upper(duration) - lower(duration)));
	return out;
END; $$ LANGUAGE plpgsql IMMUTABLE;



-- compile formula into sql
CREATE OR REPLACE FUNCTION compile_formula( formula text ) RETURNS text AS $$
DECLARE
	out text;
BEGIN
	out := (select
		'select ('
			|| regexp_replace(regexp_replace(coalesce(lower(formula), '0'),
				'\$memory_in_mb',     '($1::numeric)', 'g'),
				'\$time_in_seconds',  '($2::numeric)', 'g')
			|| ')::numeric;'
	);
	return out;
END; $$ LANGUAGE plpgsql;
