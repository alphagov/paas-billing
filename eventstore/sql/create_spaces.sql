create table if not exists spaces (
	guid uuid not null,
	valid_from timestamptz not null,
	created_at timestamptz not null,
	updated_at timestamptz,
	space_name text not null check (length(space_name)>0),
	organization_guid uuid not null,
	org_url text not null,
	quota_definition_guid uuid not null,
	isolation_segment_guid uuid not null,

	primary key (guid, valid_from)
);
