create table if not exists spaces (
	guid uuid not null,
	valid_from timestamptz not null,
	name text not null check (length(name)>0),
	created_at timestamptz not null,
	updated_at timestamptz not null,
	isolation_segment_guid uuid,
	quota_definition_guid uuid,

	primary key (guid, valid_from)
);
