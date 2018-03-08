-- in Postgres 9.5 there is no 'IF NOT EXISTS' when adding a column
DO $$
    BEGIN
        BEGIN
            ALTER TABLE pricing_plan_components ADD COLUMN vat_rate_id INTEGER REFERENCES vat_rates (id) DEFAULT 1;
        EXCEPTION
            WHEN duplicate_column THEN RAISE NOTICE 'column vat_rate_id already exists in pricing_plan_components';
        END;
    END;
$$
