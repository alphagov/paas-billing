DROP MATERIALIZED VIEW IF EXISTS billable;           -- FIXME legacy: remove me later
DROP TABLE IF EXISTS billable_event_components;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS pricing_plan_components;
DROP TABLE IF EXISTS pricing_plans;
DROP TABLE IF EXISTS vat_rates;
DROP TABLE IF EXISTS currency_rates;
DROP TYPE IF EXISTS resource_state;
DROP TYPE IF EXISTS currency_code;
DROP TYPE IF EXISTS vat_code;
