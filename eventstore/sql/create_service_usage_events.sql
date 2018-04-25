
CREATE TABLE IF NOT EXISTS service_usage_events (
	id SERIAL,
	guid CHAR(36) UNIQUE,
	created_at TIMESTAMP,
	raw_message JSONB
);

ALTER TABLE service_usage_events ALTER COLUMN guid SET NOT NULL;
ALTER TABLE service_usage_events ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE service_usage_events ALTER COLUMN raw_message SET NOT NULL;
CREATE INDEX IF NOT EXISTS service_usage_id_idx ON service_usage_events (id);
CREATE INDEX IF NOT EXISTS service_usage_state_idx ON service_usage_events ( (raw_message->>'state') );
CREATE INDEX IF NOT EXISTS service_usage_type_idx ON service_usage_events ( (raw_message->>'service_instance_type') );
CREATE INDEX IF NOT EXISTS service_usage_space_name_idx ON service_usage_events ( (raw_message->>'space_name') text_pattern_ops);

ALTER TABLE service_usage_events ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE service_usage_events ALTER COLUMN guid TYPE uuid USING guid::uuid;

