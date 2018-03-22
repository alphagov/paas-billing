CREATE TABLE IF NOT EXISTS currency_rates(
	code text NOT NULL,
	valid_from timestamptz NOT NULL,
	rate decimal NOT NULL,

	PRIMARY KEY (code, valid_from),
	CHECK (code = ANY ('{GBP, USD, EUR}'::text[])),
	CHECK (rate >= 0)
);

-- include a default GBP currency
INSERT INTO currency_rates (code, valid_from, rate)
	SELECT 'GBP', 'epoch'::timestamptz, 1 WHERE NOT EXISTS (
	    SELECT code, valid_from, rate
	    FROM currency_rates
	    WHERE code = 'GBP' AND valid_from = 'epoch'::timestamptz
	)
;
