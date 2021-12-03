CREATE TABLE IF NOT EXISTS sources
(
    source CHAR(3) NOT NULL,
    source_description TEXT NOT NULL,
    active BOOLEAN NOT NULL -- Is this feed active? If true then this source (e.g. Cloud Foundry) should contribute to tenants' bills
);
-- INSERT INTO sources (source, source_description, active) SELECT 'cf', 'Cloud Foundry events', TRUE;