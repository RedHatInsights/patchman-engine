ALTER TABLE advisory_metadata ADD COLUMN release_versions JSONB;

GRANT SELECT ON advisory_metadata TO evaluator;
GRANT SELECT ON advisory_metadata TO vmaas_sync;
GRANT SELECT ON advisory_metadata TO listener;
GRANT SELECT ON advisory_metadata TO manager;
