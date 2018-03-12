CREATE TABLE IF NOT EXISTS vat_rates (
	id SERIAL PRIMARY KEY,
	name VARCHAR(32),
	rate NUMERIC,

	CHECK (length(trim(name)) > 0),
	CHECK (rate >= 0)
);

-- FIXME: Remove after deployed in prod and duplicates have been removed
DELETE FROM vat_rates a USING vat_rates b
WHERE a.id > b.id
AND a.name = b.name
AND a.rate = b.rate;

DO $$
BEGIN
    IF nextval('vat_rates_id_seq') = 1 THEN
        INSERT INTO vat_rates (name, rate) VALUES
        ('Standard', 0.2),
        ('Zero rate', 0);
    END IF;
END;
$$;
