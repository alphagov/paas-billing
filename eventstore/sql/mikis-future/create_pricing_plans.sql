CREATE TABLE pricing_plans (
  sequence SERIAL UNIQUE,
  plan_id text NOT NULL,
  valid_from timestamptz NOT NULL,

  name text NOT NULL,
  metadata JSONB NOT NULL, -- pricing metadata, e.g., memory_in_mb number_of_nodes storage_in_mb

  PRIMARY KEY (plan_id, valid_from)
);

CREATE TABLE pricing_plan_components (
  pricing_plan_sequence INTEGER, -- simplify how components are joined to plans: use an autoincrement ID not a duplicate valid_from
  name text NOT NULL,
  price_formula text NOT NULL,
  vat_code text NOT NULL,
  currency_code text NOT NULL,
  apply_management_fee boolean NOT NULL,

  PRIMARY KEY (pricing_plan_sequence, name),
  FOREIGN KEY (pricing_plan_sequence) REFERENCES pricing_plans (sequence) ON DELETE CASCADE,
  CONSTRAINT name_must_not_be_blank CHECK (length(trim(name)) > 0),
  CONSTRAINT formula_must_not_be_blank CHECK (length(trim(price_formula)) > 0)
);

CREATE VIEW pricing_plans_with_duration AS (
  SELECT
    *,
    TSTZRANGE(
      valid_from,
      LEAD(valid_from, 1, 'infinity') OVER (
        PARTITION BY
          plan_id
        ORDER BY
          valid_from
        ROWS BETWEEN current row AND 1 following
      )
    ) AS duration
  FROM
    pricing_plans
);

CREATE VIEW pricing_plan_components_with_vat_currency_with_duration AS (
  SELECT
    pp.sequence AS pricing_plan_sequence,
    cr.sequence AS currency_rate_sequence,
    vr.sequence AS vat_rate_sequence,
    pp.plan_id AS pricing_plan_id,
    pp.metadata AS metadata,
    pp.name AS pricing_plan_name,
    ppc.name AS pricing_component_name,
    ppc.price_formula,
    ppc.apply_management_fee,
    cr.code AS currency_code,
    cr.rate AS currency_rate,
    vr.code AS vat_code,
    vr.rate AS vat_rate,
    pp.duration * cr.duration * vr.duration AS duration
  FROM
    pricing_plans_with_duration pp
  -- TODO: Verify these join conditions. We want to multiply pricing plan components * currency * vat to get one row per overlap.
  LEFT JOIN pricing_plan_components ppc ON
    ppc.pricing_plan_sequence = pp.sequence
  LEFT JOIN currency_rates_with_duration cr ON
    cr.code = ppc.currency_code
    AND cr.duration && pp.duration
  LEFT JOIN vat_rates_with_duration vr ON
    vr.code = ppc.vat_code
    AND vr.duration && pp.duration
);
