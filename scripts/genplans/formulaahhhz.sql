
delete from pricing_plan_components;
delete from pricing_plans;

delete from currency_rates;
insert into currency_rates (code, valid_from, rate) values
	('GBP', '2001-01-01', 1),
	('USD', '2001-01-01', 0.8)
;

insert into pricing_plans (
	id, name, valid_from, plan_guid,
	memory_in_mb, storage_in_mb, number_of_nodes
) values 
	(1, 'compute', '2010-01-01 00:00:00+00', 'f4d4b95a-f55e-4593-8d54-3364c25798c4', 
		0, 0, 1),
	
	(20, 'xlarge postgres9.5 high-availability enc', '2001-01-01 00:00:00+00', '7ae6a962-50f2-4fbd-acd1-f8f3dc5b1301', 
		0, 2*1024*1024, 1),

	(2, 'large postgres9.5 high-availability enc', '2001-01-01 00:00:00+00', '1bc93270-3d89-4ddd-a916-53e1b1d899e9', 
		0, 512*1024, 1),

	(17, 'large postgres9.5 high-availability', '2001-01-01 00:00:00+00', '5ca3b793-9a59-4447-8bbe-64e052102bb6', 
		0, 512*1024, 1),

	(14, 'medium postgres9.5 high-availability enc', '2001-01-01 00:00:00+00', '3de5429b-424a-412b-b3a7-b6b08688ce5c', 
		0, 100*1024, 1),

	(8, 'medium postgres9.5 high-availability', '2001-01-01 00:00:00+00', 'ef9ee73a-7e82-47e6-9f1b-126cdb9ef49c', 
		0, 100*1024, 1),

	(15, 'large postgres9.5', '2001-01-01 00:00:00+00', '8a7346f9-2c91-473c-aaba-e1a5ed8ce036', 
		0, 512*1024, 1),

	(7, 'medium postgres9.5', '2001-01-01 00:00:00+00', '7bbefbf0-70c1-4ae4-88af-fc2170c0b4d6', 
		0, 100*1024, 1),

	(6, 'small postgres9.5 high-availability enc', '2001-01-01 00:00:00+00', '1f4439d4-5010-480e-8fe0-9e7a2be6fe90', 
		0, 20*1024, 1),

	(16, 'small postgres9.5 high-availability', '2001-01-01 00:00:00+00', 'dbd330b0-13e7-47b5-aec2-6303430c4776', 
		0, 20*1024, 1),

	(12, 'small postgres9.5', '2001-01-01 00:00:00+00', '4b47d304-003d-4771-a8ba-281adc90b2a6', 
		0, 20*1024, 1),

	(4, 'tiny postgres9.5', '2001-01-01 00:00:00+00', 'e264800e-20cb-4bf0-99fc-84bd42681d81', 
		0, 5*1024, 1),

	(3, 'tiny elasticsearch [compose]', '2001-01-01 00:00:00+00', '31bb8b32-d5e0-4310-8a8d-3e8bff4b2bbc', 
		0, 0, 1),

	(13, 'tiny redis [compose]', '2001-01-01 00:00:00+00', '53ca5c56-5474-4d64-9211-fe9aee86d502',
	       0, 0, 1),

	(10, 'tiny mongodb', '2001-01-01 00:00:00+00', '5d56dd1b-52d5-4ffb-a034-fc5c79794c90', 
		0, 0, 1),

	(9, 'tiny redis clustered', '2001-01-01 00:00:00+00', '957e6177-323c-4eeb-8630-c4bfa979a86c', 
		0, 0, 1),

	(5, 'tiny mysql', '2001-01-01 00:00:00+00', 'c03510d2-54bd-4fb0-9d9a-1e5ffb82aa39', 
		0, 5*1024, 1),

	(11, 'cloudfront cdn', '2001-01-01 00:00:00+00', 'e4f39630-e385-410b-972f-6b4cc7aad112', 
		0, 0, 0)
;

insert into pricing_plan_components (
	pricing_plan_id, name, formula, currency
) values 
	-- app instances
	-- per minute hour (minimm of 1hour) 0.01USD per GB/hr (based on division of 30GB r4.xlarge cell)
	-- overprovisioning (for HA/platform etc) is charged at X% of the memory-based formula
	(1, 'instance', '$number_of_nodes * ceil($time_in_seconds / 3600) * ($memory_in_mb/1024.0) * 0.01', 'USD'),
	(1, 'ha, routing & provisioning', '0.40 * ($number_of_nodes * (ceil($time_in_seconds / 60) / 60) * ($memory_in_mb/1024.0) * 0.01)', 'USD'), -- perentage of ($same_formula_as_above)

	-- XL m4.4xlarge postgres
	(20, 'compute', 'ceil($time_in_seconds/3600) * 1.612', 'USD'),
	(20, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- postgres m4.2xlarge
	-- RDS instances are per hour billing
	-- RDS storage is per month billing
	(2, 'compute', 'ceil($time_in_seconds/3600) * 0.806', 'USD'),
	(2, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- postgres small (t2.small)
	(6, 'instance', 'ceil($time_in_seconds/3600)*0.039', 'USD'),
	(6, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- Compose Elasticsearch (tiny) 
	(3, 'estimate', 'ceil($time_in_seconds/2678401)*45.00', 'USD'),

	-- Compose Redis (tiny)
	(13, 'estimate', 'ceil($time_in_seconds/2678401)*6.64', 'USD'),

	-- Postgres M-HA-enc-dedicated-9.5 (m4.large)
	(14, 'instance', 'ceil($time_in_seconds/3600) * 0.402', 'USD'),
	(14, 'storage', '($storage_in_mb/1024) * ($time_in_seconds/2678401) * 0.127', 'USD'),

	-- Postgres db.t2.small (S-dedicated-9.5)
	(12, 'instance', 'ceil($time_in_seconds/3600) * 0.039', 'USD'),
	(12, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- Postgres db.m4.2xlarge (L-HA-dedicated-9.5)
	(17, 'instance', 'ceil($time_in_seconds/3600) * 1.612', 'USD'),
	(17, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- Compose Mongo (tiny)
	(10, 'estimate', 'ceil($time_in_seconds/2678401)*31.00', 'USD'),

	-- Postgres db.m4.large (M-dedicated-9.5)
	(7, 'instance', 'ceil($time_in_seconds/3600) * 0.201', 'USD'),
	(7, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),
	
	-- (15, 'Postgres db.m4.2xlarge (L-dedicated-9.5)', '2001-01-01 00:00:00+00', '8a7346f9-2c91-473c-aaba-e1a5ed8ce036', 0, 0, 1),
	(15, 'instance', 'ceil($time_in_seconds/3600) * 0.806', 'USD'),
	(15, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),
	
	-- (9, 'Elasticahe Redis cache.t2.micro (tiny)', '2001-01-01 00:00:00+00', '957e6177-323c-4eeb-8630-c4bfa979a86c', 0, 0, 1),
	(9, 'estimate', '$number_of_nodes * ceil($time_in_seconds/3600) * 0.018', 'USD'),
	
	-- (5, 'MySQL db.t2.micro (Free)', '2001-01-01 00:00:00+00', 'c03510d2-54bd-4fb0-9d9a-1e5ffb82aa39', 0, 0, 1),
	(5, 'instance', 'ceil($time_in_seconds/3600) * 0.018', 'USD'),
	(5, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),
	
	-- (16, 'Postgres db.t2.small (S-HA-dedicated-9.5)', '2001-01-01 00:00:00+00', 'dbd330b0-13e7-47b5-aec2-6303430c4776', 0, 0, 1),
	(16, 'instance', 'ceil($time_in_seconds/3600) * 0.078', 'USD'),
	(16, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),
	
	-- (4, 'Postgres db.t2.micro (Free)', '2001-01-01 00:00:00+00', 'e264800e-20cb-4bf0-99fc-84bd42681d81', 0, 0, 1),
	(4, 'instance', 'ceil($time_in_seconds/3600) * 0.02', 'USD'),
	(4, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- (8, 'Postgres db.m4.large (M-HA-dedicated-9.5)', '2001-01-01 00:00:00+00', 'ef9ee73a-7e82-47e6-9f1b-126cdb9ef49c', 0, 0, 1)
	(8, 'instance', 'ceil($time_in_seconds/3600) * 0.402', 'USD'),
	(8, 'storage', '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127', 'USD'),

	-- (11, 'Cloudfront CDN (custom domain)', '2001-01-01 00:00:00+00', 'e4f39630-e385-410b-972f-6b4cc7aad112', 0, 0, 1),
	(11, 'cloudfront', '0', 'GBP')
;

-- dump new data to screen
\echo New plan configuration
select * from pricing_plans;
select * from pricing_plan_components;

-- show missing plans
\echo The following plans are not accounted for
with plans as (select distinct raw_message->>'service_label' as plan_kind,raw_message->>'service_plan_name' as plan_name, raw_message->>'service_plan_guid' as plan_guid from service_usage_events)
, pricing_plans as (select distinct plan_guid from pricing_plans)
select * from plans p where (select count(*) from pricing_plans pp where pp.plan_guid = p.plan_guid) = 0;

-- update usage view
refresh materialized view resource_usage;

