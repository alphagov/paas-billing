INSERT INTO resources
SELECT  DISTINCT LOWER(duration) AS "valid_from",
        UPPER(duration) AS "valid_to",
        resource_guid,
        resource_name,
        resource_type,
        org_guid,
        org_name,
        space_guid,
        space_name,
        plan_name,
        plan_guid,
        event_guid AS "cf_event_guid",
        NOW()
FROM billable_event_components;
