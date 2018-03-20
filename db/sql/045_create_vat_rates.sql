CREATE TABLE IF NOT EXISTS vat_rates (
	id SERIAL PRIMARY KEY,
	name VARCHAR(32),
	rate NUMERIC,

	CHECK (length(trim(name)) > 0),
	CHECK (rate >= 0)
);

INSERT INTO
	vat_rates (id, name, rate)
VALUES
	(1, 'Standard', 0.2),
	(2, 'Zero rate', 0)
ON CONFLICT (id) DO NOTHING;

-- reserve 100 known ids for us to use directly in migrations/tests
ALTER SEQUENCE IF EXISTS vat_rates_id_seq MINVALUE 101 START 101 RESTART 101;
