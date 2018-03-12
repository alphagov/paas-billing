DO $$
    BEGIN
        BEGIN
            ALTER TABLE pricing_plan_components
            ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'GBP';

            ALTER TABLE pricing_plan_components
            ALTER COLUMN currency DROP DEFAULT;

            ALTER TABLE pricing_plan_components
            ADD CONSTRAINT currency_valid
            CHECK (currency = ANY ('{GBP, USD, EUR}'::text[]));

            EXCEPTION
            WHEN duplicate_column
            THEN RAISE NOTICE 'column <column_name> already exists in <table_name>.';
        END;
    END;
$$;
