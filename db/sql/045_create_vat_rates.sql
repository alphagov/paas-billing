CREATE TABLE IF NOT EXISTS vat_rates (
	id SERIAL PRIMARY KEY,
	name VARCHAR(32),
	rate NUMERIC,

	CHECK (length(trim(name)) > 0),
	CHECK (rate >= 0)
);

INSERT INTO
	vat_rates (name, rate)
VALUES
	('Standard', 0.2),
	('Zero rate', 0)
ON CONFLICT (id) DO NOTHING;
