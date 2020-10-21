CREATE TABLE IF NOT EXISTS system_package_v1
(
    rh_account_id INT NOT NULL REFERENCES rh_account,
    system_id     INT NOT NULL,
    package_id    INT NOT NULL REFERENCES package,
    -- Use null to represent up-to-date packages
    update_data   JSONB DEFAULT NULL,
    latest_evra   TEXT GENERATED ALWAYS AS ( ((update_data ->> -1)::jsonb ->> 'evra')::text) STORED,
    PRIMARY KEY (rh_account_id, system_id, package_id) INCLUDE (latest_evra)
) PARTITION BY HASH (rh_account_id);


GRANT SELECT, INSERT, UPDATE, DELETE ON system_package_v1 TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package_v1 TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package_v1 TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package_v1 TO vmaas_sync;

SELECT create_table_partitions('system_package_v1', 128,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

GRANT SELECT ON ALL TABLES IN SCHEMA public TO evaluator;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO listener;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO manager;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO vmaas_sync;


DO
$$
    DECLARE
        row text;
    BEGIN
        FOR row IN (SELECT indexname from pg_indexes t where t.indexname ~ '^system_package_[0-9]+$')
            LOOP
                RAISE NOTICE 'Copying the % partition', row;
                EXECUTE 'INSERT INTO system_package_v1 ( SELECT rh_account_id, system_id, package_id, update_data FROM ' ||
                        row || ') ON CONFLICT DO NOTHING';
            END LOOP;
    END
$$ LANGUAGE plpgsql;


DO
$$
    DECLARE
        old integer;
        new integer;
    BEGIN
        SELECT count(*) from system_package into old;
        SELECT count(*) from system_package_v1 into new;
        RAISE NOTICE 'Old: %, new: %', old, new;
    END
$$ LANGUAGE plpgsql;
