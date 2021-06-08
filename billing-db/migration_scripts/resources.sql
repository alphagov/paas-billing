INSERT INTO resources_reference
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
        number_of_nodes,
        memory_in_mb,
        storage_in_mb,
        event_guid AS "cf_event_guid",
        TIMESTAMPTZ('2021-06-08T00:00:00') AS last_updated
FROM billable_event_components;
WHERE space_name != '^(SMOKE|ACC|CATS|PERF|BACC|AIVENBACC|ASATS)'
