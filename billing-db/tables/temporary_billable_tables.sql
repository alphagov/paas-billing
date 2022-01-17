-- These temporary tables need to be created when creating the database connection.

-- What needs to be billed. This can be used for any resources, past or future, so can be used by the billing calculator.
CREATE TEMPORARY TABLE IF NOT EXISTS billable_resources
(
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL,
    resource_guid UUID NULL,
    resource_type TEXT NULL,
    resource_name TEXT NULL,
    org_guid UUID NULL,
    org_name TEXT NULL,
    org_quota_definition_guid UUID NULL,
    space_guid UUID NULL,
    space_name TEXT NULL,
    plan_name TEXT NULL,
    plan_guid UUID NULL,
    storage_in_mb NUMERIC NULL,
    memory_in_mb NUMERIC NULL,
    number_of_nodes INT NULL
);

-- The billable_by_component table needs creating before running this stored function. This is so we can preserve the contents of this table for audit/debug purposes.
CREATE TEMPORARY TABLE IF NOT EXISTS billable_by_component
(
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL,
    -- valid_from_month - useful if we're calculating bills for more than one month
    -- valid_to_month - useful if we're calculating bills for more than one month
    resource_guid UUID NULL,
    resource_type TEXT NULL,
    resource_name TEXT NULL,
    org_guid UUID NULL,
    org_name TEXT NULL,
    org_quota_definition_guid UUID NULL,
    space_guid UUID NULL,
    space_name TEXT NULL,
    plan_name TEXT NULL,
    plan_guid UUID NULL,
    component_name TEXT NULL,

    storage_in_mb NUMERIC NULL,
    memory_in_mb NUMERIC NULL,
    number_of_nodes INT NULL,
    external_price DECIMAL NULL,
    generic_formula TEXT NULL,
    vat_code VARCHAR NULL,
    currency_code CHAR(3) NULL, -- ISO currency code. Original currency code
    time_in_seconds INT NULL,
    charge_usd_exc_vat DECIMAL NULL,
    charge_gbp_exc_vat DECIMAL NULL,
    charge_gbp_inc_vat DECIMAL NULL,
    is_processed BOOLEAN NULL
);

CREATE INDEX IF NOT EXISTS billable_by_component_i1 ON billable_by_component (generic_formula, storage_in_mb, memory_in_mb, number_of_nodes, external_price);
CREATE INDEX IF NOT EXISTS billable_by_component_i2 ON billable_by_component (generic_formula);
