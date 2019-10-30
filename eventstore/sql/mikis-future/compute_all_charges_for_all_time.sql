DO $$
DECLARE
  run_from DATE := ( SELECT MIN(created_at)::date FROM cf_usage_events );
  run_to DATE := ( SELECT MAX(created_at)::date FROM cf_usage_events );
  run_date DATE;
BEGIN
  RAISE INFO '[%] --- Emptying cf_usage_periods ---', clock_timestamp();
  DELETE FROM cf_usage_periods;
  RAISE INFO '[%] --- Starting runs from % to % ---', clock_timestamp(), run_from, run_to;
  FOR run_date IN SELECT * FROM generate_series(run_from, run_to, INTERVAL '1 DAY')
  LOOP
    RAISE INFO '[%] Running % (of % to %)', clock_timestamp(), run_date, run_from, run_to;
    INSERT INTO cf_usage_periods (SELECT * FROM get_cf_usage_periods_for_date(run_date));
  END LOOP;
END;
$$ LANGUAGE plpgsql;
