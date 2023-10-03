DO $$
    DECLARE
      rownum INT;
      total INT;
      account_id INT;
      sys_id BIGINT;
      update_list TEXT[] := NULL;
      json TEXT := '';
      pkgname_id BIGINT;
    BEGIN
      -- order accounts so that we read from system_package by partitions
      FOR rownum, total, account_id IN
          SELECT row_number() over (), count(*) over (), id FROM
            ( SELECT id FROM rh_account order by hash_partition_id(id, 128), id) as o
      LOOP
          RAISE NOTICE 'Migrating account % (%/%)', account_id, rownum, total;
          FOR sys_id, update_list IN
              SELECT system_id,
                     array_agg(CONCAT('"', package_id, '":{', '"installable":"' || i.evra || '"',
                                      CASE WHEN i.evra IS NOT NULL AND a.evra IS NOT NULL THEN ',' ELSE '' END,
                                      '"applicable":"' || a.evra || '"', '}')) as update_list
                FROM system_package2 sp
                LEFT JOIN package i ON sp.installable_id = i.id
                LEFT JOIN package a ON sp.applicable_id = a.id
               WHERE sp.rh_account_id = account_id
               GROUP BY system_id
          LOOP
              json := '{' || array_to_string(update_list, ',') || '}';
              -- RAISE NOTICE 'system_package_data (%, %, %)', account_id, sys_id, json;
              INSERT INTO system_package_data VALUES (account_id, sys_id, json::jsonb); 
          END LOOP;

          FOR pkgname_id, update_list IN
              SELECT sp.name_id,
                     array_agg(CONCAT('"', sp.system_id, '":{"installed":"', p.evra,'"',
                               ',"installable":"' || i.evra || '"',
                               ',"applicable":"' || a.evra || '"', ']')) as update_list
                FROM system_package2 sp
                JOIN package p ON p.id = sp.package_id
                LEFT JOIN package i ON sp.installable_id = i.id
                LEFT JOIN package a ON sp.applicable_id = a.id
               WHERE sp.rh_account_id = account_id
               GROUP BY sp.name_id
          LOOP
              json := '{' || array_to_string(update_list, ',') || '}';
              -- RAISE NOTICE 'package_system_data (%, %, %)', account_id, pkgname_id, json;
              INSERT INTO package_system_data VALUES (account_id, pkgname_id, json::jsonb); 
          END LOOP;
      END LOOP;
    END;
$$
;    


DROP FUNCTION IF EXISTS update_status(update_data jsonb);

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

    DELETE
    FROM system_package_data
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    UPDATE package_system_data
    SET update_data = update_data - v_system_id::text
    WHERE rh_account_id = v_account_id;
    DELETE
    FROM package_system_data
    WHERE rh_account_id = v_account_id
      AND (update_data IS NULL OR update_data = '{}'::jsonb);

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
         system_package_data as (
             DELETE
                 FROM system_package_data
                     WHERE (rh_account_id, system_id) in (select rh_account_id, id from systems)
         ),
         package_system_data_u as (
             UPDATE package_system_data psd
                SET update_data = update_data - s.id::text
               FROM (select rh_account_id, id from systems) s
              WHERE psd.rh_account_id = s.rh_account_id
         ),
         package_system_data_d as (
             DELETE
                 FROM package_system_data
                     WHERE rh_account_id in (select rh_account_id from systems)
                     AND (update_data IS NULL OR update_data = '{}'::jsonb)
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
