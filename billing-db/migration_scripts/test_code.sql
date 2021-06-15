CREATE TABLE new_billing AS SELECT * FROM get_tenant_bill('<insert tenant name here>', '2020-12-01', '2021-01-01') WHERE 1=2;
SELECT DISTINCT 'INSERT INTO new_billing SELECT * FROM get_tenant_bill(''' || name || ''', ''2020-12-01'', ''2021-01-01''); ' FROM orgs WHERE name NOT LIKE 'A%' AND name NOT LIKE 'B%' AND name NOT LIKE 'C%' AND name NOT LIKE 'P%' AND name NOT LIKE 'S%';
