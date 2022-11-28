-- **do not alter - add new migrations instead**

BEGIN;


--
-- add aggregation function anydistinct() for text and uuid types, which
-- compares successive values for distinctness and returns true if any
-- are found to be. assuming the type has transitive equality, a false
-- value therefore implies all values are equal (or null).
--
-- postgresql polymorphism isn't quite powerful enough to avoid having
-- to declare this per-type.
--

CREATE TYPE anydistinct_stype_uuid AS (prior boolean, prev uuid);

CREATE FUNCTION anydistinct_sfunc_uuid (state anydistinct_stype_uuid, newval uuid) RETURNS anydistinct_stype_uuid AS $$
	-- prior being null indicates this being the first element, which is never considered distinct
	SELECT
		state.prior IS NOT NULL AND (state.prior OR state.prev IS DISTINCT FROM newval) AS prior,
		newval AS prev;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION anydistinct_finalfunc_uuid (state anydistinct_stype_uuid) RETURNS boolean AS $$
	SELECT state.prior;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE AGGREGATE anydistinct (uuid) (
	SFUNC = anydistinct_sfunc_uuid
	, STYPE = anydistinct_stype_uuid
	, FINALFUNC = anydistinct_finalfunc_uuid
);


CREATE TYPE anydistinct_stype_text AS (prior boolean, prev text);

CREATE FUNCTION anydistinct_sfunc_text (state anydistinct_stype_text, newval text) RETURNS anydistinct_stype_text AS $$
	-- prior being null indicates this being the first element, which is never considered distinct
	SELECT
		state.prior IS NOT NULL AND (state.prior OR state.prev IS DISTINCT FROM newval) AS prior,
		newval AS prev;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE FUNCTION anydistinct_finalfunc_text (state anydistinct_stype_text) RETURNS boolean AS $$
	SELECT state.prior;
$$ LANGUAGE SQL IMMUTABLE PARALLEL SAFE;

CREATE AGGREGATE anydistinct (text) (
	SFUNC = anydistinct_sfunc_text
	, STYPE = anydistinct_stype_text
	, FINALFUNC = anydistinct_finalfunc_text
);


COMMIT;
