CREATE TABLE IF NOT EXISTS resources
(
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ NOT NULL,
    -- Next two columns are purely for optimisation. Contain entries such as "Jan 2020".
    valid_from_month VARCHAR NOT NULL, -- e.g. TO_CHAR(valid_from, 'Month') || ' ' || DATE_PART('year', valid_from)
    valid_to_month VARCHAR NOT NULL, -- e.g. TO_CHAR(valid_to, 'Month') || ' ' || DATE_PART('year', valid_to)
	resource_guid UUID NOT NULL,
	resource_name TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	org_guid UUID NOT NULL,
	org_name TEXT NOT NULL,
	space_guid UUID NOT NULL,
	space_name TEXT NOT NULL,
	plan_name TEXT NOT NULL,
	plan_guid UUID NOT NULL,
	-- source VARCHAR NULL, -- This can be added later if needed. Source system from which this was last updated from, e.g. Cloudfoundry
    cf_event_guid UUID NULL -- Cloudfoundry event_guid that was last used to update this row. May be handy for audit purposes.
);

CREATE INDEX CONCURRENTLY resources_i1 ON resources (valid_from, valid_to);
CREATE INDEX CONCURRENTLY resources_i2 ON resources (org_name, valid_from, valid_to);
