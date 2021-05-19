alter table orgs add column if not exists owner text check (length(owner)>0);
