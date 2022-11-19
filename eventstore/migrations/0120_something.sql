-- **do not alter - add new migrations instead**

BEGIN;






-- indexing the resource_state will at least give us statistics on that value
-- that should be usable even if the index isn't used for filtering. we're
-- mostly interested in the partiality of the index anyway - indexed value is
-- less important

CREATE INDEX service_usage_events_service_resource_state_part_idx ON service_usage_events ((service_event_resource_state(raw_message))) WHERE service_event_filter(raw_message);


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
