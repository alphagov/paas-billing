CREATE TABLE cf_usage_events (
  guid uuid NOT NULL,
  created_at timestamptz NOT NULL,
  in_use boolean NOT NULL, -- whether this resource is being in-use (i.e., billable) as of this event

  resource_type text NOT NULL, -- "app" or "service" (or name of service?)
  resource_guid uuid NOT NULL,
  resource_name text NOT NULL,

  pricing_plan_id text NOT NULL, -- "app-run", "app-task", "app-staging", or GUID of the service plan
  pricing_metadata JSONB,

  org_guid uuid NOT NULL,
  space_guid uuid NOT NULL,
  service_guid uuid,

  PRIMARY KEY (guid),
  CONSTRAINT created_at_not_zero_value CHECK (created_at > 'epoch'::timestamptz)
);

CREATE INDEX IF NOT EXISTS cf_usage_events_created_at_idx ON cf_usage_events (created_at);
