-- **do not alter - add new migrations instead**

-- "migration" written before we had proper migration handling, hence the
-- various attempts at mitigating previously existing objects

BEGIN;

alter table orgs add column if not exists owner text not null default 'Owner not set';

COMMIT;
