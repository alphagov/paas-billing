
CREATE TEMPORARY TABLE foo AS (
		(SELECT * FROM app_event_ranges)
	union all
		(SELECT * FROM task_event_ranges)
	union all
		(SELECT * FROM staging_event_ranges)
	union all
		(SELECT * FROM service_event_ranges)
);
