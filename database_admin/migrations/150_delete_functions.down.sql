DROP FUNCTION IF EXISTS delete_system(inventory_id_in uuid);
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
    FROM system_inventory
    WHERE inventory_id = inventory_id_in
    LIMIT 1
        FOR UPDATE OF system_inventory
    INTO v_system_id, v_account_id;

    IF v_system_id IS NULL OR v_account_id IS NULL THEN
        RAISE NOTICE 'Not found';
        RETURN;
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

    RETURN QUERY DELETE FROM system_inventory
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
        FROM system_inventory
        WHERE inventory_id = ANY (inventory_ids)
        ORDER BY rh_account_id, id FOR UPDATE OF system_inventory),
         marked as (
             UPDATE system_inventory sp
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
         packages2 as (
             DELETE
                 FROM system_package2
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         patch_systems as (
             DELETE
                 FROM system_patch
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         deleted as (
             DELETE
                 FROM system_inventory
                     WHERE (rh_account_id, id) in (select rh_account_id, id from systems)
                     RETURNING id
         )
    SELECT count(*)
    FROM deleted
    INTO tmp_cnt;

    RETURN tmp_cnt;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION delete_culled_systems(delete_limit INTEGER)
    RETURNS INTEGER
AS
$fun$
DECLARE
    ids UUID[];
BEGIN
    ids := ARRAY(
            SELECT inventory_id
            FROM system_inventory
            WHERE culled_timestamp < now()
            ORDER BY id
            LIMIT delete_limit
        );
    return delete_systems(ids);
END;
$fun$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION mark_stale_systems(mark_limit integer)
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (
        SELECT rh_account_id, id, stale_warning_timestamp < now() as expired
        FROM system_inventory
        WHERE stale != (stale_warning_timestamp < now())
        ORDER BY rh_account_id, id FOR UPDATE OF system_inventory
        LIMIT mark_limit
    )
    UPDATE system_inventory si
    SET stale = ids.expired
    FROM ids
    WHERE si.rh_account_id = ids.rh_account_id
      AND si.id = ids.id;
    GET DIAGNOSTICS marked = ROW_COUNT;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;
