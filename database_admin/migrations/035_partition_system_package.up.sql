CREATE OR REPLACE FUNCTION create_table_partitions(tbl regclass, parts INTEGER, rest text)
    RETURNS VOID AS
$$
DECLARE
    I INTEGER;
BEGIN
    I := 0;
    WHILE I < parts
        LOOP
            EXECUTE 'CREATE TABLE ' || text(tbl) || '_' || text(I) || ' PARTITION OF ' || text(tbl) ||
                    ' FOR VALUES WITH ' || ' ( MODULUS ' || text(parts) || ', REMAINDER ' || text(I) || ')' ||
                    rest || ';';
            I = I + 1;
        END LOOP;
END;
$$ LANGUAGE plpgsql;

DROP TABLE IF EXISTS system_package;

CREATE TABLE IF NOT EXISTS system_package
(
    rh_account_id INT NOT NULL REFERENCES rh_account,
    system_id     INT NOT NULL REFERENCES system_platform,
    package_id    INT NOT NULL REFERENCES package,
    -- Use null to represent up-to-date packages
    update_data   JSONB DEFAULT NULL,
    PRIMARY KEY (rh_account_id, system_id, package_id)
) PARTITION BY HASH (rh_account_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_package TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package TO vmaas_sync;

SELECT create_table_partitions('system_package', 16,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

GRANT SELECT ON ALL TABLES IN SCHEMA public TO evaluator;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO listener;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO manager;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO vmaas_sync;
