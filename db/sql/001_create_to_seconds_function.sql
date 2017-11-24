
-- to_seconds extracts the number of seconds from a tstzrange
CREATE OR REPLACE FUNCTION to_seconds(tstzrange) RETURNS numeric AS $$
	select extract(epoch from (upper($1) - lower($1)))::numeric;
$$ LANGUAGE SQL IMMUTABLE STRICT;
