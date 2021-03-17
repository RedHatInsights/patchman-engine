ALTER TABLE repo add column third_party BOOLEAN NOT NULL DEFAULT true;

GRANT SELECT, INSERT, UPDATE, DELETE ON repo TO evaluator;
GRANT SELECT, UPDATE ON repo TO vmaas_sync;
