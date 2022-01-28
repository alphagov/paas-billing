-- For the billing calculator, we can easily create the billable_by_component table and populate it with what the user wants to get the prices for, then call the calculate_bill
-- stored function to calculate the actual bill. This means that the calculation of prospective bills and real bills uses exactly the same code and formulae.

-- The output from this needs to be granular; it is easy to aggregate the results further in the calling stored function/procedure. Alternatively, the calling
--   stored function/procedure can aggregate directly from the billable_by_component table.
CREATE OR REPLACE FUNCTION calculate_bill ()
RETURNS TABLE
(
    org_name TEXT,
    org_guid UUID,
    org_quota_definition_guid UUID,
    plan_guid UUID,
    plan_name TEXT,
    space_guid UUID,
    space_name TEXT,
    resource_guid UUID,
    resource_type TEXT,
    resource_name TEXT,
    component_name TEXT,
    charge_usd_exc_vat DECIMAL,
    charge_gbp_exc_vat DECIMAL,
    charge_gbp_inc_vat DECIMAL
)
LANGUAGE plpgsql AS $$
DECLARE _unprocessed_formula VARCHAR DEFAULT NULL;
BEGIN
    TRUNCATE TABLE billable_by_component;
    DROP TABLE IF EXISTS billable_by_component_fx;
    DROP TABLE IF EXISTS charges_formulae;

    -- ----------------------------------------------------------------------------------------------------------------------------------------------------------
    -- 1. Get records for each AWS resource, taking account of any changes in charge amounts/formulae during the interval for which the bill is being calculated.
    -- ----------------------------------------------------------------------------------------------------------------------------------------------------------

    -- Get a unique copy of some of the charges table before running the UNION ALL query below.
    CREATE TEMPORARY TABLE charges_formulae
    AS
    SELECT DISTINCT c.plan_guid,
        c.plan_name,
        c.valid_from,
        c.valid_to,
        c.storage_in_mb,
        c.memory_in_mb,
        c.number_of_nodes,
        c.external_price,
        c.component_name,
        f.generic_formula,
        c.vat_code,
        c.currency_code
    FROM charges c
    LEFT OUTER JOIN billing_formulae f
    ON c.formula_name = f.formula_name;

    CREATE INDEX charges_formulae_i1 ON charges_formulae (plan_guid, valid_from, valid_to);

    INSERT INTO billable_by_component
    (
        valid_from,
        valid_to,
        resource_guid,
        resource_type,
        resource_name,
        org_guid,
        org_name,
        org_quota_definition_guid,
        space_guid,
        space_name,
        plan_name,
        plan_guid,
        component_name,
        time_in_seconds,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes,
        external_price,
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
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            EXTRACT(EPOCH FROM (br.valid_to - c.valid_from)), -- time_in_seconds
            COALESCE(br.storage_in_mb,c.storage_in_mb) AS storage_in_mb,
            COALESCE(br.memory_in_mb,c.memory_in_mb) AS memory_in_mb,
            COALESCE(br.number_of_nodes,c.number_of_nodes) AS number_of_nodes,
            c.external_price,
            c.generic_formula,
            c.vat_code,
            c.currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges_formulae c
    WHERE br.plan_guid = c.plan_guid
    AND br.valid_from < c.valid_from
    AND br.valid_to > c.valid_from
    AND br.valid_to <= c.valid_to
    UNION ALL
    -- charges.valid_from, charges.valid_to:  |---------------------------|
    -- Resource present:                         |-----------------|
    -- Resource present:                      |---------------------------|
    SELECT  br.valid_from,
            br.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)), -- time_in_seconds
            COALESCE(br.storage_in_mb,c.storage_in_mb) AS storage_in_mb,
            COALESCE(br.memory_in_mb,c.memory_in_mb) AS memory_in_mb,
            COALESCE(br.number_of_nodes,c.number_of_nodes) AS number_of_nodes,
            c.external_price,
            c.generic_formula,
            c.vat_code,
            c.currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges_formulae c
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
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            EXTRACT(EPOCH FROM (c.valid_to - br.valid_from)), -- time_in_seconds
            COALESCE(br.storage_in_mb,c.storage_in_mb) AS storage_in_mb,
            COALESCE(br.memory_in_mb,c.memory_in_mb) AS memory_in_mb,
            COALESCE(br.number_of_nodes,c.number_of_nodes) AS number_of_nodes,
            c.external_price,
            c.generic_formula,
            c.vat_code,
            c.currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges_formulae c
    WHERE br.plan_guid = c.plan_guid
    AND br.valid_from >= c.valid_from
    AND br.valid_from < c.valid_to
    AND br.valid_to > c.valid_to
    UNION ALL
    -- charges.valid_from, charges.valid_to:  |---------------------------|
    -- Resource present:            |---------------------------------------------|
    SELECT  c.valid_from,
            c.valid_to,
            br.resource_guid,
            br.resource_type,
            br.resource_name,
            br.org_guid,
            br.org_name,
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            c.component_name,
            EXTRACT(EPOCH FROM (c.valid_to - c.valid_from)), -- time_in_seconds
            COALESCE(br.storage_in_mb,c.storage_in_mb) AS storage_in_mb,
            COALESCE(br.memory_in_mb,c.memory_in_mb) AS memory_in_mb,
            COALESCE(br.number_of_nodes,c.number_of_nodes) AS number_of_nodes,
            c.external_price,
            c.generic_formula,
            c.vat_code,
            c.currency_code,
            0,
            CASE WHEN generic_formula IS NOT NULL AND generic_formula LIKE '%*%' THEN FALSE ELSE NULL END
    FROM billable_resources br,
         charges_formulae c
    WHERE br.plan_guid = c.plan_guid
    AND br.valid_from < c.valid_from
    AND br.valid_to > c.valid_to;

    -- -------------------------------------------------
    -- 2. Calculate the charge(s) for each AWS resource.
    -- -------------------------------------------------

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * time_in_seconds * (memory_in_mb::DECIMAL/1024.0) * (0.01 / 3600)) * external_price,
        is_processed = TRUE
    WHERE generic_formula = '($number_of_nodes * $time_in_seconds * ($memory_in_mb/1024.0) * (0.01 / 3600)) * external_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = ceil(time_in_seconds::DECIMAL/3600) * external_price,
        is_processed = TRUE
    WHERE generic_formula = 'ceil($time_in_seconds/3600) * external_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = number_of_nodes * ceil(time_in_seconds::DECIMAL/3600) * external_price,
        is_processed = TRUE
    WHERE generic_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * external_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * ceil(time_in_seconds::DECIMAL / 3600) * (memory_in_mb/1024.0) * 0.01) * external_price,
        is_processed = TRUE
    WHERE generic_formula = '($number_of_nodes * ceil($time_in_seconds / 3600) * ($memory_in_mb/1024.0) * 0.01) * external_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = number_of_nodes * time_in_seconds * (memory_in_mb::DECIMAL/1024.0) * (0.01 / 3600),
        is_processed = TRUE
    WHERE generic_formula = '$number_of_nodes * $time_in_seconds * ($memory_in_mb/1024.0) * (0.01 / 3600)'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = number_of_nodes * ceil(time_in_seconds::DECIMAL / 3600) * (memory_in_mb/1024.0) * 0.01,
        is_processed = TRUE
    WHERE generic_formula = '$number_of_nodes * ceil($time_in_seconds / 3600) * ($memory_in_mb/1024.0) * 0.01'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (storage_in_mb/1024) * (time_in_seconds::DECIMAL/2678401) * external_price,
        is_processed = TRUE
    WHERE generic_formula = '($storage_in_mb/1024) * ($time_in_seconds/2678401) * external_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * time_in_seconds * (memory_in_mb::DECIMAL/1024.0) * (external_price / 3600)),
        is_processed = TRUE
    WHERE generic_formula = '($number_of_nodes * $time_in_seconds * ($memory_in_mb/1024.0) * (external_price / 3600))'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * ceil(time_in_seconds::DECIMAL / 3600) * (memory_in_mb/1024.0) * 0.01) * external_price,
        is_processed = TRUE
    WHERE generic_formula = '$number_of_nodes * ceil($time_in_seconds / 3600) * ($memory_in_mb/1024.0) * external_price'
    AND billable_by_component.charge_usd_exc_vat = 0;

    UPDATE billable_by_component
    SET charge_usd_exc_vat = (number_of_nodes * time_in_seconds * (memory_in_mb::DECIMAL/1024.0) * (external_price / 3600)),
        is_processed = TRUE
    WHERE generic_formula = '$number_of_nodes * $time_in_seconds * ($memory_in_mb/1024.0) * (external_price / 3600)'
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
        org_quota_definition_guid,
        space_guid,
        space_name,
        plan_name,
        plan_guid,
        component_name,
        time_in_seconds,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes,
        external_price,
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (br.valid_to - c.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (br.valid_to - c.valid_from)))::NUMERIC/time_in_seconds),
            -- Following line assumes the charges accrue evenly through the whole billing interval. The formulae that are used also assume this.
            -- The calculation in the following line is: amount in USD * USD/GBP exchange rate * (time this USD/GBP exchange rate is active / time interval we're billing for)
            br.charge_usd_exc_vat * c.rate * ((EXTRACT(EPOCH FROM (br.valid_to - c.valid_from)))::NUMERIC/time_in_seconds)
    FROM billable_by_component br,
         currency_exchange_rates c
    WHERE br.valid_from < c.valid_from
    AND   br.valid_to > c.valid_from
    AND   br.valid_to <= c.valid_to
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: amount in USD * USD/GBP exchange rate * (time this USD/GBP exchange rate is active / time interval we're billing for)
            br.charge_usd_exc_vat * c.rate * ((EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)))::NUMERIC/time_in_seconds)
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (c.valid_to - br.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (c.valid_to - br.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: amount in USD * USD/GBP exchange rate * (time this USD/GBP exchange rate is active / time interval we're billing for)
            br.charge_usd_exc_vat * c.rate * ((EXTRACT(EPOCH FROM (c.valid_to - br.valid_from)))::NUMERIC/time_in_seconds)
    FROM billable_by_component br,
         currency_exchange_rates c
    WHERE br.valid_from >= c.valid_from
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (c.valid_to - c.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (c.valid_to - c.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: amount in USD * USD/GBP exchange rate * (time this USD/GBP exchange rate is active / time interval we're billing for)
            br.charge_usd_exc_vat * c.rate * ((EXTRACT(EPOCH FROM (c.valid_to - c.valid_from)))::NUMERIC/time_in_seconds)
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
        org_quota_definition_guid,
        space_guid,
        space_name,
        plan_name,
        plan_guid,
        component_name,
        time_in_seconds,
        storage_in_mb,
        memory_in_mb,
        number_of_nodes,
        external_price,
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (br.valid_to - v.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (br.valid_to - v.valid_from)))::NUMERIC/time_in_seconds),
            br.charge_gbp_exc_vat * ((EXTRACT(EPOCH FROM (br.valid_to - v.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: charge including VAT * (proportion of time this VAT rate is active versus time we're billing for)
            (br.charge_gbp_exc_vat + (br.charge_gbp_exc_vat * v.vat_rate)) * ((EXTRACT(EPOCH FROM (br.valid_to - v.valid_from)))::NUMERIC/time_in_seconds) -- charge_inc_vat
            -- The charge including VAT is: charge_gbp_exc_vat (charge excluding VAT) + VAT charge
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from < v.valid_from
    AND   br.valid_to > v.valid_from
    AND   br.valid_to <= v.valid_to
    AND   v.vat_code = br.vat_code
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)))::NUMERIC/time_in_seconds),
            br.charge_gbp_exc_vat * ((EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: charge including VAT * (proportion of time this VAT rate is active versus time we're billing for)
            (br.charge_gbp_exc_vat + (br.charge_gbp_exc_vat * v.vat_rate)) * ((EXTRACT(EPOCH FROM (br.valid_to - br.valid_from)))::NUMERIC/time_in_seconds) -- charge_inc_vat
            -- The charge including VAT is: charge_gbp_exc_vat (charge excluding VAT) + VAT charge
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from >= v.valid_from
    AND   br.valid_from < v.valid_to
    AND   br.valid_to > v.valid_from
    AND   br.valid_to <= v.valid_to
    AND   v.vat_code = br.vat_code
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (v.valid_to - br.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (v.valid_to - br.valid_from)))::NUMERIC/time_in_seconds),
            br.charge_gbp_exc_vat * ((EXTRACT(EPOCH FROM (v.valid_to - br.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: charge including VAT * (proportion of time this VAT rate is active versus time we're billing for)
            (br.charge_gbp_exc_vat + (br.charge_gbp_exc_vat * v.vat_rate)) * ((EXTRACT(EPOCH FROM (v.valid_to - br.valid_from)))::NUMERIC/time_in_seconds) -- charge_inc_vat
            -- The charge including VAT is: charge_gbp_exc_vat (charge excluding VAT) + VAT charge
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from >= v.valid_from
    AND   br.valid_from < v.valid_to
    AND   br.valid_to > v.valid_to
    AND   v.vat_code = br.vat_code
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
            br.org_quota_definition_guid,
            br.space_guid,
            br.space_name,
            br.plan_name,
            br.plan_guid,
            br.component_name,
            EXTRACT(EPOCH FROM (v.valid_to - v.valid_from)),
            br.storage_in_mb,
            br.memory_in_mb,
            br.number_of_nodes,
            br.external_price,
            br.generic_formula,
            br.is_processed,
            br.vat_code,
            br.currency_code,
            br.charge_usd_exc_vat * ((EXTRACT(EPOCH FROM (v.valid_to - v.valid_from)))::NUMERIC/time_in_seconds),
            br.charge_gbp_exc_vat * ((EXTRACT(EPOCH FROM (v.valid_to - v.valid_from)))::NUMERIC/time_in_seconds),
            -- The calculation in the following line is: charge including VAT * (proportion of time this VAT rate is active versus time we're billing for)
            (br.charge_gbp_exc_vat + (br.charge_gbp_exc_vat * v.vat_rate)) * ((EXTRACT(EPOCH FROM (v.valid_to - v.valid_from)))::NUMERIC/time_in_seconds) -- charge_inc_vat
            -- The charge including VAT is: charge_gbp_exc_vat (charge excluding VAT) + VAT charge
    FROM billable_by_component_fx br,
         vat_rates_new v
    WHERE br.valid_from < v.valid_from
    AND   br.valid_to > v.valid_to
    AND   v.vat_code = br.vat_code;

    RETURN QUERY
    SELECT bac.org_name,
           bac.org_guid,
           bac.org_quota_definition_guid,
           bac.plan_guid,
           bac.plan_name,
           bac.space_guid,
           bac.space_name,
           bac.resource_guid,
           bac.resource_type,
           bac.resource_name,
           bac.component_name,
           SUM(bac.charge_usd_exc_vat) AS charge_usd_exc_vat,
           SUM(bac.charge_gbp_exc_vat) AS charge_gbp_exc_vat,
           SUM(bac.charge_gbp_inc_vat) AS charge_gbp_inc_vat
    FROM billable_by_component bac
    GROUP BY bac.org_name,
             bac.org_guid,
             bac.org_quota_definition_guid,
             bac.plan_guid,
             bac.plan_name,
             bac.space_guid,
             bac.space_name,
             bac.resource_guid,
             bac.resource_type,
             bac.resource_name,
             bac.component_name;
END
$$;
