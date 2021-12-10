CREATE TABLE process_lock
(
    process_name TEXT PRIMARY KEY NOT NULL, -- Only have one entry per process. When a process is running, it should lock this row thus preventing another instance of the process from updating this row. This ensures only one instance of the process is running at a given time.
    process_running BOOLEAN DEFAULT FALSE
);
