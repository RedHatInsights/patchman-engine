CREATE OR REPLACE FUNCTION rename_table_with_partitions(tbl regclass, oldtext text, newtext text)
    RETURNS VOID AS
$$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        EXECUTE 'ALTER TABLE ' || r.relname || ' RENAME TO ' || replace(r.relname, oldtext, newtext);
    END LOOP;
    EXECUTE 'ALTER TABLE ' || text(tbl) || ' RENAME TO ' || replace(text(tbl), oldtext, newtext);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION rename_index_with_partitions(idx regclass, oldtext text, newtext text)
    RETURNS VOID AS
$$
DECLARE
    r record;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(idx)
    LOOP
        EXECUTE 'ALTER INDEX ' || r.relname || ' RENAME TO ' || replace(r.relname, oldtext, newtext);
    END LOOP;
    EXECUTE 'ALTER INDEX ' || text(idx) || ' RENAME TO ' || replace(text(idx), oldtext, newtext);
END;
$$ LANGUAGE plpgsql;


-- switch partitioned tables

-- rename old tables
ALTER TABLE system_platform RENAME TO system_platform_v1;
ALTER TABLE system_platform_v1
    RENAME CONSTRAINT system_platform_display_name_check TO system_platform_display_name_check_v1;
ALTER SEQUENCE system_platform_id_seq RENAME TO system_platform_id_seq_v1;
ALTER INDEX system_platform_pkey RENAME TO system_platform_pkey_v1;

ALTER TABLE system_advisories RENAME TO system_advisories_v1;
ALTER INDEX system_advisories_pkey RENAME TO system_advisories_pkey_v1;

-- move new table
SELECT rename_table_with_partitions('system_platform_v2', '_v2', '');
ALTER TABLE system_platform
    RENAME CONSTRAINT system_platform_v2_display_name_check TO system_platform_display_name_check;
ALTER SEQUENCE system_platform_v2_id_seq RENAME TO system_platform_id_seq;
SELECT rename_index_with_partitions('system_platform_v2_pkey', '_v2', '');
SELECT rename_index_with_partitions('system_platform_v2_rh_account_id_inventory_id_key', '_v2', '');

SELECT rename_table_with_partitions('system_advisories_v2', '_v2', '');
SELECT rename_index_with_partitions('system_advisories_v2_pkey', '_v2', '');

-- update functions to the new tables
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
        INNER JOIN system_platform sp
           ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id
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
        SELECT sp.rh_account_id, sp.id
        FROM system_platform sp
        WHERE (sp.id = system_id_in OR system_id_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
        )
        UPDATE system_platform sp
           SET advisory_count_cache = system_advisories_count(sp.id, NULL),
               advisory_enh_count_cache = system_advisories_count(sp.id, 1),
               advisory_bug_count_cache = system_advisories_count(sp.id, 2),
               advisory_sec_count_cache = system_advisories_count(sp.id, 3)
          FROM to_update to_up
         WHERE sp.rh_account_id = to_up.rh_account_id AND sp.id = to_up.id;
    GET DIAGNOSTICS COUNT = ROW_COUNT;
    RETURN COUNT;
END;
$refresh_system$ LANGUAGE plpgsql;

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
        SELECT rh_account_id, id
        FROM system_platform
        WHERE stale_warning_timestamp < now()
          AND stale = false
        ORDER BY rh_account_id, id FOR UPDATE OF system_platform
        )
        UPDATE system_platform sp
           SET stale = true
          FROM ids
         WHERE sp.rh_account_id = ids.rh_account_id
           AND sp.id = ids.id;
    GET DIAGNOSTICS marked = ROW_COUNT;
    RETURN marked;
END;
$fun$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION delete_culled_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    culled integer;
    _uuid uuid;
BEGIN
    WITH ids AS (SELECT inventory_id
                 FROM system_platform
                 WHERE culled_timestamp < now()
                 ORDER BY id FOR UPDATE OF system_platform
        )
        SELECT delete_system(inventory_id) into _uuid from ids;
    GET DIAGNOSTICS culled = ROW_COUNT;
    RETURN culled;
END;
$fun$ LANGUAGE plpgsql;