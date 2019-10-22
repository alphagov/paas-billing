CREATE TABLE costs (
  cost_guid uuid NOT NULL,
  day date NOT NULL,
  duration tstzrange NOT NULL,
  metadata JSONB NOT NULL,

  currency_code currency_code NOT NULL,
  base_cost numeric NOT NULL,

  resource_guid uuid NOT NULL,
  resource_name text NOT NULL,
  resource_type text NOT NULL,
  org_guid uuid NOT NULL,
  org_name text NOT NULL,
  space_guid uuid NOT NULL,
  space_name text NOT NULL,

  PRIMARY KEY (cost_guid, day),
  CONSTRAINT no_empty_duration CHECK (not isempty(duration))
);

CREATE TABLE charges (
  cost_guid uuid NOT NULL,
  day date NOT NULL,

  currency_code currency_code NOT NULL,
  base_cost numeric NOT NULL,
  management_fee numeric NOT NULL,
  management_fee numeric NOT NULL,
  vat numeric NOT NULL,
  total numeric NOT NULL,

  PRIMARY KEY (cost_guid, day)
);
