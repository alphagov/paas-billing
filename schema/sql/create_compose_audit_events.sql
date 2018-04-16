
CREATE TABLE IF NOT EXISTS compose_audit_events (
	id SERIAL,
	event_id CHAR(24) UNIQUE,
	created_at TIMESTAMPTZ,
	raw_message JSONB
);
CREATE INDEX IF NOT EXISTS compose_audit_events_id ON compose_audit_events (id);

ALTER TABLE compose_audit_events ALTER COLUMN event_id SET NOT NULL;
ALTER TABLE compose_audit_events ALTER COLUMN created_at SET NOT NULL;
ALTER TABLE compose_audit_events ALTER COLUMN raw_message SET NOT NULL;
ALTER TABLE compose_audit_events ALTER COLUMN event_id TYPE text USING trim(event_id);

DO $$ BEGIN
	ALTER TABLE compose_audit_events ADD CONSTRAINT event_id_not_blank CHECK (length(event_id) > 0);
EXCEPTION
	WHEN duplicate_object THEN RAISE NOTICE 'constraint already exists';
END; $$;

DO $$ BEGIN
	ALTER TABLE compose_audit_events ADD CONSTRAINT created_at_not_zero_value CHECK (created_at > 'epoch'::timestamptz);
EXCEPTION
	WHEN duplicate_object THEN RAISE NOTICE 'constraint already exists';
END; $$;

