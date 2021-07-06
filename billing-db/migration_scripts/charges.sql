-- Only needs to be run once

CREATE TEMPORARY TABLE billing_formulae_conversion
AS
SELECT DISTINCT (TRIM(c.formula))::TEXT as generic_formula,
  (TRIM(c.formula))::TEXT as original_formula,
  c.name AS component_name,
  NULL::NUMERIC as aws_price,
  CASE WHEN p.name ILIKE '%postgres%' THEN 'https://aws.amazon.com/rds/postgresql/pricing/' ELSE NULL::VARCHAR END as formula_source
FROM pricing_plans p, pricing_plan_components c
WHERE p.plan_guid = c.plan_guid
AND p.valid_from = c.valid_from;

-- This code is only for eu-west-1
UPDATE billing_formulae_conversion SET generic_formula = '0', aws_price = NULL WHERE generic_formula = '0';
UPDATE billing_formulae_conversion SET generic_formula = '((1936.57/(48*1024))/30/24) * memory_in_mb * ceil(time_in_seconds / 3600)', aws_price = NULL WHERE generic_formula = '((1936.57/(48*1024))/30/24) * $memory_in_mb * ceil($time_in_seconds / 3600)';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.00685 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.00685';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.018 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.018';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.02 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.02';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.036 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.036';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.039 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.039';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.072 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.072';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.078 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.078';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.130 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.130';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.178 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.178';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.189 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.189';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.193 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.193';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.197 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.197';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.201 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.201';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.378 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.378';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.386 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.386';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.394 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.394';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.402 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.402';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.548 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.548';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.756 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.756';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.772 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.772';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.788 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.788';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 0.806 WHERE original_formula = 'ceil($time_in_seconds/3600) * 0.806';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 1.096 WHERE original_formula = 'ceil($time_in_seconds/3600) * 1.096';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 1.512 WHERE original_formula = 'ceil($time_in_seconds/3600) * 1.512';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 1.544 WHERE original_formula = 'ceil($time_in_seconds/3600) * 1.544';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 1.545 WHERE original_formula = 'ceil($time_in_seconds/3600) * 1.545';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 1.576 WHERE original_formula = 'ceil($time_in_seconds/3600) * 1.576';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 1.612 WHERE original_formula = 'ceil($time_in_seconds/3600) * 1.612';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 2.192 WHERE original_formula = 'ceil($time_in_seconds/3600) * 2.192';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 3.024 WHERE original_formula = 'ceil($time_in_seconds/3600) * 3.024';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 3.09 WHERE original_formula = 'ceil($time_in_seconds/3600) * 3.09';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 3.152 WHERE original_formula = 'ceil($time_in_seconds/3600) * 3.152';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 3.224 WHERE original_formula = 'ceil($time_in_seconds/3600) * 3.224';
UPDATE billing_formulae_conversion SET generic_formula = 'ceil(time_in_seconds/3600) * aws_price', aws_price = 4.384 WHERE original_formula = 'ceil($time_in_seconds/3600) * 4.384';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price', aws_price = 0.018 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * 0.018';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price', aws_price = 0.036 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * 0.036';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price', aws_price = 0.073 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * 0.073';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price', aws_price = 0.172 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * 0.172';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price', aws_price = 0.343 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * 0.343';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds/3600) * aws_price', aws_price = 0.686 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds/3600) * 0.686';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * ceil(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01', aws_price = 1 WHERE original_formula = '$number_of_nodes * ceil($time_in_seconds / 3600) * ($memory_in_mb/1024.0) * 0.01';
UPDATE billing_formulae_conversion SET generic_formula = '(number_of_nodes * ceil(time_in_seconds / 3600) * (memory_in_mb/1024.0) * 0.01) * aws_price', aws_price = 0.40 WHERE original_formula = '($number_of_nodes * ceil($time_in_seconds / 3600) * ($memory_in_mb/1024.0) * 0.01) * 0.40';
UPDATE billing_formulae_conversion SET generic_formula = 'number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)', aws_price = 1 WHERE original_formula = '$number_of_nodes * $time_in_seconds * ($memory_in_mb/1024.0) * (0.01 / 3600)';
UPDATE billing_formulae_conversion SET generic_formula = '(number_of_nodes * time_in_seconds * (memory_in_mb/1024.0) * (0.01 / 3600)) * aws_price', aws_price = 0.40 WHERE original_formula = '($number_of_nodes * $time_in_seconds * ($memory_in_mb/1024.0) * (0.01 / 3600)) * 0.40';

UPDATE billing_formulae_conversion SET generic_formula = '(storage_in_mb/1024) * ceil(time_in_seconds/2678401) * aws_price', aws_price = 0.127 WHERE original_formula = '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.127';
UPDATE billing_formulae_conversion SET generic_formula = '(storage_in_mb/1024) * ceil(time_in_seconds/2678401) * aws_price', aws_price = 0.253 WHERE original_formula = '($storage_in_mb/1024) * ceil($time_in_seconds/2678401) * 0.253';

SELECT DISTINCT generic_formula FROM billing_formulae_conversion;
SELECT DISTINCT * FROM billing_formulae_conversion;

SELECT * FROM billing_formulae_conversion WHERE generic_formula IS NULL;
-- should be none

INSERT INTO billing_formulae
(
  formula_name,
  generic_formula,
  formula_source
)
SELECT DISTINCT generic_formula || CASE WHEN component_name IS NOT NULL THEN ' (' || component_name || ') ' ELSE '' END || CASE WHEN formula_source IS NOT NULL THEN ' from ' || formula_source ELSE '' END, -- formula_name
  generic_formula,
  formula_source
FROM billing_formulae_conversion
ORDER BY generic_formula;

INSERT INTO charges
(
  plan_guid,
  plan_name, 
  valid_from,
  valid_to,
  storage_in_mb, 
  memory_in_mb, 
  number_of_nodes,
  aws_price, -- The last bit in 'ceil($time_in_seconds/3600) * 0.00685'
  component_name,
  formula_name,
  vat_code,
  currency_code
)
SELECT DISTINCT p.plan_guid,
                p.name, 
                p.valid_from AS "valid_from", 
                '9999-12-31'::TIMESTAMPTZ AS "valid_to", 
                p.storage_in_mb, 
                p.memory_in_mb, 
                p.number_of_nodes, 
                t.aws_price,
                c.name, -- storage, etc.
                f.formula_name,
                c.vat_code::VARCHAR,
                c.currency_code::VARCHAR(3)
FROM pricing_plans p
INNER JOIN pricing_plan_components c
ON p.plan_guid = c.plan_guid
AND p.valid_from = c.valid_from
LEFT OUTER JOIN billing_formulae_conversion t
ON t.original_formula = c.formula
INNER JOIN billing_formulae f
ON t.generic_formula = f.generic_formula
ORDER BY p.name;

SELECT DISTINCT plan_guid,
  plan_name, 
  valid_from,
  valid_to,
  storage_in_mb, 
  memory_in_mb, 
  number_of_nodes,
  aws_price, -- The last bit in 'ceil($time_in_seconds/3600) * 0.00685'
  component_name,
  vat_code,
  currency_code
FROM charges;

-- Update the valid_to date to be correct
do $$
declare _rowcount integer := 1;
DECLARE _counter integer := 0;
BEGIN
   WHILE _rowcount > 0 AND _counter < 100 LOOP
      WITH updated_entries AS (
        UPDATE charges SET valid_to = (
          SELECT MIN(valid_from)
          FROM charges c2
          WHERE c2.plan_guid = charges.plan_guid
          AND c2.component_name = charges.component_name
          AND c2.valid_from > charges.valid_from
          AND c2.valid_to > charges.valid_from
        )
        WHERE valid_to = '9999-12-31'
        AND (SELECT MIN(valid_from)
          FROM charges c2
          WHERE c2.plan_guid = charges.plan_guid
          AND c2.component_name = charges.component_name
          AND c2.valid_from > charges.valid_from
          AND c2.valid_to > charges.valid_from) IS NOT NULL
        RETURNING *
      )
      SELECT COUNT(*) INTO _rowcount FROM updated_entries;

      RAISE NOTICE '_rowcount %', _rowcount;

      _counter := _counter + 1;
   END LOOP;
END$$;

ANALYZE charges;
