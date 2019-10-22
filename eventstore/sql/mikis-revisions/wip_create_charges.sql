-- *** WORK IN PROGRESS *** --
-- My thoughts, not yet reflected here, are that this table should
-- link up all sources of charges into one place. Fixed start/stop
-- charges from the `cf_usage`/`cf_usage_charges` pipeline will come
-- here, but so will on-demand charges for things such as S3.

-- A view optimised for returning daily/monthly org bills.
-- One row per resource with its summed charges for the day.
CREATE VIEW charges AS (
  SELECT
    *,
    SUM(duration) AS runtime, -- FIXME: can't be done here if we're taking on-demand as well. maybe unstructured metadata?
    SUM(charge_components.base_price) AS base_price,
    SUM(charge_components.vat) AS vat,
    SUM(charge_components.management_fee) AS management_fee,
    SUM(charge_components.total) AS total
  FROM
    cf_usage_prices
  GROUP BY
    resource_guid,
    charge_components
);
