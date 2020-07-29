CREATE OR REPLACE FUNCTION delete_system(inventory_id_in varchar)
    RETURNS TABLE
            (
                deleted_inventory_id TEXT
            )
AS
$delete_system$
BEGIN
    -- opt out to refresh cache and then delete
    WITH locked_row AS (
        SELECT id
        FROM system_platform
        WHERE inventory_id = inventory_id_in
            FOR UPDATE
    )
    UPDATE system_platform
    SET opt_out = true
    WHERE inventory_id = inventory_id_in;

    DELETE
    FROM system_advisories
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);

    DELETE
    FROM system_repo
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);

    DELETE
    FROM system_package
    WHERE system_id = (SELECT id from system_platform WHERE inventory_id = inventory_id_in);

    RETURN QUERY DELETE FROM system_platform
        WHERE inventory_id = inventory_id_in
        RETURNING inventory_id;
END;
$delete_system$ LANGUAGE 'plpgsql';

GRANT UPDATE, DELETE ON system_package TO listener;
GRANT UPDATE, DELETE ON system_package TO manager;
GRANT UPDATE, DELETE ON system_package TO vmaas_sync;