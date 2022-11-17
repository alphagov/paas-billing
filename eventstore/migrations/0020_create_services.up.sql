create table if not exists services (
	guid uuid not null,
	valid_from timestamptz not null,
	created_at timestamptz not null,
	updated_at timestamptz,
	label text not null check (length(label)>0),
	description text not null,
	active bool not null,
	bindable bool not null,
	service_broker_guid uuid not null,

	primary key (guid, valid_from)
);
