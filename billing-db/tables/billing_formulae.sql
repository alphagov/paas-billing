CREATE TABLE IF NOT EXISTS billing_formulae
(
  formula_name TEXT NOT NULL,
  generic_formula TEXT NOT NULL,
  formula_source TEXT NULL, -- Web page in AWS and/or other information showing where this formula came from

  PRIMARY KEY (formula_name)
);
