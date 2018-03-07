CREATE TABLE IF NOT EXISTS compose_audit_events (
	id SERIAL,
	event_id CHAR(24) UNIQUE,
	created_at TIMESTAMPTZ,
	raw_message JSONB
);
CREATE INDEX IF NOT EXISTS idx_compose_audit_events_id ON compose_audit_events (id);

CREATE TABLE IF NOT EXISTS compose_audit_events_cursor (
	name VARCHAR(16) UNIQUE,
	value CHAR(24)
);

-- we want to make sure there are rows in the table so we can update them with an UPDATE
INSERT INTO
	compose_audit_events_cursor (name, value)
VALUES
	('latest_event_id', NULL),
	('cursor', NULL)
ON CONFLICT (name) DO NOTHING;
