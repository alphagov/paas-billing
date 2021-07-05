-- New billing
CREATE TABLE new_billing AS SELECT * FROM get_tenant_bill('', '2020-12-01', '2021-01-01') WHERE 1=2;
CREATE TEMPORARY TABLE calls AS SELECT DISTINCT 'INSERT INTO new_billing SELECT * FROM get_tenant_bill(''' || name || ''', ''2020-12-01'', ''2021-01-01''); ' FROM orgs;
\copy calls to calls.sql
\i calls.sql

-- Current billing
CREATE TABLE old_billing AS SELECT * FROM new_billing WHERE 1=2;
INSERT INTO old_billing (org_name, org_guid, plan_guid, plan_name, space_name, resource_type, resource_name, component_name, charge_usd_exc_vat, charge_gbp_exc_vat, charge_gbp_inc_vat) SELECT cbe.org_name, cbe.org_guid, cbe.plan_guid, 'Not populated for simplicity due to valid_from,to when joining to pricing_plans', cbe.space_name, cbe.resource_type, cbe.resource_name, cbe.price->'details'->0->>'name' AS component_name, 0, (cbe.price->'details'->0->>'ex_vat')::NUMERIC AS charge_gbp_exc_vat, (cbe.price->'details'->0->>'inc_vat')::NUMERIC AS charge_gbp_inc_vat FROM consolidated_billable_events cbe WHERE LOWER(consolidated_range) = '2020-12-01' AND UPPER(consolidated_range) = '2021-01-01' AND jsonb_array_length(cbe.price->'details') >= 1;
INSERT INTO old_billing (org_name, org_guid, plan_guid, plan_name, space_name, resource_type, resource_name, component_name, charge_usd_exc_vat, charge_gbp_exc_vat, charge_gbp_inc_vat) SELECT cbe.org_name, cbe.org_guid, cbe.plan_guid, 'Not populated for simplicity due to valid_from,to when joining to pricing_plans', cbe.space_name, cbe.resource_type, cbe.resource_name, cbe.price->'details'->1->>'name' AS component_name, 0, (cbe.price->'details'->1->>'ex_vat')::NUMERIC AS charge_gbp_exc_vat, (cbe.price->'details'->1->>'inc_vat')::NUMERIC AS charge_gbp_inc_vat FROM consolidated_billable_events cbe WHERE LOWER(consolidated_range) = '2020-12-01' AND UPPER(consolidated_range) = '2021-01-01' AND jsonb_array_length(cbe.price->'details') >= 2;
INSERT INTO old_billing (org_name, org_guid, plan_guid, plan_name, space_name, resource_type, resource_name, component_name, charge_usd_exc_vat, charge_gbp_exc_vat, charge_gbp_inc_vat) SELECT cbe.org_name, cbe.org_guid, cbe.plan_guid, 'Not populated for simplicity due to valid_from,to when joining to pricing_plans', cbe.space_name, cbe.resource_type, cbe.resource_name, cbe.price->'details'->2->>'name' AS component_name, 0, (cbe.price->'details'->2->>'ex_vat')::NUMERIC AS charge_gbp_exc_vat, (cbe.price->'details'->2->>'inc_vat')::NUMERIC AS charge_gbp_inc_vat FROM consolidated_billable_events cbe WHERE LOWER(consolidated_range) = '2020-12-01' AND UPPER(consolidated_range) = '2021-01-01' AND jsonb_array_length(cbe.price->'details') >= 3;

CREATE TABLE old_billing_totals_details AS SELECT org_guid, plan_guid, space_name, resource_name, component_name, NULL::NUMERIC AS total_charge_gbp_exc_vat, NULL::NUMERIC AS total_charge_gbp_exc_vat_round FROM old_billing WHERE 1=2;
CREATE TABLE new_billing_totals_details AS SELECT org_guid, plan_guid, space_name, resource_name, component_name, NULL::NUMERIC AS total_charge_gbp_exc_vat, NULL::NUMERIC AS total_charge_gbp_exc_vat_round FROM old_billing WHERE 1=2;

INSERT INTO old_billing_totals_details (org_guid, plan_guid, space_name, resource_name, component_name, total_charge_gbp_exc_vat, total_charge_gbp_exc_vat_round) SELECT org_guid, plan_guid, space_name, resource_name, component_name, SUM(charge_gbp_exc_vat), ROUND(SUM(charge_gbp_exc_vat), 3) FROM old_billing GROUP BY org_guid, plan_guid, space_name, resource_name, component_name;
INSERT INTO new_billing_totals_details (org_guid, plan_guid, space_name, resource_name, component_name, total_charge_gbp_exc_vat, total_charge_gbp_exc_vat_round) SELECT org_guid, plan_guid, space_name, resource_name, component_name, SUM(charge_gbp_exc_vat), ROUND(SUM(charge_gbp_exc_vat), 3) FROM new_billing GROUP BY org_guid, plan_guid, space_name, resource_name, component_name;

\copy old_billing_totals_details to old_billing_totals_details.dat
\copy new_billing_totals_details to new_billing_totals_details.dat

CREATE INDEX old_billing_idx1 ON old_billing_totals_details (org_guid, plan_guid, space_name, resource_name, component_name);
CREATE INDEX new_billing_idx1 ON new_billing_totals_details (org_guid, plan_guid, space_name, resource_name, component_name);

SELECT o.org_guid, o.plan_guid, o.space_name, o.resource_name, o.component_name, o.total_charge_gbp_exc_vat_round AS old_charge, n.total_charge_gbp_exc_vat_round AS new_charge FROM old_billing_totals_details o, new_billing_totals_details n WHERE o.org_guid = n.org_guid AND o.plan_guid = n.plan_guid and o.space_name = n.space_name AND o.resource_name = n.resource_name AND o.component_name = n.component_name AND o.total_charge_gbp_exc_vat_round != n.total_charge_gbp_exc_vat_round;
-- Should be none

-- Now reconcile
CREATE TABLE old_not_new AS SELECT * FROM old_billing_totals_details;

CREATE TABLE new_not_old AS SELECT * FROM new_billing_totals_details;

DELETE FROM old_not_new USING new_billing_totals_details n WHERE old_not_new.org_guid = n.org_guid AND old_not_new.plan_guid = n.plan_guid and old_not_new.space_name = n.space_name AND old_not_new.resource_name = n.resource_name AND old_not_new.component_name = n.component_name;

DELETE FROM new_not_old USING old_billing_totals_details n WHERE new_not_old.org_guid = n.org_guid AND new_not_old.plan_guid = n.plan_guid and new_not_old.space_name = n.space_name AND new_not_old.resource_name = n.resource_name AND new_not_old.component_name = n.component_name;

SELECT COUNT(*) FROM old_not_new;

SELECT COUNT(*) FROM new_not_old;
