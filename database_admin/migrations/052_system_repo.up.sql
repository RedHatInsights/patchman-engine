CALL raise_notice('system_repo migration:');

-- system_repo
ALTER TABLE system_repo
    ADD COLUMN IF NOT EXISTS rh_account_id INT,
    DROP CONSTRAINT IF EXISTS system_platform_id;

CALL raise_notice('column added');

-- data migration
UPDATE system_repo sr
    SET rh_account_id = (SELECT sp.rh_account_id FROM system_platform sp WHERE sp.id = sr.system_id);

CALL raise_notice('data altered');

-- system_repo
ALTER TABLE system_repo
    ALTER COLUMN rh_account_id SET NOT NULL,
    DROP CONSTRAINT IF EXISTS system_repo_rh_account_id_system_id_repo_id_key,
    ADD UNIQUE (rh_account_id, system_id, repo_id),
    DROP CONSTRAINT IF EXISTS system_repo_system_id_repo_id_key,
    DROP CONSTRAINT IF EXISTS system_platform_id,
    ADD CONSTRAINT system_platform_id
        FOREIGN KEY (rh_account_id, system_id)
            REFERENCES system_platform_v2 (rh_account_id, id);
DROP INDEX IF EXISTS system_repo_system_id_idx;

CALL raise_notice('constraints updated');

GRANT SELECT, INSERT, UPDATE, DELETE ON system_repo TO listener;
GRANT DELETE ON system_repo TO manager;
GRANT SELECT ON system_repo TO evaluator;
GRANT SELECT, DELETE on system_repo to vmaas_sync;

-- user for evaluator component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO evaluator;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO evaluator;

-- user for listener component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO listener;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO listener;

-- user for UI manager component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO manager;

-- user for VMaaS sync component
GRANT SELECT ON ALL TABLES IN SCHEMA public TO vmaas_sync;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vmaas_sync;

CALL raise_notice('permission granted');
