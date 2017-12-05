-- whitelist only
CREATE OR REPLACE FUNCTION validate_formula() RETURNS trigger AS $$
DECLARE
	invalid_formula text;
	illegal_token text;
	dummy_price numeric;
BEGIN
	invalid_formula := lower(NEW.formula);
	invalid_formula := (select regexp_replace(invalid_formula, '::(integer|bigint|numeric)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '([0-9]+)?\.([0-9]+)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '([0-9]+)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\$memory_in_mb', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\$time_in_seconds', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\(|\)', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\*', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\-', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\+', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\/', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\^', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '\s+', '#', 'g'));
	invalid_formula := (select regexp_replace(invalid_formula, '#+', '', 'g'));
	IF (invalid_formula != '') THEN
		illegal_token := (select * from regexp_split_to_table(invalid_formula, '\s+') limit 1);
		RAISE EXCEPTION 'illegal token in formula: %', illegal_token;
	END IF;
	-- attempt to use the formula to ensure it works with common edge case inputs
	dummy_price := (select eval_formula(0,tstzrange(now(), now()), NEW.formula));
	dummy_price := (select eval_formula(1,tstzrange(now(), now() + '1 second'), NEW.formula));
	dummy_price := (select eval_formula(null,null, NEW.formula));
	RETURN NEW;
END;
$$ language plpgsql;

-- setup trigger
DROP TRIGGER IF EXISTS tgr_validate_formula ON pricing_plans;
CREATE TRIGGER tgr_validate_formula BEFORE INSERT OR UPDATE ON pricing_plans
FOR EACH ROW EXECUTE PROCEDURE validate_formula();

