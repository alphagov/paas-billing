CREATE TABLE IF NOT EXISTS service_usage_events (
	id SERIAL,
	guid CHAR(36) UNIQUE,
	created_at TIMESTAMP,
	raw_message JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_usage_id ON service_usage_events (id);
CREATE INDEX IF NOT EXISTS idx_service_usage_state ON service_usage_events ( (raw_message->>'state') );
CREATE INDEX IF NOT EXISTS idx_service_usage_type ON service_usage_events ( (raw_message->>'service_instance_type') );
