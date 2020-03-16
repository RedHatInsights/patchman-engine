ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS stale_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS stale_warning_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS culled_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS stale BOOLEAN NOT NULL DEFAULT FALSE;

CREATE OR REPLACE FUNCTION delete_culled_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    culled integer;
BEGIN
    select count(*)
    from (
             select delete_system(inventory_id)
             from system_platform
             where culled_timestamp < now()
         ) t
    INTO culled;
    RETURN culled;
END;
$fun$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION mark_stale_systems()
    RETURNS INTEGER
AS
$fun$
DECLARE
    marked integer;
BEGIN
    with updated as (UPDATE system_platform
        SET stale = true
        WHERE stale_timestamp < now()
        RETURNING id
    )
    select count(*)
    from updated
    INTO marked;
    return marked;
END;
$fun$ LANGUAGE plpgsql;

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
               AND sp.stale = FALSE
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

CREATE OR REPLACE FUNCTION on_system_update()
    RETURNS TRIGGER
AS
$system_update$
DECLARE
    was_counted  BOOLEAN;
    should_count BOOLEAN;
BEGIN
    IF TG_OP != 'UPDATE' OR NEW.last_evaluation IS NULL THEN
        RETURN NEW;
    END IF;

    was_counted := OLD.opt_out = FALSE AND OLD.stale = FALSE;
    should_count := NEW.opt_out = FALSE AND NEW.stale = FALSE;

    -- Nothing changed
    IF was_counted = should_count THEN
        RETURN NEW;
    END IF;

    IF was_counted = TRUE AND should_count = FALSE THEN

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
                     WHERE ead.advisory_id = ta.advisory_id AND
                           ead.rh_account_id = NEW.rh_account_id
             )
        DELETE
        FROM advisory_account_data
        WHERE rh_account_id = NEW.rh_account_id
          AND systems_affected = 0;

    ELSIF was_counted = FALSE AND should_count = TRUE THEN
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
                     WHERE ead.advisory_id = ta.advisory_id
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
    ELSE
        RAISE EXCEPTION 'Shouldnt happen';
    END IF;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS system_platform_on_update on system_platform;

CREATE TRIGGER system_platform_on_update
    AFTER UPDATE OF opt_out, stale
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE on_system_update();
