BEGIN;

-- because we can only cluster by *one* index/ordering, it's best to choose one
-- that is helpful for *all* event types that are pulled from app_usage_events.
-- no single existing index fulfils that, so temporarily create a custom one.
-- should broadly match ordering used by *_event_ranges_partial_idx indexes
-- and corresponding window function in create_events. doesn't have to be an
-- exact expression match as we just have to broadly statistically correlate
-- with the order of these to be helpful.
CREATE INDEX app_usage_events_cluster_idx ON app_usage_events(
	(CASE
		WHEN app_event_filter(raw_message) THEN app_event_resource_guid
		WHEN task_event_filter(raw_message) THEN task_event_resource_guid
		WHEN staging_event_filter(raw_message) THEN staging_event_resource_guid
	END) ASC NULLS FIRST,  -- rows passing no filter put at beginning of table
	created_at DESC,
	id DESC
);

CLUSTER app_usage_events USING app_usage_events_cluster_idx;

-- index not useful when we're not running a CLUSTER operation & adds expense
-- to INSERTs
DROP INDEX app_usage_events_cluster_idx;

ANALYZE;


COMMIT;
