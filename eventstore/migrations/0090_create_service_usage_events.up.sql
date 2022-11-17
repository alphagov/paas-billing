-- **do not alter - add new migrations instead**

-- "migration" written before we had proper migration handling, hence the
-- various attempts at mitigating previously existing objects

BEGIN;

CREATE TABLE IF NOT EXISTS service_usage_events (
	id SERIAL,
	guid uuid UNIQUE NOT NULL,
	created_at timestamptz NOT NULL,
	raw_message JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS service_usage_id_idx ON service_usage_events (id);
CREATE INDEX IF NOT EXISTS service_usage_state_idx ON service_usage_events ( (raw_message->>'state') );
CREATE INDEX IF NOT EXISTS service_usage_type_idx ON service_usage_events ( (raw_message->>'service_instance_type') );
CREATE INDEX IF NOT EXISTS service_usage_space_name_idx ON service_usage_events ( (raw_message->>'space_name') text_pattern_ops);
ALTER TABLE service_usage_events ADD COLUMN IF NOT EXISTS processed boolean DEFAULT false;

COMMIT;
