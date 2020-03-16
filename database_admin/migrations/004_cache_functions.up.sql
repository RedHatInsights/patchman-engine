DROP FUNCTION IF EXISTS opt_out_system_update_cache CASCADE;
-- opt_out_system_update_cache
CREATE OR REPLACE FUNCTION opt_out_system_update_cache()
    RETURNS TRIGGER AS
$opt_out_system_update_cache$
BEGIN
    IF (TG_OP = 'UPDATE') AND NEW.last_evaluation IS NOT NULL THEN
        -- system opted out
        IF OLD.opt_out = FALSE AND NEW.opt_out = TRUE THEN
            -- decrement affected advisory counts for system
            WITH to_update_advisories AS (
                SELECT ead.advisory_id, ead.status_id AS global_status_id, sa.status_id
                FROM advisory_account_data ead
                         INNER JOIN
                     system_advisories sa ON ead.advisory_id = sa.advisory_id
                WHERE ead.rh_account_id = NEW.rh_account_id
                  AND sa.system_id = NEW.id
                  AND sa.when_patched IS NULL
                ORDER BY ead.advisory_id
                    FOR UPDATE OF ead
                -- decrement systems_affected and systems_status_divergent in case status is different
            ),
                 update_divergent AS (
                     UPDATE advisory_account_data ead
                         SET systems_affected = systems_affected - 1,
                             systems_status_divergent = systems_status_divergent - 1
                         FROM to_update_advisories
                         WHERE ead.advisory_id = to_update_advisories.advisory_id AND
                               ead.rh_account_id = NEW.rh_account_id AND
                               to_update_advisories.global_status_id != to_update_advisories.status_id
                 )
                 -- decrement only systems_affected in case status is same
            UPDATE advisory_account_data ead
            SET systems_affected = systems_affected - 1
            FROM to_update_advisories
            WHERE ead.advisory_id = to_update_advisories.advisory_id
              AND ead.rh_account_id = NEW.rh_account_id
              AND to_update_advisories.global_status_id = to_update_advisories.status_id;
            -- delete zero advisory counts
            DELETE
            FROM advisory_account_data
            WHERE rh_account_id = NEW.rh_account_id
              AND systems_affected = 0;

            -- system opted in
        ELSIF OLD.opt_out = TRUE AND NEW.opt_out = FALSE THEN
            -- increment affected advisory counts for system
            WITH to_update_advisories AS (
                SELECT ead.advisory_id, ead.status_id AS global_status_id, sa.status_id
                FROM advisory_account_data ead
                         INNER JOIN
                     system_advisories sa ON ead.advisory_id = sa.advisory_id
                WHERE ead.rh_account_id = NEW.rh_account_id
                  AND sa.system_id = NEW.id
                  AND sa.when_patched IS NULL
                ORDER BY ead.advisory_id
                    FOR UPDATE OF ead
                -- increment systems_affected and systems_status_divergent in case status is different
            ),
                 update_divergent AS (
                     UPDATE advisory_account_data ead
                         SET systems_affected = systems_affected + 1,
                             systems_status_divergent = systems_status_divergent + 1
                         FROM to_update_advisories
                         WHERE ead.advisory_id = to_update_advisories.advisory_id AND
                               ead.rh_account_id = NEW.rh_account_id AND
                               to_update_advisories.global_status_id != to_update_advisories.status_id
                 )
                 -- increment only systems_affected in case status is same
            UPDATE advisory_account_data ead
            SET systems_affected = systems_affected + 1
            FROM to_update_advisories
            WHERE ead.advisory_id = to_update_advisories.advisory_id
              AND ead.rh_account_id = NEW.rh_account_id
              AND to_update_advisories.global_status_id = to_update_advisories.status_id;
            -- insert cache if not exists
            INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
            SELECT sa.advisory_id, NEW.rh_account_id, 1
            FROM system_advisories sa
            WHERE sa.system_id = NEW.id
              AND sa.when_patched IS NULL
              AND NOT EXISTS(
                    SELECT 1
                    FROM advisory_account_data
                    WHERE rh_account_id = NEW.rh_account_id
                      AND advisory_id = sa.advisory_id
                )
            ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET systems_affected = advisory_account_data.systems_affected + EXCLUDED.systems_affected;
        END IF;
    END IF;
    RETURN NEW;
END;
$opt_out_system_update_cache$
    LANGUAGE 'plpgsql';

-- update system advisories counts (all and according types)
CREATE OR REPLACE FUNCTION update_system_caches(system_id_in INT)
    RETURNS VOID AS
$update_system_caches$
BEGIN
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE sp.id = system_id_in
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems;
END;
$update_system_caches$
    LANGUAGE 'plpgsql';

-- count system advisories according to advisory type
CREATE OR REPLACE FUNCTION system_advisories_count(system_id_in INT, advisory_type_id_in INT)
    RETURNS INT AS
$system_advisories_count$
DECLARE
    result_cnt INT;
BEGIN
    SELECT COUNT(advisory_id)
    FROM system_advisories sa
             JOIN advisory_metadata am ON sa.advisory_id = am.id
    WHERE am.advisory_type_id = advisory_type_id_in
      AND sa.system_id = system_id_in
      AND sa.when_patched IS NULL
    INTO result_cnt;
    RETURN result_cnt;
END;
$system_advisories_count$
    LANGUAGE 'plpgsql';

-- refresh_all_cached_counts
-- WARNING: executing this procedure takes long time,
--          use only when necessary, e.g. during upgrade to populate initial caches
CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    -- update advisories count for ordered systems
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        ORDER BY sp.rh_account_id, sp.id
            FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered advisory
    WITH locked_rows AS (
        SELECT ead.rh_account_id, ead.advisory_id
        FROM advisory_account_data ead
        ORDER BY ead.rh_account_id, ead.advisory_id
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sa.advisory_id, sp.rh_account_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
             GROUP BY sa.advisory_id, sp.rh_account_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_id, rh_account_id, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE (advisory_id, rh_account_id) NOT IN (SELECT advisory_id, rh_account_id FROM current_counts);
END;
$refresh_all_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_account_cached_counts(rh_account_in varchar)
    RETURNS void AS
$refresh_account_cached_counts$
DECLARE
    rh_account_id_in INT;
BEGIN
    -- update advisory count for ordered systems
    SELECT id FROM rh_account WHERE name = rh_account_in INTO rh_account_id_in;
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE sp.rh_account_id = rh_account_id_in
        ORDER BY sp.id
            FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems
    WHERE sp.id = to_update_systems.id;

    -- update system count for ordered advisory
    WITH locked_rows AS (
        SELECT ead.advisory_id
        FROM advisory_account_data ead
        WHERE ead.rh_account_id = rh_account_id_in
        ORDER BY ead.advisory_id
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sa.advisory_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
               AND sp.rh_account_id = rh_account_id_in
             GROUP BY sa.advisory_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_id, rh_account_id_in, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE advisory_id NOT IN (SELECT advisory_id FROM current_counts)
      AND rh_account_id = rh_account_id_in;
END;
$refresh_account_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_advisory_cached_counts(advisory_name varchar)
    RETURNS void AS
$refresh_advisory_cached_counts$
DECLARE
    advisory_md_id INT;
BEGIN
    -- update system count for advisory
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_md_id;
    WITH locked_rows AS (
        SELECT ead.rh_account_id
        FROM advisory_account_data ead
        WHERE ead.advisory_id = advisory_md_id
        ORDER BY ead.rh_account_id
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sp.rh_account_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
               AND sa.advisory_id = advisory_md_id
             GROUP BY sp.rh_account_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_md_id, rh_account_id, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE rh_account_id NOT IN (SELECT rh_account_id FROM current_counts)
      AND advisory_id = advisory_md_id;
END;
$refresh_advisory_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_advisory_account_cached_counts(advisory_name varchar, rh_account_name varchar)
    RETURNS void AS
$refresh_advisory_account_cached_counts$
DECLARE
    advisory_md_id   INT;
    rh_account_id_in INT;
BEGIN
    -- update system count for ordered advisories
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_md_id;
    SELECT id FROM rh_account WHERE name = rh_account_name INTO rh_account_id_in;
    WITH locked_rows AS (
        SELECT ead.rh_account_id, ead.advisory_id
        FROM advisory_account_data ead
        WHERE ead.advisory_id = advisory_md_id
          AND ead.rh_account_id = rh_account_id_in
            FOR UPDATE OF ead
    ),
         current_counts AS (
             SELECT sa.advisory_id, sp.rh_account_id, count(sa.system_id) as systems_affected
             FROM system_advisories sa
                      INNER JOIN
                  system_platform sp ON sa.system_id = sp.id
             WHERE sp.last_evaluation IS NOT NULL
               AND sp.opt_out = FALSE
               AND sa.when_patched IS NULL
               AND sa.advisory_id = advisory_md_id
               AND sp.rh_account_id = rh_account_id_in
             GROUP BY sa.advisory_id, sp.rh_account_id
         ),
         upserted AS (
             INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
                 SELECT advisory_md_id, rh_account_id_in, systems_affected FROM current_counts
                 ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_affected = EXCLUDED.systems_affected
         )
    DELETE
    FROM advisory_account_data
    WHERE NOT EXISTS(SELECT 1 FROM current_counts)
      AND advisory_id = advisory_md_id
      AND rh_account_id = rh_account_id_in;
END;
$refresh_advisory_account_cached_counts$
    LANGUAGE 'plpgsql';


CREATE OR REPLACE FUNCTION refresh_system_cached_counts(inventory_id_in varchar)
    RETURNS void AS
$refresh_system_cached_counts$
BEGIN
    -- update advisory count for system
    UPDATE system_platform sp
    SET advisory_count_cache     = (
        SELECT COUNT(advisory_id)
        FROM system_advisories sa
        WHERE sa.system_id = sp.id
          AND sa.when_patched IS NULL
    ),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    WHERE sp.inventory_id = inventory_id_in;
END;
$refresh_system_cached_counts$
    LANGUAGE 'plpgsql';


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
    RETURN QUERY DELETE FROM system_platform
        WHERE inventory_id = inventory_id_in
        RETURNING inventory_id;
END;
$delete_system$
    LANGUAGE 'plpgsql';



CREATE TRIGGER system_platform_opt_out_cache
    AFTER UPDATE OF opt_out
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE opt_out_system_update_cache();
