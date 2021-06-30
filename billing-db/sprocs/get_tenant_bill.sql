-- These temporary tables need to be created when creating the database connection.

-- What needs to be billed. This can be used for any resources, past or future, so can be used by the billing calculator.
CREATE TEMPORARY TABLE billable_resources
(
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL,
    resource_guid UUID NULL,
    resource_type TEXT NULL,
    resource_name TEXT NULL,
    org_guid UUID NULL,
    org_name TEXT NULL,
    space_guid UUID NULL,
    space_name TEXT NULL,
    plan_name TEXT NULL,
    plan_guid UUID NULL,
    storage_in_mb NUMERIC NULL,
    memory_in_mb NUMERIC NULL,
    number_of_nodes INT NULL
);

-- The billable_by_component table needs creating before running this stored function. This is so we can preserve the contents of this table for audit/debug purposes.
CREATE TEMPORARY TABLE billable_by_component
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
    space_guid UUID NULL,
    space_name TEXT NULL,
    plan_name TEXT NULL,
    plan_guid UUID NULL,
    component_name TEXT NULL,

    storage_in_mb NUMERIC NULL,
    memory_in_mb NUMERIC NULL,
    number_of_nodes INT NULL,
    aws_price DECIMAL NULL,
    generic_formula TEXT NULL,
    vat_code VARCHAR NULL,
    currency_code CHAR(3) NULL, -- ISO currency code. Original currency code
    time_in_seconds INT NULL,
    charge_usd_exc_vat DECIMAL NULL,
    charge_gbp_exc_vat DECIMAL NULL,
    charge_gbp_inc_vat DECIMAL NULL,
    is_processed BOOLEAN NULL
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS billable_by_component_i1 ON billable_by_component (generic_formula, storage_in_mb, memory_in_mb, number_of_nodes, aws_price);
CREATE INDEX CONCURRENTLY IF NOT EXISTS billable_by_component_i2 ON billable_by_component (generic_formula);

-- Calculate bill for a given month, or any date/time range, for a tenant.
CREATE OR REPLACE FUNCTION get_tenant_bill
(
    _org_name TEXT,
    _from_date TIMESTAMP,
    _to_date TIMESTAMP
)
RETURNS TABLE
(
    org_name TEXT,
    org_guid UUID,
    plan_name TEXT,
    space_name TEXT,
    resource_name TEXT,
    charge_usd_exc_vat DECIMAL,
    charge_gbp_exc_vat DECIMAL,
    charge_gbp_inc_vat DECIMAL
)
LANGUAGE plpgsql AS $$
BEGIN
    TRUNCATE TABLE billable_resources;

    INSERT INTO billable_resources
    (
        valid_from,
        valid_to,
        resource_guid,
        resource_type,
        resource_name,
        org_guid,
        org_name,
        space_guid,
        space_name,
        plan_name,
        plan_guid,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes
    )
    -- _from_date, _to_date:                  |---------------------------|
    -- Resource present:            |-----------------|
    SELECT  GREATEST(_from_date, r.valid_from),
            LEAST(_to_date, r.valid_to),
            r.resource_guid,
            r.resource_type,
            r.resource_name,
            r.org_guid,
            r.org_name,
            r.space_guid,
            r.space_name,
            r.plan_name,
            r.plan_guid,
            r.storage_in_mb,
            r.memory_in_mb,
            r.number_of_nodes
    FROM  resources r
    WHERE r.org_name = _org_name
    AND   r.valid_from < _from_date
    AND   r.valid_to > _from_date
    AND   r.valid_to < _to_date
    UNION ALL
    -- _from_date, _to_date:                  |---------------------------|
    -- Resource present:                         |-----------------|
    -- Resource present:                      |---------------------------|
    SELECT  r.valid_from,
            r.valid_to,
            r.resource_guid,
            r.resource_type,
            r.resource_name,
            r.org_guid,
            r.org_name,
            r.space_guid,
            r.space_name,
            r.plan_name,
            r.plan_guid,
            r.storage_in_mb,
            r.memory_in_mb,
            r.number_of_nodes
    FROM  resources r
    WHERE r.org_name = _org_name
    AND   r.valid_from >= _from_date
    AND   r.valid_from < _to_date
    AND   r.valid_to > _from_date
    AND   r.valid_to <= _to_date
    UNION ALL
    -- _from_date, _to_date:                  |---------------------------|
    -- Resource present:                                       |-----------------|
    SELECT  GREATEST(_from_date, r.valid_from),
            LEAST(_to_date, r.valid_to),
            r.resource_guid,
            r.resource_type,
            r.resource_name,
            r.org_guid,
            r.org_name,
            r.space_guid,
            r.space_name,
            r.plan_name,
            r.plan_guid,
            r.storage_in_mb,
            r.memory_in_mb,
            r.number_of_nodes
    FROM  resources r
    WHERE r.org_name = _org_name
    AND   r.valid_from > _from_date
    AND   r.valid_from < _to_date
    AND   r.valid_to > _to_date
    UNION ALL
    -- _from_date, _to_date:                  |---------------------------|
    -- Resource present:            |---------------------------------------------|
    SELECT  _from_date,
            _to_date,
            r.resource_guid,
            r.resource_type,
            r.resource_name,
            r.org_guid,
            r.org_name,
            r.space_guid,
            r.space_name,
            r.plan_name,
            r.plan_guid,
            r.storage_in_mb,
            r.memory_in_mb,
            r.number_of_nodes
    FROM  resources r
    WHERE r.org_name = _org_name
    AND   r.valid_from < _from_date
    AND   r.valid_to > _to_date;

    RETURN QUERY
    SELECT * FROM calculate_bill();
END
$$;
