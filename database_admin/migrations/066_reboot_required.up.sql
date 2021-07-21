ALTER TABLE advisory_metadata ADD COLUMN reboot_required BOOLEAN NOT NULL DEFAULT false;

GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO vmaas_sync;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO listener;
GRANT SELECT ON advisory_metadata TO manager;
