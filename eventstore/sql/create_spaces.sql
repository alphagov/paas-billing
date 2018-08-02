create table if not exists spaces (
	guid uuid not null,
	valid_from timestamptz not null,
	name text not null check (length(name)>0),
	created_at timestamptz not null,
	updated_at timestamptz not null,

	primary key (guid, valid_from)
);
