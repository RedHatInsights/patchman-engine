ALTER TABLE advisory_metadata DROP COLUMN synced;
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_metadata TO listener;
ALTER TABLE package DROP COLUMN synced;
