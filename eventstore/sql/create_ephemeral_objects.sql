CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION to_seconds(tstzrange) RETURNS numeric AS $$
	select extract(epoch from (upper($1) - lower($1)))::numeric;
$$ LANGUAGE SQL IMMUTABLE STRICT;


CREATE OR REPLACE FUNCTION iso8601(t timestamptz) returns text AS $$ BEGIN
	RETURN to_char(t at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
END; $$ LANGUAGE plpgsql;

-- FIXME: should be dropped once we'd upgrade postgres version.
CREATE OR REPLACE FUNCTION pg_size_bytes(input text) returns bigint as $$ declare
	size text;
	unit text := 'bytes';
	value numeric := 0;
	n bigint;
begin
	size = trim(regexp_replace(input, E'\\s+', ' ', 'g'));
	unit = lower(regexp_replace(size, E'[\\-\\.0-9]', '', 'g'));
	value = array_to_string(regexp_matches(size, E'^([0-9e\\-\\.]+)', 'i'), '')::numeric;
	if value is null then
		raise exception 'invalid size "%"', input;
	end if;
	if unit ~* 'bytes$' or unit = '' then
		value = value;
	elsif unit ~* 'kb$' then
		value = value * 1024;
	elsif unit ~* 'mb$' then
		value = value * 1024 * 1024;
	elsif unit ~* 'gb$' then
		value = value * 1024 * 1024 * 1024;
	elsif unit ~* 'tb$' then
		value = value * 1024 * 1024 * 1024 * 1024;
	else
		n = value::bigint;
		value = null;
	end if;
	if value is null then
		raise exception 'invalid size "%"', input using
			hint = 'Valid units are "bytes", "kB", "MB", "GB", and "TB"',
			detail = 'Invalid size unit: "' || unit || '"';
	end if;
	return value::bigint;
END; $$ LANGUAGE plpgsql;
-- FIXME: END


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

-- validate formula whitelist's selected terms that can be evaluated in
-- "formulas" this should not be considered "safe" for untrusted input it is
-- intended as a technique to restrict formulas from becoming complex SQL
-- expressions
CREATE OR REPLACE FUNCTION validate_formula() RETURNS trigger AS $$
DECLARE
	invalid_formula text;
	illegal_token text;
	dummy_price numeric;
BEGIN
	IF (NEW.formula = '') THEN
		RAISE EXCEPTION 'formula can not be empty';
	END IF;
	invalid_formula := lower(NEW.formula);
	invalid_formula := (select regexp_replace(invalid_formula, '::(integer|bigint|numeric)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '([0-9]+)?\.([0-9]+)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '([0-9]+)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\$memory_in_mb', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\$storage_in_mb', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\$time_in_seconds', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\$number_of_nodes', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, 'ceil', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\(|\)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\*', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\-', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\+', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\/', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\^', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\s+', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '#+', '', 'g'));
	IF (invalid_formula != '') THEN
		illegal_token := (select * from regexp_split_to_table(invalid_formula, '\s+') limit 1);
		RAISE EXCEPTION 'illegal token in formula: %', illegal_token;
	END IF;
	-- attempt to use the formula to ensure it works with common edge case inputs
	dummy_price := (select eval_formula(0, 0, 0, tstzrange(now(), now()), NEW.formula));
	dummy_price := (select eval_formula(1, 1, 1, tstzrange(now(), now() + '1 second'), NEW.formula));
	dummy_price := (select eval_formula(null, null, null, null, NEW.formula));
	RETURN NEW;
END;
$$ language plpgsql;


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

-------------------------------------- SCHEMA

CREATE TYPE vat_code AS ENUM ('Standard', 'Reduced', 'Zero');
CREATE TYPE currency_code AS ENUM ('USD', 'GBP', 'EUR');

CREATE TABLE pricing_plans (
	plan_guid uuid NOT NULL,
	valid_from timestamptz NOT NULL,
	name text NOT NULL,
	memory_in_mb integer NOT NULL DEFAULT 0,
	number_of_nodes integer NOT NULL DEFAULT 0,
	storage_in_mb integer NOT NULL DEFAULT 0,

	PRIMARY KEY (plan_guid, valid_from),
	CONSTRAINT name_must_not_be_blank CHECK (length(trim(name)) > 0),
	CONSTRAINT valid_from_start_of_month CHECK (
	  (extract (day from valid_from)) = 1 AND
	  (extract (hour from valid_from)) = 0 AND
	  (extract (minute from valid_from)) = 0 AND
	  (extract (second from valid_from)) = 0
	)
);


CREATE TABLE IF NOT EXISTS currency_rates(
	code currency_code NOT NULL,
	valid_from timestamptz NOT NULL,
	rate numeric NOT NULL,

	PRIMARY KEY (code, valid_from),
	CONSTRAINT rate_must_be_greater_than_zero CHECK (rate > 0),
	CONSTRAINT valid_from_start_of_month CHECK (
	  (extract (day from valid_from)) = 1 AND
	  (extract (hour from valid_from)) = 0 AND
	  (extract (minute from valid_from)) = 0 AND
	  (extract (second from valid_from)) = 0
	)
);

CREATE TABLE IF NOT EXISTS vat_rates (
	code vat_code NOT NULL,
	valid_from timestamptz NOT NULL,
	rate numeric NOT NULL,

	PRIMARY KEY (code, valid_from),
	CONSTRAINT rate_must_be_greater_than_zero CHECK (rate >= 0),
	CONSTRAINT valid_from_start_of_month CHECK (
	  (extract (day from valid_from)) = 1 AND
	  (extract (hour from valid_from)) = 0 AND
	  (extract (minute from valid_from)) = 0 AND
	  (extract (second from valid_from)) = 0
	)
);

CREATE TABLE pricing_plan_components (
	plan_guid uuid NOT NULL,
	valid_from timestamptz NOT NULL,
	name text NOT NULL,
	formula text NOT NULL,
	vat_code vat_code NOT NULL, 
	currency_code currency_code NOT NULL,

	PRIMARY KEY (plan_guid, valid_from, name),
	FOREIGN KEY (plan_guid, valid_from) REFERENCES pricing_plans (plan_guid, valid_from) ON DELETE CASCADE, 
	CONSTRAINT name_must_not_be_blank CHECK (length(trim(name)) > 0),
	CONSTRAINT formula_must_not_be_blank CHECK (length(trim(formula)) > 0)
);
CREATE TRIGGER tgr_ppc_validate_formula BEFORE INSERT OR UPDATE ON pricing_plan_components FOR EACH ROW EXECUTE PROCEDURE validate_formula();


CREATE TYPE resource_state AS ENUM ('STARTED', 'STOPPED');

