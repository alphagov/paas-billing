CREATE TABLE IF NOT EXISTS currency_rates(
	id serial PRIMARY KEY,
	code text NOT NULL,
	valid_from timestamptz NOT NULL,
	rate decimal NOT NULL,

	CHECK (code = ANY ('{GBP, USD, EUR}'::text[])),
	CHECK (rate >= 0)
);

INSERT INTO currency_rates (code, valid_from, rate)
SELECT 'GBP', '2000-01-01T00:00:00', 1
WHERE NOT EXISTS (
    SELECT code, valid_from, rate
    FROM currency_rates
    WHERE code = 'GBP' AND valid_from = '2000-01-01T00:00:00' AND rate = 1
);
