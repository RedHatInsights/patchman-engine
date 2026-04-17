DROP FUNCTION IF EXISTS delete_system(inventory_id_in uuid);
CREATE OR REPLACE FUNCTION delete_system(inventory_id_in uuid)
    RETURNS uuid
AS
$delete_system$
DECLARE
    v_system_id  INT;
    v_account_id INT;
    v_inventory_id uuid;
BEGIN
    -- opt out to refresh cache and then delete
    SELECT id, rh_account_id
    FROM system_inventory
    WHERE inventory_id = inventory_id_in
    LIMIT 1
        FOR UPDATE OF system_inventory
    INTO v_system_id, v_account_id;

    IF v_system_id IS NULL OR v_account_id IS NULL THEN
        RAISE NOTICE 'Not found';
        RETURN NULL;
    END IF;

    UPDATE system_inventory
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
    FROM system_package2
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE
    FROM system_patch
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    DELETE FROM system_inventory
        WHERE rh_account_id = v_account_id AND
              id = v_system_id
        RETURNING inventory_id INTO v_inventory_id;

    RETURN v_inventory_id;
END;
$delete_system$ LANGUAGE 'plpgsql';

DROP FUNCTION IF EXISTS delete_systems(UUID[]);
DROP FUNCTION IF EXISTS delete_culled_systems(INTEGER);
DROP FUNCTION IF EXISTS mark_stale_systems(INTEGER);
