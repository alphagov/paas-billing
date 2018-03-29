
CREATE OR REPLACE FUNCTION to_seconds(tstzrange) RETURNS numeric AS $$
	select extract(epoch from (upper($1) - lower($1)))::numeric;
$$ LANGUAGE SQL IMMUTABLE STRICT;


CREATE OR REPLACE FUNCTION iso8601(t timestamptz) returns text AS $$ BEGIN
	RETURN to_char(t at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
END; $$ LANGUAGE plpgsql;

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
