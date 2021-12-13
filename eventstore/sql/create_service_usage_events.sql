CREATE TABLE IF NOT EXISTS service_usage_events (
	id SERIAL,
	guid uuid UNIQUE NOT NULL,
	created_at timestamptz NOT NULL,
	raw_message JSONB NOT NULL,
	processed BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS service_usage_id_idx ON service_usage_events (id);
CREATE INDEX IF NOT EXISTS service_usage_state_idx ON service_usage_events ( (raw_message->>'state') );
CREATE INDEX IF NOT EXISTS service_usage_type_idx ON service_usage_events ( (raw_message->>'service_instance_type') );
CREATE INDEX IF NOT EXISTS service_usage_space_name_idx ON service_usage_events ( (raw_message->>'space_name') text_pattern_ops);
CREATE INDEX IF NOT EXISTS service_usage_processed_idx ON service_usage_events (created_at, processed);
