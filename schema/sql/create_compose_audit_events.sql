
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

CREATE TABLE IF NOT EXISTS compose_audit_events_cursor (
	name VARCHAR(16) UNIQUE,
	value CHAR(24)
);
INSERT INTO
	compose_audit_events_cursor (name, value)
VALUES
	('latest_event_id', NULL),
	('cursor', NULL)
ON CONFLICT (name) DO NOTHING;

ALTER TABLE compose_audit_events_cursor ALTER COLUMN name SET NOT NULL;
ALTER TABLE compose_audit_events_cursor ALTER COLUMN name TYPE text USING trim(name);
ALTER TABLE compose_audit_events_cursor ALTER COLUMN value TYPE text USING trim(value);
