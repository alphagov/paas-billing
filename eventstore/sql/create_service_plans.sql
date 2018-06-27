create table if not exists service_plans (
	guid uuid not null,
	valid_from timestamptz not null,
	created_at timestamptz not null,
	updated_at timestamptz,
	name text not null check (length(name)>0),
	description text not null,
	unique_id uuid not null,
	service_guid uuid not null,
	service_valid_from timestamptz not null,
	active boolean not null,
	public boolean not null,
	free boolean not null,
	extra text,

	foreign key (service_guid, service_valid_from) references services (guid, valid_from),
	primary key (guid, valid_from)
);

alter table service_plans alter column unique_id type text using unique_id::text;
