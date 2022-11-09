alter table orgs add column if not exists owner text not null default 'Owner not set';
