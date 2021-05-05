-- This will eventually be renamed as vat_rates.
CREATE TABLE vat_rates_new
(
  vat_code VARCHAR, -- e.g. Standard, Reduced or Zero
  valid_from TIMESTAMP,
  valid_to TIMESTAMP,
  vat_rate NUMERIC,

  PRIMARY KEY (vat_code, valid_to)
);
