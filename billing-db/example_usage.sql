```
-- Method 1 - calling stored function to get bill

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
CREATE INDEX CONCURRENTLY IF NOT EXISTS billable_by_component_i1 ON billable_by_component (generic_formula, storage_in_mb, memory_in_mb, number_of_nodes, external_price);
CREATE INDEX CONCURRENTLY IF NOT EXISTS billable_by_component_i2 ON billable_by_component (generic_formula);

-- Get bill for the month
SELECT * FROM get_tenant_bill('govuk-pay', '2020-12-01', '2021-01-01');

-- Get bill for the whole year
SELECT * FROM get_tenant_bill('govuk-pay', '2020-12-01', '2021-01-01');

-- Random time interval
SELECT * FROM get_tenant_bill('govuk-pay', '2020-12-02', '2021-01-15');

-- You can easily create a sibling stored function alongside get_tenant_bill

-- Method 1 - populating the resources whose bill is to be calculated manually

-- Example use: billing calculator

-- Monthly cost
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
SELECT  NOW(),
        NOW() + INTERVAL '1 month',
        md5(random()::text || clock_timestamp()::text)::UUID, -- resource_guid
        'service', -- resource_type
        'test-db',
        md5(random()::text || clock_timestamp()::text)::UUID, -- org_guid
        'test-org',
        md5(random()::text || clock_timestamp()::text)::UUID, -- space_guid
        'test-space',
        'postgres large-10 high-iops',
        '804512b7-c949-4d46-82da-ee6fc3c1cb51',
        2621440, -- storage_in_mb
        0, -- memory_in_mb
        0; -- number_of_nodes

-- Note, if you're adding multiple entries into the above, they need to have the same org_guid and space_guid.

SELECT * FROM calculate_bill();

-- Annual cost

TRUNCATE TABLE billable_resources;

-- Note, if you're adding multiple entries into the above, they need to have the same org_guid and space_guid.

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
SELECT  NOW(),
        NOW() + INTERVAL '1 year',
        md5(random()::text || clock_timestamp()::text)::UUID, -- resource_guid
        'service', -- resource_type
        'test-db',
        md5(random()::text || clock_timestamp()::text)::UUID, -- org_guid
        'test-org',
        md5(random()::text || clock_timestamp()::text)::UUID, -- space_guid
        'test-space',
        'postgres large-10 high-iops',
        '804512b7-c949-4d46-82da-ee6fc3c1cb51',
        2621440, -- storage_in_mb
        0, -- memory_in_mb
        0; -- number_of_nodes

SELECT * FROM calculate_bill();
```
