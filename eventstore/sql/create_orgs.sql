create table if not exists orgs (
	guid uuid not null,
	valid_from timestamptz not null,
	created_at timestamptz not null,
	updated_at timestamptz,
	org_name text not null check (length(org_name)>0),
	quota_definition_guid uuid not null,
	default_isolation_segment_guid uuid not null,

	primary key (guid, valid_from)
);
