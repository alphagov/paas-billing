CREATE OR REPLACE FUNCTION compile_formula( metadata JSONB, duration tstzrange, formula text ) RETURNS text AS $$
DECLARE
  out text;
BEGIN
  out := coalesce(lower(formula), '0');
  out := (select 'select (' || out || ')::numeric;');
  return out;
END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION eval_formula(
  metadata JSONB,
  duration tstzrange,
  formula text
) returns numeric AS $$
DECLARE
  out numeric;
BEGIN
  return 7;
END; $$ LANGUAGE plpgsql IMMUTABLE;
