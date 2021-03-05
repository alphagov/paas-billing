DROP FUNCTION IF EXISTS generate_billable_event_components();

CREATE TABLE billable_event_components_temp (
	event_guid uuid NOT NULL,
	resource_guid uuid NOT NULL,
	resource_name text NOT NULL,
	resource_type text NOT NULL,
	org_guid uuid NOT NULL,
	org_name text NOT NULL,
	space_guid uuid NOT NULL,
	space_name text NOT NULL,
	duration tstzrange NOT NULL,
	plan_guid uuid NOT NULL,
	plan_valid_from timestamptz NOT NULL,
	plan_name text NOT NULL,
	number_of_nodes integer NOT NULL,
	memory_in_mb numeric NOT NULL,
	storage_in_mb numeric NOT NULL,
	component_name text NOT NULL,
	component_formula text NOT NULL,
	currency_code currency_code NOT NULL,
	currency_rate numeric NOT NULL,
	vat_code vat_code NOT NULL,
	vat_rate numeric NOT NULL,
	cost_for_duration numeric NOT NULL,

	PRIMARY KEY (event_guid, plan_guid, duration, component_name),
	CONSTRAINT no_empty_duration CHECK (not isempty(duration))
);

CREATE OR REPLACE FUNCTION generate_billable_event_components()
RETURNS SETOF billable_event_components_temp
LANGUAGE plpgsql AS
$$
BEGIN
	-- Uncomment if needed. These statements have not been uncommented so far in case these table 
	-- names are used in the future elsewhere in the billing code, when tests don't cover the new 
	-- code and this stored function.
	-- DROP TABLE IF EXISTS valid_pricing_plans;
	-- DROP TABLE IF EXISTS valid_currency_rates;
	-- DROP TABLE IF EXISTS valid_vat_rates;
	-- DROP TABLE IF EXISTS events_extract;
	-- DROP TABLE IF EXISTS rds;
	-- DROP TABLE IF EXISTS rds_upgrades;

	CREATE TEMPORARY TABLE valid_pricing_plans ON COMMIT DROP AS
    SELECT
        *,
        tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
            partition by plan_guid order by valid_from rows between current row and 1 following
        )) AS "valid_for"
    FROM
        pricing_plans;

	CREATE INDEX valid_pricing_plans_i1 ON valid_pricing_plans (plan_guid);

	CREATE TEMPORARY TABLE valid_currency_rates ON COMMIT DROP AS
    SELECT
        *,
        tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
            partition by code order by valid_from rows between current row and 1 following
        )) AS "valid_for"
    FROM
        currency_rates;

	CREATE INDEX valid_currency_rates_i1 ON valid_currency_rates (code);

	CREATE TEMPORARY TABLE valid_vat_rates ON COMMIT DROP AS
    SELECT
        *,
        tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
            partition by code order by valid_from rows between current row and 1 following
        )) AS "valid_for"
    FROM
        vat_rates;

	CREATE INDEX valid_vat_rates_i1 ON valid_vat_rates (code);

	-- Select into a events_extract table, holding active resources from billable_event_components
	CREATE TEMPORARY TABLE events_extract ON COMMIT DROP AS
	SELECT
		ev.event_guid,
		ev.resource_guid,
		ev.resource_name,
		ev.resource_type,
		ev.org_guid,
		ev.org_name,
		ev.space_guid,
		ev.space_name,
		ev.duration * vpp.valid_for * vcr.valid_for * vvr.valid_for as duration,
		vpp.plan_guid as plan_guid,
		vpp.valid_from as plan_valid_from,
		vpp.name as plan_name,
		coalesce(ev.number_of_nodes, vpp.number_of_nodes)::integer as number_of_nodes,
		coalesce(ev.memory_in_mb, vpp.memory_in_mb)::numeric as memory_in_mb,
		coalesce(ev.storage_in_mb, vpp.storage_in_mb)::numeric as storage_in_mb,
		ppc.name AS component_name,
		ppc.formula as component_formula,
		vcr.code as currency_code,
		vcr.rate as currency_rate,
		vvr.code as vat_code,
		vvr.rate as vat_rate,
		NULL::DECIMAL as cost_for_duration,
		-- Two separate fields used here, otherwise the table gets very large. Could use UNION ALL instead, which would make later queries more efficient.
		DATE_TRUNC('MONTH', LOWER(duration))::TIMESTAMP AS lower_change_month,
		DATE_TRUNC('MONTH', UPPER(duration))::TIMESTAMP AS upper_change_month
	FROM
		events ev
	LEFT JOIN
		valid_pricing_plans vpp on ev.plan_guid = vpp.plan_guid
		and vpp.valid_for && ev.duration
	LEFT JOIN
		pricing_plan_components ppc on ppc.plan_guid = vpp.plan_guid
		and ppc.valid_from = vpp.valid_from
	LEFT JOIN
		valid_currency_rates vcr on vcr.code = ppc.currency_code
		and vcr.valid_for && (ev.duration * vpp.valid_for)
	LEFT JOIN
		valid_vat_rates vvr on vvr.code = ppc.vat_code
		and vvr.valid_for && (ev.duration * vpp.valid_for * vcr.valid_for);

	CREATE INDEX events_extract_i1 ON events_extract (LOWER(plan_name));
	-- CREATE INDEX events_extract_i2 ON events_extract (lower_change_month);
	-- CREATE INDEX events_extract_i3 ON events_extract (upper_change_month);

	-- Get Postgres RDS instances from event_extract. We need a UNION ALL of events containing the date when event ended and the date
	-- when the same/another event started in the same column so we can aggregate on this date and find Postgres instances that were 
	-- upgraded within the same month.
	CREATE TEMPORARY TABLE rds ON COMMIT DROP AS
	SELECT	org_guid, 
			org_name, 
			space_guid, 
			space_name,
			resource_guid,
			component_name,
			lower_change_month AS change_month
	FROM events_extract
	WHERE LOWER(plan_name) LIKE '%postgres%'
	UNION ALL
	SELECT	org_guid, 
			org_name, 
			space_guid, 
			space_name,
			resource_guid,
			component_name,
			upper_change_month AS change_month
	FROM events_extract
	WHERE LOWER(plan_name) LIKE '%postgres%';

	-- Get RDS resources that have been upgraded within the same month. Look for multiple entries for the same resource_id and month.
	CREATE TEMPORARY TABLE rds_upgrades ON COMMIT DROP AS
	SELECT	org_guid, 
			org_name, 
			space_guid, 
			space_name,
			resource_guid,
			component_name,
			change_month,
			COUNT(*) AS num
	FROM rds
	GROUP BY org_guid, 
			org_name, 
			space_guid, 
			space_name,
			resource_guid,
			component_name,
			change_month
	HAVING COUNT(*) > 1;

	-- Join to events_extract, updating the formula so it does not contain ceil() for the entries where the Postgres RDS has been updated during the month.
	UPDATE events_extract SET component_formula = '($storage_in_mb/1024) * 0.127 * ceil($time_in_seconds)/2678400'
	FROM rds_upgrades u
	WHERE events_extract.component_formula = '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127'
	AND   u.num > 1
	AND   u.org_guid = events_extract.org_guid
	AND   u.org_name = events_extract.org_name
	AND   u.space_guid = events_extract.space_guid
	AND   u.space_name = events_extract.space_name
	AND   u.resource_guid = events_extract.resource_guid
	AND   u.component_name = events_extract.component_name
	AND   ( u.change_month = events_extract.lower_change_month
			OR u.change_month = events_extract.upper_change_month );

	UPDATE events_extract SET component_formula = '($storage_in_mb/1024) * 0.253 * ceil($time_in_seconds)/2678400'
	FROM rds_upgrades u
	WHERE events_extract.component_formula = '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.253'
	AND   u.num > 1
	AND   u.org_guid = events_extract.org_guid
	AND   u.org_name = events_extract.org_name
	AND   u.space_guid = events_extract.space_guid
	AND   u.space_name = events_extract.space_name
	AND   u.resource_guid = events_extract.resource_guid
	AND   u.component_name = events_extract.component_name
	AND   ( u.change_month = events_extract.lower_change_month
			OR u.change_month = events_extract.upper_change_month );

	RETURN QUERY
	SELECT	ev.event_guid,
			ev.resource_guid,
			ev.resource_name,
			ev.resource_type,
			ev.org_guid,
			ev.org_name,
			ev.space_guid,
			ev.space_name,
			/* ev.duration * vpp.valid_for * vcr.valid_for * vvr.valid_for as */ duration,
			/* vpp.plan_guid as */ plan_guid,
			/* vpp.valid_from as */ plan_valid_from,
			/* vpp.name as */ plan_name,
			number_of_nodes AS number_of_nodes,
			memory_in_mb,
			storage_in_mb AS storage_in_mb,
			/* ppc.name AS */ component_name,
			/* ppc.formula as */ component_formula,
			/* vcr.code as */ currency_code,
			/* vcr.rate as */ currency_rate,
			/* vvr.code as */ vat_code,
			/* vvr.rate as */ vat_rate,
			(eval_formula(
				memory_in_mb,
				storage_in_mb,
				number_of_nodes,
				ev.duration,
				component_formula
			) * currency_rate) AS cost_for_duration
	FROM events_extract ev;
END
$$;

INSERT INTO billable_event_components_temp (select * from generate_billable_event_components());

CREATE INDEX billable_event_components_temp_org_idx on billable_event_components_temp (org_guid);
CREATE INDEX billable_event_components_temp_space_idx on billable_event_components_temp (space_guid);
CREATE INDEX billable_event_components_temp_duration_idx on billable_event_components_temp using gist (duration);

DROP TABLE IF EXISTS billable_event_components;
ALTER TABLE billable_event_components_temp RENAME TO billable_event_components;
ALTER INDEX billable_event_components_temp_pkey RENAME TO billable_event_components_pkey;
ALTER INDEX billable_event_components_temp_org_idx RENAME TO billable_event_components_org_idx;
ALTER INDEX billable_event_components_temp_space_idx RENAME TO billable_event_components_space_idx;
ALTER INDEX billable_event_components_temp_duration_idx RENAME TO billable_event_components_duration_idx;
