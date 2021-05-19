alter table orgs add column if not exists owner text not null check (length(name)>0);
