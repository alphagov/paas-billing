-- pricing plans contain formulas for calculating prices
CREATE TABLE IF NOT EXISTS pricing_plans(
	id serial PRIMARY KEY,
	name text NOT NULL,
	valid_from timestamptz NOT NULL,
	plan_guid text NOT NULL,
	formula text NOT NULL,

	CHECK (length(trim(plan_guid)) > 0),                         -- not empty
	CHECK (length(trim(name)) > 0),                              -- not empty
	CHECK (length(trim(formula)) > 0)                            -- not empty
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pricing_plans_valid_and_plan_guid ON pricing_plans (valid_from, plan_guid);
