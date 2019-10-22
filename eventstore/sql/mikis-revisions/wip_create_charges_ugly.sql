PREPARE cf_usage_charges_on_day(date) AS (
  WITH
  cf_usage_on_day AS (
    SELECT
      *,
      $1 as day,
      duration * tstzrange($1, LEAST($1 + INTERVAL '1 DAY', now()), '[)') AS duration_on_day
    FROM
      cf_usage
    WHERE
      duration && tstzrange($1, LEAST($1 + INTERVAL '1 DAY', now()), '[)')
  ),
  cf_usage_pricing_on_day AS (
    SELECT
      cud.*,
      pricing.*,
      cud.metadata || pricing.metadata AS zz_metadata,
      cud.duration_on_day * pricing.duration AS zz_duration
    FROM
      cf_usage_on_day cud
    LEFT JOIN pricing_plan_components_with_vat_currency_with_duration pricing ON
      pricing.plan_guid = cud.plan_guid
      AND pricing.duration && cud.duration_on_day
  )
  SELECT
    cf_usage_id,
    day,
    SUM(UPPER(zz_duration) - LOWER(zz_duration)) AS running_time,
    jsonb_agg(jsonb_build_object(
      'base_price', eval_formula(zz_metadata, zz_duration, price_formula) * currency_rate,
      'vat', eval_formula(zz_metadata, zz_duration, price_formula) * currency_rate * vat_rate,
      'management_fee', CASE WHEN apply_management_fee THEN eval_formula(zz_metadata, zz_duration, price_formula) * currency_rate * 1.1 ELSE 0 END,
      'total', eval_formula(zz_metadata, zz_duration, price_formula) * currency_rate + eval_formula(zz_metadata, zz_duration, price_formula) * currency_rate * vat_rate + CASE WHEN apply_management_fee THEN eval_formula(zz_metadata, zz_duration, price_formula) * currency_rate * 1.1 ELSE 0 END,
      'pricing_plan_id', pricing_plan_id,
      'currency_rate_id', currency_rate_id,
      'vat_rate_id', vat_rate_id
    )) AS charge_components,
    resource_type,
    resource_guid,
    resource_name,
    zz_metadata,
    org_guid,
    space_guid,
    service_guid
  FROM
    cf_usage_pricing_on_day
  GROUP BY
    cf_usage_id,
    day,
    resource_type,
    resource_guid,
    resource_name,
    zz_metadata,
    org_guid,
    space_guid,
    service_guid,
    pricing_plan_id,
    currency_rate_id,
    vat_rate_id
);
