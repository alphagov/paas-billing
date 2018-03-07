ALTER TABLE pricing_plans DROP CONSTRAINT IF EXISTS valid_from_start_of_month;
ALTER TABLE pricing_plans ADD CONSTRAINT valid_from_start_of_month CHECK (
  (extract (day from valid_from)) = 1 AND
  (extract (hour from valid_from)) = 0 AND
  (extract (minute from valid_from)) = 0 AND
  (extract (second from valid_from)) = 0
);
