-- When creating or recreating a usage and adoption spreadsheet
-- this file can be used to query the data

-- To generate a CSV via the psql utility you can:
--
-- * Use "\a" to ensure your output is unaligned
-- * Use "\f ," to output the results separated by commas
-- * Use "\o /tmp/results-ireland.csv" to save the output in a CSV
-- * Use "\i scripts/usage-and-adoption-generate-csv.sql" to run this query

WITH
distinct_orgs_first_seen AS (
  SELECT
    DISTINCT ON (org_guid)
      org_guid,
      LAST_VALUE(org_name)
        OVER (
          PARTITION BY (org_guid)
          ORDER BY LOWER(duration) ASC
          RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        )
      AS org_name,
      FIRST_VALUE(LOWER(duration)::date)
        OVER (
          PARTITION BY (org_guid)
          ORDER BY LOWER(duration) ASC
          RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        )
      AS seen
  FROM billable_event_components
),
org_bills_by_plan_by_month AS
  (SELECT date_trunc('month', DAY)::date AS mon,
          'london' AS region,
          org_guid,
          CASE
              WHEN plan_name = 'app' THEN 'compute'
              WHEN plan_name = 'staging' THEN 'compute'
              WHEN plan_name = 'task' THEN 'compute'
              WHEN plan_name ~ 'postgres.*' THEN 'postgres'
              WHEN plan_name ~ 'mysql.*' THEN 'mysql'
              WHEN plan_name ~ 'elasticsearch.*' THEN 'elasticsearch'
              WHEN plan_name ~ 'influx.*' THEN 'influxdb'
              WHEN plan_name ~ 'redis.*' THEN 'redis'
              WHEN plan_name ~ 'aws-s3.*' THEN 's3'
              ELSE plan_name::text
          END AS service,
          sum(cost) AS total_cost
   FROM tmp_billable_event_components_by_day
   GROUP BY mon, region, org_guid, service
),
aggregated_org_bills_by_plan_by_month AS (
  SELECT mon,
         region,
         org_guid,
         jsonb_object(array_agg(service), array_agg(total_cost::text)) AS service_costs
   FROM org_bills_by_plan_by_month
   GROUP BY mon, region, org_guid
)
SELECT mon,
       '' as department,
       region,
       aggregated_org_bills_by_plan_by_month.org_guid,
       org_name,
       seen::date AS first_seen,
       COALESCE(service_costs->>'compute', '0') AS compute,
       COALESCE(service_costs->>'postgres', '0') AS postgres,
       COALESCE(service_costs->>'mysql', '0') AS mysql,
       COALESCE(service_costs->>'elasticsearch', '0') AS elasticsearch,
       COALESCE(service_costs->>'redis', '0') AS redis,
       COALESCE(service_costs->>'s3', '0') AS s3,
       COALESCE(service_costs->>'influxdb', '0') AS influxdb,

       COALESCE(service_costs->>'compute', '0')::numeric
       + COALESCE(service_costs->>'postgres', '0')::numeric
       + COALESCE(service_costs->>'mysql', '0')::numeric
       + COALESCE(service_costs->>'elasticsearch', '0')::numeric
       + COALESCE(service_costs->>'redis', '0')::numeric
       + COALESCE(service_costs->>'s3', '0')::numeric
       + COALESCE(service_costs->>'influxdb', '0')::numeric
       AS total
FROM aggregated_org_bills_by_plan_by_month JOIN distinct_orgs_first_seen
ON aggregated_org_bills_by_plan_by_month.org_guid = distinct_orgs_first_seen.org_guid
WHERE true
      AND org_name NOT LIKE 'AIVENBACC%'
      AND org_name NOT LIKE 'BACC%'
      AND org_name NOT LIKE 'ACC%'
      AND org_name NOT LIKE 'SMOKE%'
      AND org_name !~* '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'
ORDER BY mon ASC
