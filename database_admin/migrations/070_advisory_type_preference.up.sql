ALTER TABLE advisory_type ADD COLUMN preference INTEGER NOT NULL DEFAULT 0;

GRANT SELECT ON advisory_type TO evaluator;
GRANT SELECT ON advisory_type TO vmaas_sync;
GRANT SELECT ON advisory_type TO listener;
GRANT SELECT ON advisory_type TO manager;

UPDATE advisory_type SET preference = 100 WHERE name = 'unknown';
UPDATE advisory_type SET preference = 300 WHERE name = 'enhancement';
UPDATE advisory_type SET preference = 400 WHERE name = 'bugfix';
UPDATE advisory_type SET preference = 500 WHERE name = 'security';
UPDATE advisory_type SET preference = 200 WHERE name = 'unspecified';
