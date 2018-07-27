
CREATE TABLE billable_event_components (
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

	PRIMARY KEY (event_guid, plan_guid, duration, component_name),
	CONSTRAINT no_empty_duration CHECK (not isempty(duration))
);

CREATE OR REPLACE FUNCTION generate_billable_event_components() RETURNS SETOF billable_event_components AS $$
	with	
	valid_pricing_plans as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by plan_guid order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			pricing_plans
	),
	valid_currency_rates as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by code order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			currency_rates
	),
	valid_vat_rates as (
		select
			*,
			tstzrange(valid_from, lead(valid_from, 1, 'infinity') over (
				partition by code order by valid_from rows between current row and 1 following
			)) as valid_for
		from
			vat_rates
	)
	select
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
		vvr.rate as vat_rate
	from
		events ev
	left join
		valid_pricing_plans vpp on ev.plan_guid = vpp.plan_guid
		and vpp.valid_for && ev.duration
	left join
		pricing_plan_components ppc on ppc.plan_guid = vpp.plan_guid
		and ppc.valid_from = vpp.valid_from
	left join
		valid_currency_rates vcr on vcr.code = ppc.currency_code
		and vcr.valid_for && (ev.duration * vpp.valid_for)
	left join
		valid_vat_rates vvr on vvr.code = ppc.vat_code
		and vvr.valid_for && (ev.duration * vpp.valid_for * vcr.valid_for)
; $$ LANGUAGE SQL;

INSERT INTO billable_event_components (select * from generate_billable_event_components());

CREATE INDEX billable_event_components_org_idx on billable_event_components (org_guid);
CREATE INDEX billable_event_components_space_idx on billable_event_components (space_guid);
CREATE INDEX billable_event_components_duration_idx on billable_event_components using gist (duration);

