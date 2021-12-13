CREATE TABLE IF NOT EXISTS app_usage_events (
	id SERIAL, -- this should probably be called "sequence" it's not really an id
	guid uuid UNIQUE NOT NULL,
	created_at timestamptz NOT NULL,
	raw_message JSONB NOT NULL,
	processed BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS app_usage_id_idx ON app_usage_events (id);
CREATE INDEX IF NOT EXISTS app_usage_state_idx ON app_usage_events ( (raw_message->>'state') );
CREATE INDEX IF NOT EXISTS app_usage_space_name_idx ON app_usage_events ( (raw_message->>'space_name') text_pattern_ops);
CREATE INDEX IF NOT EXISTS app_usage_processed_idx ON app_usage_events (created_at, processed);

DO $$ BEGIN
	ALTER TABLE app_usage_events ADD CONSTRAINT created_at_not_zero_value CHECK (created_at > 'epoch'::timestamptz);
EXCEPTION
	WHEN duplicate_object THEN RAISE NOTICE 'constraint already exists';
END; $$;
