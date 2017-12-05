-- function for representing dates as ISO 8601 UTC
CREATE OR REPLACE FUNCTION iso8601(t timestamptz) returns text AS $$ BEGIN
	RETURN to_char(t at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
END; $$ LANGUAGE plpgsql;
