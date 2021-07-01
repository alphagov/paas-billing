CREATE TABLE IF NOT EXISTS resources
(
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ NOT NULL,
	resource_guid UUID NOT NULL,
	resource_name TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	org_guid UUID NOT NULL,
	org_name TEXT NOT NULL,
	space_guid UUID NOT NULL,
	space_name TEXT NOT NULL,
	plan_name TEXT NOT NULL,
	plan_guid UUID NOT NULL,
	storage_in_mb NUMERIC NULL,
	memory_in_mb NUMERIC NULL,
	number_of_nodes INT NULL,
	-- source VARCHAR NULL, -- This can be added later if needed. Source system from which this was last updated from, e.g. Cloudfoundry
	cf_event_guid UUID NULL, -- Cloudfoundry event_guid that was last used to update this row. Not used by code that calculates tenant bills (e.g. calculate_bill())
	last_updated TIMESTAMPTZ NOT NULL -- When this row was last updated
);

CREATE INDEX CONCURRENTLY resources_i1 ON resources (valid_from, valid_to);
CREATE INDEX CONCURRENTLY resources_i2 ON resources (org_name, valid_from, valid_to);
