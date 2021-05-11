-- These temporary tables need to be created when creating the database connection.

-- What needs to be billed. This can be used for any resources, past or future, so can be used by the billing calculator.
CREATE TEMPORARY TABLE billable_resources
(
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL,
    time_in_seconds INT NULL,
    resource_guid UUID NULL,
    resource_type TEXT NULL,
    resource_name TEXT NULL,
    org_guid UUID NULL,
    org_name TEXT NULL,
    space_guid UUID NULL,
    space_name TEXT NULL,
    plan_name TEXT NULL,
    plan_guid UUID NULL -- Later on this field may not be needed
);

-- The billable_by_component table needs creating before running this stored function. This is so we can preserve the contents of this table for audit/debug purposes.
CREATE TEMPORARY TABLE billable_by_component
(
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL,
    -- valid_from_month TODO - useful if we're calculating bills for more than one month
    -- valid_to_month TODO - useful if we're calculating bills for more than one month
    resource_guid UUID NULL,
    resource_type TEXT NULL,
    resource_name TEXT NULL,
    org_guid UUID NULL,
    org_name TEXT NULL,
    space_guid UUID NULL,
    space_name TEXT NULL,
    plan_name TEXT NULL,
    plan_guid UUID NULL, -- Later on this field may not be needed
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

-- For the billing calculator, we can easily create the billable_by_component table and populate it with what the user wants to get the prices for, then call the calculate_bill
-- stored function to calculate the actual bill. This means that the calculation of prospective bills and real bills uses exactly the same code and formulae.

-- The output from this needs to be granular; it is easy to aggregate the results further in the calling stored function/procedure. Alternatively, the calling
--   stored function/procedure can aggregate directly from the billable_by_component table.
CREATE OR REPLACE FUNCTION calculate_bill ()
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
DECLARE _unprocessed_formula VARCHAR DEFAULT NULL;
BEGIN
    TRUNCATE TABLE billable_by_component;
    DROP TABLE IF EXISTS billable_by_component_fx;

    -- ----------------------------------------------------------------------------------------------------------------------------------------------------------
    -- 1. Get records for each AWS resource, taking account of any changes in charge amounts/formulae during the interval for which the bill is being calculated.
    -- ----------------------------------------------------------------------------------------------------------------------------------------------------------

    INSERT INTO billable_by_component
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
        component_name,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes,
        aws_price,
        generic_formula,
        vat_code,
        currency_code,
        charge_usd_exc_vat,
        is_processed
    )
    -- charges.valid_from, charges.valid_to:  |---------------------------|
    -- Resource present:            |-----------------|
    SELECT  c.valid_from,
            br.valid_to,
            resource_guid,
            resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            storage_in_mb,
            memory_in_mb,
            number_of_nodes,
            aws_price,
            generic_formula,
            vat_code,
            currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges c
    WHERE br.plan_guid = c.plan_guid
    AND br.valid_from < c.valid_from
    AND br.valid_to > c.valid_from
    AND br.valid_to < c.valid_to
    UNION ALL
    -- charges.valid_from, charges.valid_to:  |---------------------------|
    -- Resource present:                         |-----------------|
    -- Resource present:                      |---------------------------|
    SELECT  br.valid_from,
            br.valid_to,
            resource_guid,
            resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            storage_in_mb,
            memory_in_mb,
            number_of_nodes,
            aws_price,
            generic_formula,
            vat_code,
            currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges c
    WHERE br.plan_guid = c.plan_guid
    AND   br.valid_from >= c.valid_from
    AND   br.valid_from < c.valid_to
    AND   br.valid_to > c.valid_from
    AND   br.valid_to <= c.valid_to
    UNION ALL
    -- charges.valid_from, charges.valid_to:  |---------------------------|
    -- Resource present:                                       |-----------------|
    SELECT  br.valid_from,
            c.valid_to,
            resource_guid,
            resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            storage_in_mb,
            memory_in_mb,
            number_of_nodes,
            aws_price,
            generic_formula,
            vat_code,
            currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges c
    WHERE br.plan_guid = c.plan_guid
    AND br.valid_from > c.valid_from
    AND br.valid_from < c.valid_to
    AND br.valid_to > c.valid_to
    UNION ALL
    -- charges.valid_from, charges.valid_to:  |---------------------------|
    -- Resource present:            |---------------------------------------------|
    SELECT  c.valid_from,
            c.valid_to,
            resource_guid,
            resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            storage_in_mb,
            memory_in_mb,
            number_of_nodes,
            aws_price,
            generic_formula,
            vat_code,
            currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges c
    WHERE br.plan_guid = c.plan_guid
    AND br.valid_from < c.valid_from
    AND br.valid_to > c.valid_to;

    -- -------------------------------------------------
    -- 2. Calculate the charge(s) for each AWS resource.
    -- -------------------------------------------------

    -- Only run this query where there's a generic formula populated. Could only run it when the formula depends on time_in_seconds (though virtually all will).
    UPDATE billable_by_component SET time_in_seconds = EXTRACT(EPOCH FROM (valid_to - valid_from))
    WHERE  generic_formula IS NOT NULL;
    -- AND    ((number_of_nodes != 0) OR (memory_in_mb != 0) OR (storage_in_mb != 0)); -- Very little performance gain if uncomment this line and it is risky for future formula changes.

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * time_in_seconds * (memory_in_mb::DECIMAL/1024.0) * (0.01 / 3600)) * aws_price,
        is_processed = TRUE
    WHERE generic_formula = '(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * aws_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = ceil(time_in_seconds::DECIMAL/3600) * aws_price,
        is_processed = TRUE
    WHERE generic_formula = 'ceil(time_in_seconds/3600) * aws_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = number_of_nodes * ceil(time_in_seconds::DECIMAL/3600) * aws_price,
        is_processed = TRUE
    WHERE generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * ceil(time_in_seconds::DECIMAL / 3600) * (memory_in_mb/1024.0) * 0.01) * aws_price,
        is_processed = TRUE
    WHERE generic_formula = '(number_of_nodes * ceil(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01) * aws_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = number_of_nodes * time_in_seconds * (memory_in_mb::DECIMAL/1024.0) * (0.01 / 3600),
        is_processed = TRUE
    WHERE generic_formula = 'number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = ceil(time_in_seconds::DECIMAL/3600) * aws_price,
        is_processed = TRUE
    WHERE generic_formula = 'ceil(time_in_seconds/3600) * aws_price '
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = number_of_nodes * ceil(time_in_seconds::DECIMAL / 3600) * (memory_in_mb/1024.0) * 0.01,
        is_processed = TRUE
    WHERE generic_formula = 'number_of_nodes * ceil(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01'
    AND billable_by_component.charge_usd_exc_vat = 0;

    -- TODO: Replace 0.253 with aws_hourly_charge or aws_price in the following. Will need to do this for stuff in paas-cf - create a new field. Could do this in a later iteration.
    UPDATE billable_by_component
    SET charge_usd_exc_vat = (storage_in_mb/1024) * ceil(time_in_seconds::DECIMAL/2678401) * 0.253,
        is_processed = TRUE
    WHERE generic_formula = '(storage_in_mb/1024) * ceil(time_in_seconds/2678401) * 0.253'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (storage_in_mb/1024) * ceil(time_in_seconds::DECIMAL/2678401) * 0.127,
        is_processed = TRUE
    WHERE generic_formula = '(storage_in_mb/1024) * ceil(time_in_seconds/2678401) * 0.127'
    AND billable_by_component.charge_usd_exc_vat = 0;

    -- Check that all formulae have been processed by the above updates.
    SELECT generic_formula INTO _unprocessed_formula
    FROM billable_by_component
    WHERE is_processed IS FALSE
    LIMIT 1;

    IF _unprocessed_formula IS NOT NULL THEN
        RAISE EXCEPTION 'No code present to calculate bill for the formula: "%". New UPDATE statement needed in calculate_bill stored function.', _unprocessed_formula; 
    END IF;

    -- ----------------------------------------------------------------------------------------------------------------------------------------------------------------------
    -- 3. Now take account of currency exchange rates. Changes in currency exchange rates can occur at any point during the interval for which the bill is  being calculated.
    -- ----------------------------------------------------------------------------------------------------------------------------------------------------------------------

    -- If the following queries are a brake on performance then they can be wrapped inside IF statements. There are other possible optimisations too.

    CREATE TEMPORARY TABLE billable_by_component_fx AS 
    SELECT *
    FROM billable_by_component
    WHERE 1=2;

    INSERT INTO billable_by_component_fx
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
        component_name,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes,
        aws_price,
        generic_formula,
        is_processed,
        vat_code,
        currency_code,
        charge_usd_exc_vat,
        charge_gbp_exc_vat
    )
    -- currency_exchange_rates.valid_from, currency_exchange_rates.valid_to:  |---------------------------|
    -- Resource present:                                            |-----------------|
    SELECT  c.valid_from,
            br.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_usd_exc_vat * c.rate
    FROM billable_by_component br,
         currency_exchange_rates c
    WHERE br.valid_from < c.valid_from
    AND   br.valid_to > c.valid_from
    AND   br.valid_to < c.valid_to
    AND   br.currency_code = c.from_ccy
    AND   c.to_ccy = 'GBP'
    UNION ALL
    -- currency_exchange_rates.valid_from, currency_exchange_rates.valid_to:  |---------------------------|
    -- Resource present:                                                         |-----------------|
    -- Resource present:                                                      |---------------------------|
    SELECT  br.valid_from,
            br.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_usd_exc_vat * c.rate
    FROM billable_by_component br,
         currency_exchange_rates c
    WHERE br.valid_from >= c.valid_from
    AND   br.valid_from < c.valid_to
    AND   br.valid_to > c.valid_from
    AND   br.valid_to <= c.valid_to
    AND   br.currency_code = c.from_ccy
    AND   c.to_ccy = 'GBP'
    UNION ALL
    -- currency_exchange_rates.valid_from, currency_exchange_rates.valid_to:  |---------------------------|
    -- Resource present:                                                                       |-----------------|
    SELECT  br.valid_from,
            c.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_usd_exc_vat * c.rate
    FROM billable_by_component br,
         currency_exchange_rates c
    WHERE br.valid_from > c.valid_from
    AND   br.valid_from < c.valid_to
    AND   br.valid_to > c.valid_to
    AND   br.currency_code = c.from_ccy
    AND   c.to_ccy = 'GBP'
    UNION ALL
    -- currency_exchange_rates.valid_from, currency_exchange_rates.valid_to:  |---------------------------|
    -- Resource present:                                            |---------------------------------------------|
    SELECT  c.valid_from,
            c.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_usd_exc_vat * c.rate
    FROM billable_by_component br,
         currency_exchange_rates c
    WHERE br.valid_from < c.valid_from
    AND   br.valid_to > c.valid_to
    AND   br.currency_code = c.from_ccy
    AND   c.to_ccy = 'GBP';

    -- ----------------------------------------------------------------------------------------------------------------------------------------
    -- 4. Now take account of VAT rates. Changes in VAT rate can occur at any point during the interval for which the bill is being calculated.
    -- ----------------------------------------------------------------------------------------------------------------------------------------

    TRUNCATE TABLE billable_by_component;

    INSERT INTO billable_by_component
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
        component_name,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes,
        aws_price,
        generic_formula,
        is_processed,
        vat_code,
        currency_code,
        charge_usd_exc_vat,
        charge_gbp_exc_vat,
        charge_gbp_inc_vat
    )
    -- vat_rates_new.valid_from, vat_rates_new.valid_to:  |---------------------------|
    -- Resource present:                        |-----------------|
    SELECT  v.valid_from,
            br.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_gbp_exc_vat,
            br.charge_gbp_exc_vat/(1 - v.vat_rate) -- charge_inc_vat
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from < v.valid_from
    AND   br.valid_to > v.valid_from
    AND   br.valid_to < v.valid_to
    AND   v.vat_code = 'Standard'
    UNION ALL
    -- vat_rates_new.valid_from, vat_rates_new.valid_to:  |---------------------------|
    -- Resource present:                                     |-----------------|
    -- Resource present:                                  |---------------------------|
    SELECT  br.valid_from,
            br.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_gbp_exc_vat,
            br.charge_gbp_exc_vat/(1 - v.vat_rate) -- charge_inc_vat
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from >= v.valid_from
    AND   br.valid_from < v.valid_to
    AND   br.valid_to > v.valid_from
    AND   br.valid_to <= v.valid_to
    AND   v.vat_code = 'Standard'
    UNION ALL
    -- vat_rates_new.valid_from, vat_rates_new.valid_to:  |---------------------------|
    -- Resource present:                                                   |-----------------|
    SELECT  br.valid_from,
            v.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_gbp_exc_vat,
            br.charge_gbp_exc_vat/(1 - v.vat_rate) -- charge_inc_vat
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from > v.valid_from
    AND   br.valid_from < v.valid_to
    AND   br.valid_to > v.valid_to
    AND   v.vat_code = 'Standard'
    UNION ALL
    -- vat_rates_new.valid_from, vat_rates_new.valid_to:  |---------------------------|
    -- Resource present:                        |---------------------------------------------|
    SELECT  v.valid_from,
            v.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.aws_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat,
            br.charge_gbp_exc_vat,
            br.charge_gbp_exc_vat/(1 - v.vat_rate) -- charge_inc_vat
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from < v.valid_from
    AND   br.valid_to > v.valid_to
    AND   v.vat_code = 'Standard';

    RETURN QUERY
    SELECT bac.org_name,
           bac.org_guid,
           bac.plan_name,
           bac.space_name,
           bac.resource_name,
           SUM(bac.charge_usd_exc_vat) AS charge_usd_exc_vat,
           SUM(bac.charge_gbp_exc_vat) AS charge_gbp_exc_vat,
           SUM(bac.charge_gbp_inc_vat) AS charge_gbp_inc_vat
    FROM billable_by_component bac
    GROUP BY bac.org_name, 
             bac.org_guid, 
             bac.plan_name,
             bac.space_name,
             bac.resource_name;

    DROP TABLE billable_by_component_fx;
END
$$;
