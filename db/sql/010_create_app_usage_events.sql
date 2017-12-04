CREATE TABLE IF NOT EXISTS app_usage_events (
	id SERIAL,
	guid CHAR(36) UNIQUE,
	created_at TIMESTAMP,
	raw_message JSONB
);
CREATE INDEX IF NOT EXISTS idx_app_usage_id ON app_usage_events (id);
CREATE INDEX IF NOT EXISTS idx_app_usage_state ON app_usage_events ( (raw_message->>'state') );
