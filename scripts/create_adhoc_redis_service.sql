-- Add the missing historic recorded service in prod
--
--  redis tiny (compose)
--
-- Some events refer to this service plan with GUID:
--
--  53ca5c56-5474-4d64-9211-fe9aee86d502
--
INSERT INTO services (
	guid,
	valid_from,
	created_at,
	updated_at,
	label,
	description,
	active,
	bindable,
	service_broker_guid
) (
    SELECT
        '2e5bc51e-b210-41e1-b0b7-e0ae5566c801'::uuid,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        'redis-compose',
        'Redis instance',
        false,
        false,
        '30b034f9-1eb0-4a44-8f16-d4d76baba415'::uuid
    WHERE
        '2e5bc51e-b210-41e1-b0b7-e0ae5566c801'::uuid NOT IN (
            SELECT DISTINCT guid FROM services
        )
);

INSERT INTO service_plans (
    guid,
    valid_from,
    created_at,
    updated_at,
    name,
    description,
    unique_id,
    service_guid,
    service_valid_from,
    active,
    public,
    free,
    extra
) (
    SELECT
        -- Old GUID in the existing events in prod
        '53ca5c56-5474-4d64-9211-fe9aee86d502'::uuid,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        'redis tiny (compose)',
        'redis tiny (compose)',
        -- new unique ID we want to match
        'a8574a4b-9c6c-40ea-a0df-e9b7507948c8'::uuid,
        -- service.guid from above
        '2e5bc51e-b210-41e1-b0b7-e0ae5566c801'::uuid,
        '2016-01-01T00:00:00+00:00'::timestamptz,
        false,
        false,
        false,
        ''
    WHERE
        '53ca5c56-5474-4d64-9211-fe9aee86d502'::uuid NOT IN (
            SELECT DISTINCT guid FROM service_plans
        )
);

