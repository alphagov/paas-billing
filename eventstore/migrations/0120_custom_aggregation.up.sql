CREATE FUNCTION propagate_nonnull_agg_sfunc (state anyelement, newval anyelement) RETURNS anyelement AS $$
	SELECT COALESCE(newval, state);
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE AGGREGATE propagate_nonnull_agg (ORDER BY anyelement) (
	SFUNC = propagate_nonnull_agg_sfunc
	, STYPE = anyelement
);
