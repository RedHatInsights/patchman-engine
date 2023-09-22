CREATE TABLE IF NOT EXISTS system_package_data
(
    rh_account_id  INT    NOT NULL,
    system_id      BIGINT NOT NULL,
    update_data    JSONB DEFAULT NULL,

    PRIMARY KEY (rh_account_id, system_id),
    FOREIGN KEY (rh_account_id, system_id) REFERENCES system_platform (rh_account_id, id)
) PARTITION BY HASH (rh_account_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_package_data TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package_data TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package_data TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package_data TO vmaas_sync;

SELECT create_table_partitions('system_package_data', 32,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

CREATE TABLE IF NOT EXISTS package_system_data
(
    rh_account_id   INT    NOT NULL,
    package_name_id BIGINT NOT NULL,
    update_data     JSONB DEFAULT NULL,

    PRIMARY KEY (rh_account_id, package_name_id),
    FOREIGN KEY (rh_account_id) REFERENCES rh_account (id)
) PARTITION BY HASH (rh_account_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON package_system_data TO evaluator;
GRANT SELECT, UPDATE, DELETE ON package_system_data TO listener;
GRANT SELECT, UPDATE, DELETE ON package_system_data TO manager;
GRANT SELECT, UPDATE, DELETE ON package_system_data TO vmaas_sync;

SELECT create_table_partitions('package_system_data', 32,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

GRANT SELECT ON ALL TABLES IN SCHEMA public TO evaluator;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO evaluator;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO listener;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO listener;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO manager;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO vmaas_sync;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vmaas_sync;

CREATE OR REPLACE FUNCTION update_status(update_data jsonb)
    RETURNS TEXT as
$$
DECLARE
    len int;
BEGIN
    len = jsonb_array_length(jsonb_path_query_array(update_data, '$ ? (@.status != "Installed")'));
    IF len IS NULL or len = 0 THEN
        RETURN 'None';
    END IF;
    len = jsonb_array_length(jsonb_path_query_array(update_data, '$ ? (@.status == "Installable")'));
    IF len > 0 THEN
        RETURN 'Installable';
    END IF;
    RETURN 'Applicable';
END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;
