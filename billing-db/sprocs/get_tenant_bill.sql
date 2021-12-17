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
    plan_guid UUID,
    plan_name TEXT,
    space_name TEXT,
    resource_type TEXT,
    resource_name TEXT,
    component_name TEXT,
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
    -- Case 0:
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
    AND   r.valid_to <= _to_date
    UNION ALL
    -- Case 1:
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
    -- Case 2:
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
    AND   r.valid_from >= _from_date
    AND   r.valid_from < _to_date
    AND   r.valid_to > _to_date
    UNION ALL
    -- Case 3:
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
