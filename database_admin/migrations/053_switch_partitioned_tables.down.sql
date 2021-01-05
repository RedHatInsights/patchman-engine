-- switch partitioned tables
CALL raise_notice('reverting partitioned tables');

-- move partitioned tables to _v2
SELECT rename_table_with_partitions('system_platform', 'system_platform', 'system_platform_v2');
ALTER TABLE IF EXISTS system_platform_v2
    RENAME CONSTRAINT system_platform_display_name_check TO system_platform_v2_display_name_check;
ALTER SEQUENCE IF EXISTS system_platform_id_seq RENAME TO system_platform_v2_id_seq;
SELECT rename_index_with_partitions('system_platform_pkey', 'system_platform', 'system_platform_v2');
SELECT rename_index_with_partitions('system_platform_rh_account_id_inventory_id_key', 'system_platform', 'system_platform_v2');

SELECT rename_table_with_partitions('system_advisories', 'system_advisories', 'system_advisories_v2');
SELECT rename_index_with_partitions('system_advisories_pkey', 'system_advisories', 'system_advisories_v2');

CALL raise_notice('partitioned tables renamed to _v2');

-- rename old tables
ALTER TABLE IF EXISTS system_platform_v1 RENAME TO system_platform;
ALTER TABLE IF EXISTS system_platform
    RENAME CONSTRAINT system_platform_display_name_check_v1 TO system_platform_display_name_check;
ALTER SEQUENCE IF EXISTS system_platform_id_seq_v1 RENAME TO system_platform_id_seq;
ALTER INDEX IF EXISTS system_platform_pkey_v1 RENAME TO system_platform_pkey;

ALTER TABLE IF EXISTS system_advisories_v1 RENAME TO system_advisories;
ALTER INDEX IF EXISTS system_advisories_pkey_v1 RENAME TO system_advisories_pkey;

CALL raise_notice('_v1 tables moved back');

-- restore old functions
CREATE OR REPLACE FUNCTION refresh_advisory_caches_multi(advisory_ids_in INTEGER[] DEFAULT NULL,
                                                         rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_advisory$
BEGIN
    -- Lock rows
    PERFORM aad.rh_account_id, aad.advisory_id
    FROM advisory_account_data aad
    WHERE (aad.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
      AND (aad.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        FOR UPDATE OF aad;

    WITH current_counts AS (
        SELECT sa.advisory_id, sp.rh_account_id, count(sa.system_id) as systems_affected
        FROM system_advisories sa
                 INNER JOIN
             system_platform sp ON sa.system_id = sp.id
        WHERE sp.last_evaluation IS NOT NULL
          AND sp.opt_out = FALSE
          AND sp.stale = FALSE
          AND sa.when_patched IS NULL
          AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        GROUP BY sa.advisory_id, sp.rh_account_id
    ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_id, rh_account_id, systems_affected
                 FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE (advisory_id, rh_account_id) NOT IN (SELECT advisory_id, rh_account_id FROM current_counts)
      AND (advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
      AND (rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);
END;
$refresh_advisory$ language plpgsql;

CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in INTEGER DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS INTEGER AS
$refresh_system$
DECLARE
    COUNT INTEGER;
BEGIN
    WITH to_update AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE (sp.id = system_id_in OR system_id_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
    ),
         updated as (
             UPDATE system_platform sp
                 SET advisory_count_cache = system_advisories_count(sp.id, NULL),
                     advisory_enh_count_cache = system_advisories_count(sp.id, 1),
                     advisory_bug_count_cache = system_advisories_count(sp.id, 2),
                     advisory_sec_count_cache = system_advisories_count(sp.id, 3)
                 FROM to_update to_up
                 WHERE sp.id = to_up.id
                 RETURNING sp.id)
    SELECT count(*)
    FROM updated
    INTO COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION delete_system(inventory_id_in varchar)
    RETURNS TABLE
            (
                deleted_inventory_id TEXT
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
        FOR UPDATE OF system_platform
    INTO v_system_id, v_account_id;

    IF v_system_id IS NULL OR v_account_id IS NULL THEN
        RAISE NOTICE 'Not found';
        RETURN;
    END IF;

    UPDATE system_platform
    SET opt_out = true
    WHERE rh_account_id = v_account_id
      AND id = v_system_id;

    DELETE
    FROM system_advisories
    WHERE system_id = v_system_id;

    DELETE
    FROM system_repo
    WHERE system_id = v_system_id;

    DELETE
    FROM system_package
    WHERE rh_account_id = v_account_id
      AND system_id = v_system_id;

    RETURN QUERY DELETE FROM system_platform
        WHERE rh_account_id = v_account_id AND
              id = v_system_id
        RETURNING inventory_id;
END;
$delete_system$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION mark_stale_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    WITH ids AS (
        SELECT id
        FROM system_platform
        WHERE stale_warning_timestamp < now()
          AND stale = false
        ORDER BY id FOR UPDATE OF system_platform
    ),
         updated as (
             UPDATE system_platform
                 SET stale = true
                 FROM ids
                 WHERE system_platform.id = ids.id
                 RETURNING ids.id
         )
    SELECT count(*)
    FROM updated
    INTO marked;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;