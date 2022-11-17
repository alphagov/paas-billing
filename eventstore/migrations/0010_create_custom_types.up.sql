-- **do not alter - add new migrations instead**

-- "migration" written before we had proper migration handling, hence the
-- various attempts at mitigating previously existing objects

BEGIN;

DO $$ BEGIN
CREATE TYPE vat_code AS ENUM ('Standard', 'Reduced', 'Zero');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
CREATE TYPE currency_code AS ENUM ('USD', 'GBP', 'EUR');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
CREATE TYPE resource_state AS ENUM ('STARTED', 'STOPPED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

COMMIT;
