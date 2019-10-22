PREPARE cf_usage_charges_on_day(date) AS (
  WITH cf_usage_pricing_on_day AS (
    SELECT
      cud.*,
      pricing.*,
      cud.metadata || pricing.metadata AS metadata,
      cud.duration_on_day * pricing.duration AS duration
    FROM
      (EXECUTE cf_usage_on_day ($1)) cud
    LEFT JOIN pricing_plan_components_with_vat_currency_with_duration pricing ON
      pricing.plan_guid = cud.plan_guid
      AND pricing.duration && cud.duration_on_day
  )
  SELECT
    cf_usage_id,
    day,
    SUM(duration) AS running_time,
    jsonb_agg(jsonb_build_object(
      'base_price', eval_formula(metadata, duration, price_formula) * currency_rate AS base_price,
      'vat', base_price * vat_rate AS vat,
      'management_fee', CASE WHEN apply_management_fee THEN base_price * 1.1 ELSE 0 END AS management_fee,
      'total', base_price + vat + management_fee,
      'pricing_plan_id', pricing_plan_id,
      'currency_rate_id', currency_rate_id,
      'vat_rate_id', vat_rate_id
    )) AS charge_components,
    resource_type,
    resource_guid,
    resource_name,
    metadata,
    org_guid,
    space_guid,
    plan_guid,
    service_guid
  FROM
    cf_usage_pricing_on_day
  GROUP BY
    cf_usage_id,
    day,
    running_time,
    resource_type,
    resource_guid,
    resource_name,
    metadata,
    org_guid,
    space_guid,
    plan_guid,
    service_guid,
    pricing_plan_id,
    currency_rate_id,
    vat_rate_id,
    charge_components
);

CREATE TABLE cf_charges (
  cf_charge_id SERIAL,
  cf_usage_id INTEGER NOT NULL,
  day DATE NOT NULL,
  pricing_plan_id INTEGER NOT NULL,
  duration tstzrange NOT NULL,

  resource_type text NOT NULL, -- "app-run", "app-task", "app-staging", or "service" (or name of service?)
  resource_guid uuid NOT NULL, -- GUID of the app
  resource_name text NOT NULL,
  metadata JSONB NOT NULL, -- dictionary of extra data for pricing (e.g., memory per instance)
  org_guid uuid NOT NULL,
  space_guid uuid NOT NULL,
  plan_guid uuid NOT NULL,
  service_guid uuid,

  PRIMARY KEY (cf_usage_id, day, pricing_plan_id),
  CONSTRAINT duration_must_not_be_empty CHECK (not isempty(duration))
);
