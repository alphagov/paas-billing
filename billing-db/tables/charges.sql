CREATE TABLE IF NOT EXISTS charges
(
  plan_guid UUID NOT NULL,
  plan_name TEXT NOT NULL,
  valid_from TIMESTAMPTZ NOT NULL, 
  valid_to TIMESTAMPTZ NOT NULL,
  storage_in_mb NUMERIC NOT NULL,
  memory_in_mb NUMERIC NOT NULL,
  number_of_nodes INT NOT NULL,
  aws_price NUMERIC NULL, -- e.g. price per hour obtained from prices section of AWS website
  component_name TEXT NOT NULL,
  generic_formula TEXT NOT NULL,
  vat_code VARCHAR NULL,
  currency_code CHAR(3) NULL, -- ISO currency code
  formula_source TEXT NULL -- Web page in AWS and/or other information showing where this formula came from

  -- PRIMARY KEY (plan_guid, valid_to) TODO: Uncomment once have added data to charges table. This line is commented because may want to amend the valid_to date after initial population of the charges table, in which case this constraint will fail.
);
CREATE INDEX CONCURRENTLY charges_i1 ON charges (plan_guid, valid_from, valid_to);
CREATE INDEX CONCURRENTLY charges_i2 ON charges (valid_from, valid_to);
