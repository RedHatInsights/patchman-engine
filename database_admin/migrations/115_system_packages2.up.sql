CREATE TABLE IF NOT EXISTS system_package2
(
    rh_account_id  INT    NOT NULL,
    system_id      BIGINT NOT NULL,
    name_id        BIGINT NOT NULL REFERENCES package_name (id),
    package_id     BIGINT NOT NULL REFERENCES package (id),
    -- Use null to represent up-to-date packages
    installable_id BIGINT REFERENCES package (id),
    applicable_id  BIGINT REFERENCES package (id),

    PRIMARY KEY (rh_account_id, system_id, package_id),
    FOREIGN KEY (rh_account_id, system_id) REFERENCES system_platform (rh_account_id, id)
) PARTITION BY HASH (rh_account_id);

CREATE INDEX IF NOT EXISTS system_package2_account_pkg_name_idx
    ON system_package2 (rh_account_id, name_id) INCLUDE (system_id, package_id, installable_id, applicable_id);

CREATE INDEX IF NOT EXISTS system_package2_package_id_idx on system_package2 (package_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_package2 TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package2 TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package2 TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package2 TO vmaas_sync;

SELECT create_table_partitions('system_package2', 128,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

GRANT SELECT ON ALL TABLES IN SCHEMA public TO evaluator;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO evaluator;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO listener;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO listener;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO manager;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO vmaas_sync;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vmaas_sync;
