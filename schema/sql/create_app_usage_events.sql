
CREATE TABLE IF NOT EXISTS app_usage_events (
	id SERIAL, -- this should probably be called "sequence" it's not really an id
	guid CHAR(36) UNIQUE,
	created_at TIMESTAMP,
	raw_message JSONB
);

ALTER TABLE app_usage_events ALTER COLUMN guid SET NOT NULL;
ALTER TABLE app_usage_events ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE app_usage_events ALTER COLUMN raw_message SET NOT NULL;
CREATE INDEX IF NOT EXISTS app_usage_id_idx ON app_usage_events (id);
CREATE INDEX IF NOT EXISTS app_usage_state_idx ON app_usage_events ( (raw_message->>'state') );
CREATE INDEX IF NOT EXISTS app_usage_space_name_idx ON app_usage_events ( (raw_message->>'space_name') text_pattern_ops);

ALTER TABLE app_usage_events ALTER COLUMN created_at TYPE timestamptz USING created_at AT TIME ZONE 'UTC';
ALTER TABLE app_usage_events ALTER COLUMN guid TYPE uuid USING guid::uuid;
