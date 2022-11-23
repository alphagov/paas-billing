-- **do not alter - add new migrations instead**

BEGIN;


--
-- add basic last_agg aggregation function, which simply returns the value most
-- recently presented to it. powerful when used with aggregation FILTER clause.
--


CREATE FUNCTION last_agg_sfunc (state anyelement, newval anyelement) RETURNS anyelement AS $$
	SELECT newval;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE AGGREGATE last_agg (anyelement) (
	SFUNC = last_agg_sfunc
	, STYPE = anyelement
);


COMMIT;
