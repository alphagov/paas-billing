CREATE TABLE IF NOT EXISTS consolidation_history (
  consolidated_range tstzrange NOT NULL,
  created_at timestamptz NOT NULL,

  PRIMARY KEY (consolidated_range),
  CONSTRAINT no_empty_consolidated_range CHECK (not isempty(consolidated_range)),

  CONSTRAINT range_from_start_of_month CHECK (
    (extract (day from lower(consolidated_range))) = 1 AND
    (extract (hour from lower(consolidated_range))) = 0 AND
    (extract (minute from lower(consolidated_range))) = 0 AND
    (extract (second from lower(consolidated_range))) = 0
  ),

  CONSTRAINT range_to_end_of_month CHECK (
    (extract (day from upper(consolidated_range))) = 1 AND
    (extract (hour from upper(consolidated_range))) = 0 AND
    (extract (minute from upper(consolidated_range))) = 0 AND
    (extract (second from upper(consolidated_range))) = 0
  ),

  CONSTRAINT range_exactly_one_month CHECK (
    (lower(consolidated_range) + interval '1 month') = (upper(consolidated_range))
  )
);

CREATE TABLE IF NOT EXISTS consolidated_billable_events (
  consolidated_range tstzrange REFERENCES consolidation_history(consolidated_range) NOT NULL,

  event_guid uuid NOT NULL,
  duration tstzrange NOT NULL,

  resource_guid uuid NOT NULL,
  resource_name text NOT NULL,
  resource_type text NOT NULL,

  org_guid uuid NOT NULL,
  org_name text NOT NULL,

  space_guid uuid NOT NULL,
  space_name text NOT NULL,

  plan_guid uuid NOT NULL,

  number_of_nodes integer,
  memory_in_mb integer,
  storage_in_mb integer,

  price jsonb NOT NULL,

  PRIMARY KEY (consolidated_range, event_guid, plan_guid)
);

