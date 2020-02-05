DROP FUNCTION IF EXISTS opt_out_system_update_cache() CASCADE;
DROP TRIGGER IF EXISTS system_platform_opt_out_cache ON system_platform;

CREATE OR REPLACE FUNCTION ternary(cond BOOL, iftrue ANYELEMENT, iffalse ANYELEMENT)
    RETURNS ANYELEMENT
AS
$$
SELECT CASE WHEN cond = TRUE THEN iftrue else iffalse END;
$$ LANGUAGE SQL IMMUTABLE;



-- count system advisories according to advisory type
CREATE OR REPLACE FUNCTION system_advisories_count(system_id_in INT, advisory_type_id_in INT DEFAULT NULL)
    RETURNS INT AS
$system_advisories_count$
DECLARE
    result_cnt INT;
BEGIN
    SELECT COUNT(advisory_id)
    FROM system_advisories sa
             JOIN advisory_metadata am ON sa.advisory_id = am.id
    WHERE (am.advisory_type_id = advisory_type_id_in OR advisory_type_id_in IS NULL)
      AND sa.system_id = system_id_in
      AND sa.when_patched IS NULL
    INTO result_cnt;
    RETURN result_cnt;
END;
$system_advisories_count$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION on_system_update()
    RETURNS TRIGGER
AS
$system_update$
BEGIN
    IF TG_OP != 'UPDATE' OR NEW.last_evaluation IS NULL THEN
        RETURN NEW;
    END IF;

    -- Nothing changed
    IF OLD.opt_out = NEW.opt_out THEN
        RETURN NEW;
    END IF;

    IF NEW.opt_out = TRUE THEN

        WITH to_update_advisories AS (
            SELECT ead.advisory_id, ternary(ead.status_id != sa.status_id, 1, 0) as divergent
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
             update AS (
                 UPDATE advisory_account_data ead
                     SET systems_affected = systems_affected - 1,
                         systems_status_divergent = systems_status_divergent - ta.divergent
                     FROM to_update_advisories ta
                     WHERE ead.advisory_id = to_update_advisories.advisory_id AND
                           ead.rh_account_id = NEW.rh_account_id
             )
        DELETE
        FROM advisory_account_data
        WHERE rh_account_id = NEW.rh_account_id
          AND systems_affected = 0;

    ELSIF NEW.opt_out = FALSE THEN
        -- increment affected advisory counts for system
        WITH to_update_advisories AS (
            SELECT ead.advisory_id, ternary(ead.status_id != sa.status_id, 1, 0) as divergent
            FROM advisory_account_data ead
                     INNER JOIN system_advisories sa
                                ON ead.advisory_id = sa.advisory_id
            WHERE ead.rh_account_id = NEW.rh_account_id
              AND sa.system_id = NEW.id
              AND sa.when_patched IS NULL
            ORDER BY ead.advisory_id FOR
                UPDATE OF ead
            -- increment systems_affected and systems_status_divergent in case status is different
        ),
             update as (
                 -- increment only systems_affected in case status is same
                 UPDATE advisory_account_data ead
                     SET systems_affected = systems_affected + 1,
                         systems_status_divergent = systems_status_divergent + ta.divergent
                     FROM to_update_advisories ta
                     WHERE ead.advisory_id = to_update_advisories.advisory_id
                         AND ead.rh_account_id = NEW.rh_account_id)
        INSERT
        INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
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
END;
$system_update$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_advisory_caches(advisory_id_in INTEGER DEFAULT NULL,
                                                   rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_advisory$
BEGIN
    WITH locked_rows AS (
        SELECT ead.rh_account_id, ead.advisory_id
        FROM advisory_account_data ead
        WHERE (ead.advisory_id = advisory_id_in OR advisory_id_in IS NULL)
          AND (ead.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
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
               AND (sa.advisory_id = advisory_id_in OR advisory_id_in IS NULL)
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
      AND (advisory_id = advisory_id_in OR advisory_id_in IS NULL)
      AND (rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);
END;
$refresh_advisory$ language plpgsql;

CREATE OR REPLACE FUNCTION refresh_system_caches(system_id_in INTEGER DEFAULT NULL,
                                                 rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_system$
BEGIN
    WITH to_update_systems AS (
        SELECT sp.id
        FROM system_platform sp
        WHERE (sp.id = system_id_in OR system_id_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        ORDER BY sp.rh_account_id, sp.id FOR UPDATE OF sp
    )
    UPDATE system_platform sp
    SET advisory_count_cache     = system_advisories_count(sp.id, NULL),
        advisory_enh_count_cache = system_advisories_count(sp.id, 1),
        advisory_bug_count_cache = system_advisories_count(sp.id, 2),
        advisory_sec_count_cache = system_advisories_count(sp.id, 3)
    FROM to_update_systems;
END;
$refresh_system$ LANGUAGE plpgsql;

-- update system advisories counts (all and according types)
CREATE OR REPLACE FUNCTION update_system_caches(system_id_in INT)
    RETURNS VOID AS
$update_system_caches$
BEGIN
    PERFORM refresh_system_caches(system_id_in, NULL);
END;
$update_system_caches$
    LANGUAGE 'plpgsql';

-- refresh_all_cached_counts
-- WARNING: executing this procedure takes long time,
--          use only when necessary, e.g. during upgrade to populate initial caches
CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    PERFORM refresh_system_caches(NULL, NULL);
    PERFORM refresh_advisory_caches(NULL, NULL);
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

    PERFORM refresh_system_caches(NULL, rh_account_id_in);
    PERFORM refresh_advisory_caches(NULL, rh_account_id_in);
END;
$refresh_account_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_advisory_cached_counts(advisory_name varchar)
    RETURNS void AS
$refresh_advisory_cached_counts$
DECLARE
    advisory_id_id INT;
BEGIN
    -- update system count for advisory
    SELECT id FROM advisory_metadata WHERE name = advisory_name INTO advisory_id_id;

    PERFORM refresh_advisory_caches(advisory_id_id, NULL);
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

    PERFORM refresh_advisory_caches(advisory_md_id, rh_account_id_in);
END;
$refresh_advisory_account_cached_counts$
    LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION refresh_system_cached_counts(inventory_id_in varchar)
    RETURNS void AS
$refresh_system_cached_counts$
DECLARE
    system_id int;
BEGIN

    SELECT id FROM system_platform WHERE inventory_id = inventory_id_in INTO system_id;

    PERFORM refresh_system_caches(system_id, NULL);
END;
$refresh_system_cached_counts$ LANGUAGE 'plpgsql';


CREATE TRIGGER system_platform_on_update
    AFTER UPDATE OF opt_out
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE on_system_update();