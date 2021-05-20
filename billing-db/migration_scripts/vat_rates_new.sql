INSERT INTO vat_rates_new (vat_code, valid_from, valid_to, vat_rate) SELECT 'Standard', '1970-01-01', '9999-12-31', 0.2;
INSERT INTO vat_rates_new (vat_code, valid_from, valid_to, vat_rate) SELECT 'Reduced', '1970-01-01', '9999-12-31', 0.05;
INSERT INTO vat_rates_new (vat_code, valid_from, valid_to, vat_rate) SELECT 'Zero', '1970-01-01', '9999-12-31', 0;