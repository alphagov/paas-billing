CREATE TABLE IF NOT EXISTS charges
(
  plan_guid UUID NOT NULL,
  plan_name TEXT NOT NULL,
  valid_from TIMESTAMPTZ NOT NULL, 
  valid_to TIMESTAMPTZ NOT NULL,
  storage_in_mb NUMERIC NOT NULL,
  memory_in_mb NUMERIC NOT NULL,
  number_of_nodes INT NOT NULL,
  external_price NUMERIC NULL, -- e.g. price per hour obtained from prices section of AWS or Aiven website
  component_name TEXT NOT NULL,
  formula_name VARCHAR NULL, -- Joins to billing_formulae.formula_name
  vat_code VARCHAR NULL,
  currency_code CHAR(3) NULL, -- ISO currency code

  PRIMARY KEY (plan_guid, component_name, valid_to)
);
CREATE INDEX CONCURRENTLY charges_i1 ON charges (plan_guid, valid_from, valid_to);
CREATE INDEX CONCURRENTLY charges_i2 ON charges (valid_from, valid_to);
