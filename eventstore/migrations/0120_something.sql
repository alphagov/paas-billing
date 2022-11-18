-- **do not alter - add new migrations instead**

BEGIN;

--
-- defining these parts of the main create_events query as functions should
-- allow us to ensure we're indexing and querying on the same expressions,
-- and if we're careful postgres is able to inline them into the query.
--


CREATE FUNCTION app_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (raw_message->>'state')::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION service_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (case
		when (raw_message->>'state') = 'CREATED' then 'STARTED'
		when (raw_message->>'state') = 'DELETED' then 'STOPPED'
		when (raw_message->>'state') = 'UPDATED' then 'STARTED'
	end)::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION task_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (case
		when (raw_message->>'state') = 'TASK_STARTED' then 'STARTED'
		when (raw_message->>'state') = 'TASK_STOPPED' then 'STOPPED'
	end)::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION staging_event_resource_state(raw_message jsonb) returns resource_state AS $$
	SELECT (case
		when (raw_message->>'state') = 'CREATED' then 'STARTED'
		when (raw_message->>'state') = 'DELETED' then 'STOPPED'
		when (raw_message->>'state') = 'UPDATED' then 'STARTED'
	end)::resource_state;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;


CREATE FUNCTION app_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT (raw_message->>'state' = 'STARTED' or raw_message->>'state' = 'STOPPED')
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION service_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT raw_message->>'service_instance_type' = 'managed_service_instance'
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION task_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT (raw_message->>'state' = 'TASK_STARTED' or raw_message->>'state' = 'TASK_STOPPED')
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION staging_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT (raw_message->>'state' = 'STAGING_STARTED' or raw_message->>'state' = 'STAGING_STOPPED')
		and raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-'; -- FIXME: this is open to abuse
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

-- indexing the resource_state will at least give us statistics on that value
-- that should be usable even if the index isn't used for filtering. we're
-- mostly interested in the partiality of the index anyway - indexed value is
-- less important

CREATE INDEX app_usage_events_app_resource_state_part_idx ON app_usage_events ((app_event_resource_state(raw_message))) WHERE app_event_filter(raw_message);
CREATE INDEX service_usage_events_service_resource_state_part_idx ON service_usage_events ((service_event_resource_state(raw_message))) WHERE service_event_filter(raw_message);
CREATE INDEX app_usage_events_task_resource_state_part_idx ON app_usage_events ((task_event_resource_state(raw_message))) WHERE task_event_filter(raw_message);
CREATE INDEX app_usage_events_staging_resource_state_part_idx ON app_usage_events ((staging_event_resource_state(raw_message))) WHERE staging_event_filter(raw_message);


-- compose service events are a little fancier because they do a further outer
-- join before windowing. we can prepare indexes for both sides of this join.


CREATE FUNCTION compose_service_event_filter(raw_message jsonb) returns BOOLEAN AS $$
	SELECT raw_message->>'space_name' !~ '^(SMOKE|ACC|CATS|PERF)-';
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION service_instance_guid_if_created(raw_message jsonb) returns uuid AS $$
	SELECT (case
		when (raw_message->>'state') = 'CREATED' then raw_message->>'service_instance_guid'
		-- else NULL, which won't match against anything when used in a join
	end)::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE INDEX service_usage_events_svc_inst_guid_if_crtd_part_idx ON service_usage_events ((service_instance_guid_if_created(raw_message))) WHERE compose_service_event_filter(raw_message);

CREATE FUNCTION uuid_from_data_deployment(raw_message jsonb) returns uuid AS $$
	SELECT substring(
		raw_message->'data'->>'deployment'
		from '[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12}$'
	)::uuid;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE INDEX compose_audit_events_uuid_frm_data_dpmt_idx ON compose_audit_events ((uuid_from_data_deployment(raw_message)));

COMMIT;
