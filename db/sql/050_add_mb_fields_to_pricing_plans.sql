-- FIXME: we can use an `IF NOT EXISTS` clause instead of catching exceptions when
-- we upgrade to Postgres 9.6
DO $$
    BEGIN
        BEGIN
            ALTER TABLE pricing_plans ADD COLUMN memory_in_mb INTEGER DEFAULT 0;
        EXCEPTION
            WHEN duplicate_column THEN RAISE NOTICE 'column memory_in_mb already exists in pricing_plans';
        END;

        BEGIN
            ALTER TABLE pricing_plans ADD COLUMN storage_in_mb INTEGER DEFAULT 0;
        EXCEPTION
            WHEN duplicate_column THEN RAISE NOTICE 'column storage_in_mb already exists in pricing_plans';
        END;
    END;
$$
