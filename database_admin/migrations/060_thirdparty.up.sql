ALTER TABLE system_platform add column third_party BOOLEAN NOT NULL DEFAULT false;

GRANT SELECT, INSERT, UPDATE, DELETE ON system_repo TO evaluator;
