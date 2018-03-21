-- FIXME: we can use an `IF NOT EXISTS` clause instead of catching exceptions when
-- we upgrade to Postgres 9.6
DO $$ BEGIN
	alter table pricing_plans add column number_of_nodes integer not null default 0;
EXCEPTION
	WHEN duplicate_column THEN RAISE NOTICE 'column number_of_nodes already exists in pricing_plans';
END; $$
