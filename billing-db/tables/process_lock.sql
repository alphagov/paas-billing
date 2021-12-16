CREATE TABLE IF NOT EXISTS process_lock
(
    process_name TEXT PRIMARY KEY NOT NULL, -- Only have one entry per process. When a process is running, it should lock this row thus preventing another instance of the process from updating this row. This ensures only one instance of the process is running at a given time.
    process_running BOOLEAN DEFAULT FALSE
);

TRUNCATE TABLE process_lock;

-- Data included here to make code simpler.
INSERT INTO process_lock (process_name) VALUES ('Cloud Foundry event collector');
