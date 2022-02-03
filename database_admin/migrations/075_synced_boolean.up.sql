ALTER TABLE advisory_metadata ADD COLUMN synced BOOLEAN NOT NULL DEFAULT false;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO vmaas_sync;
REVOKE ALL PRIVILEGES ON advisory_metadata FROM listener;
GRANT SELECT ON advisory_metadata TO listener;
GRANT SELECT ON advisory_metadata TO manager;

ALTER TABLE package ADD COLUMN synced BOOLEAN NOT NULL DEFAULT false;
GRANT SELECT, INSERT, UPDATE, DELETE ON package TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON package TO vmaas_sync;
GRANT SELECT ON package TO manager;
