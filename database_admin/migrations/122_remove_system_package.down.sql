CREATE OR REPLACE FUNCTION delete_system(inventory_id_in uuid)
    RETURNS TABLE
            (
                deleted_inventory_id uuid
            )
AS
$delete_system$
DECLARE
    v_system_id  INT;
    v_account_id INT;
BEGIN
    -- opt out to refresh cache and then delete
    SELECT id, rh_account_id
    FROM system_platform
    WHERE inventory_id = inventory_id_in
    LIMIT 1
        FOR UPDATE OF system_platform
    INTO v_system_id, v_account_id;

    IF v_system_id IS NULL OR v_account_id IS NULL THEN
        RAISE NOTICE 'Not found';
        RETURN;
    END IF;

    UPDATE system_platform
    SET stale = true
    WHERE rh_account_id = v_account_id
      AND id = v_system_id;

    DELETE
    FROM system_advisories
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_repo
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_package
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_package2
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    RETURN QUERY DELETE FROM system_platform
        WHERE rh_account_id = v_account_id AND
              id = v_system_id
        RETURNING inventory_id;
END;
$delete_system$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION delete_systems(inventory_ids UUID[])
    RETURNS INTEGER
AS
$$
DECLARE
    tmp_cnt INTEGER;
BEGIN

    WITH systems as (
        SELECT rh_account_id, id
        FROM system_platform
        WHERE inventory_id = ANY (inventory_ids)
        ORDER BY rh_account_id, id FOR UPDATE OF system_platform),
         marked as (
             UPDATE system_platform sp
                 SET stale = true
                 WHERE (rh_account_id, id) in (select rh_account_id, id from systems)
         ),
         advisories as (
             DELETE
                 FROM system_advisories
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         repos as (
             DELETE
                 FROM system_repo
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         packages as (
             DELETE
                 FROM system_package
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         packages2 as (
             DELETE
                 FROM system_package2
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         deleted as (
             DELETE
                 FROM system_platform
                     WHERE (rh_account_id, id) in (select rh_account_id, id from systems)
                     RETURNING id
         )
    SELECT count(*)
    FROM deleted
    INTO tmp_cnt;

    RETURN tmp_cnt;
END
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION update_status(update_data jsonb)
    RETURNS TEXT as
$$
DECLARE
    len int;
BEGIN
    len = jsonb_array_length(update_data);
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

CREATE TABLE IF NOT EXISTS system_package
(
    rh_account_id INT                                  NOT NULL REFERENCES rh_account,
    system_id     BIGINT                               NOT NULL,
    package_id    BIGINT                               NOT NULL REFERENCES package,
    -- Use null to represent up-to-date packages
    update_data   JSONB DEFAULT NULL,
    latest_evra   TEXT GENERATED ALWAYS AS ( ((update_data ->> -1)::jsonb ->> 'evra')::text) STORED
                  CHECK(NOT empty(latest_evra)),
    name_id       BIGINT REFERENCES package_name (id) NOT NULL,

    PRIMARY KEY (rh_account_id, system_id, package_id) INCLUDE (latest_evra)
) PARTITION BY HASH (rh_account_id);

CREATE INDEX IF NOT EXISTS system_package_name_pkg_system_idx
    ON system_package (rh_account_id, name_id, package_id, system_id) INCLUDE (latest_evra);

CREATE INDEX IF NOT EXISTS system_package_package_id_idx on system_package (package_id);

GRANT SELECT, INSERT, UPDATE, DELETE ON system_package TO evaluator;
GRANT SELECT, UPDATE, DELETE ON system_package TO listener;
GRANT SELECT, UPDATE, DELETE ON system_package TO manager;
GRANT SELECT, UPDATE, DELETE ON system_package TO vmaas_sync;

SELECT create_table_partitions('system_package', 128,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);
