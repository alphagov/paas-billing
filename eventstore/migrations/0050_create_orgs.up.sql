-- **do not alter - add new migrations instead**

-- "migration" written before we had proper migration handling, hence the
-- various attempts at mitigating previously existing objects

BEGIN;

create table if not exists orgs (
	guid uuid not null,
	valid_from timestamptz not null,
	name text not null check (length(name)>0),
	owner text not null default 'Owner not set',
	created_at timestamptz not null,
	updated_at timestamptz not null,
	quota_definition_guid uuid,

	primary key (guid, valid_from)
);

COMMIT;
