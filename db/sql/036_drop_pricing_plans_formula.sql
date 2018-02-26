DROP TRIGGER IF EXISTS tgr_validate_formula ON pricing_plans;

ALTER TABLE pricing_plans DROP COLUMN IF EXISTS formula;
