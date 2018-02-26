-- pricing_plan_components contains formulas for calculating prices
CREATE TABLE IF NOT EXISTS pricing_plan_components(
	id serial PRIMARY KEY,
	pricing_plan_id integer REFERENCES pricing_plans (id) ON DELETE CASCADE,
	name text NOT NULL,
	formula text NOT NULL,

	CHECK (length(trim(name)) > 0),                              -- not empty
	CHECK (length(trim(formula)) > 0)                            -- not empty
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pricing_plan_id_name ON pricing_plan_components (pricing_plan_id, name);

-- setup formula validation trigger
DROP TRIGGER IF EXISTS tgr_ppc_validate_formula ON pricing_plan_components;
CREATE TRIGGER tgr_ppc_validate_formula BEFORE INSERT OR UPDATE ON pricing_plan_components
FOR EACH ROW EXECUTE PROCEDURE validate_formula();
