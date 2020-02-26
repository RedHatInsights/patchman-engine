-- Drop the trigger just to be safe
DROP TRIGGER IF EXISTS system_platform_on_update on system_platform;



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
        -- Lock Rows, blocking other transactions which would want to modify affected rows in advisory_account_data
        WITH to_update_advisories AS (
            SELECT aad.advisory_id,
                   aad.rh_account_id,
                   aad.systems_affected - 1                                                    as systems_affected_dst,
                   aad.systems_status_divergent - ternary(aad.status_id != sa.status_id, 1, 0) as divergent
            FROM advisory_account_data aad
                     INNER JOIN system_advisories sa ON aad.advisory_id = sa.advisory_id
            WHERE aad.rh_account_id = NEW.rh_account_id
              AND sa.system_id = NEW.id
              AND sa.when_patched IS NULL
            ORDER BY aad.advisory_id
                FOR UPDATE OF aad
        ),
             -- Update rows where count is not 0, Does overwrite the value, relying on pevious locking to ensure
             -- changes are consistent
             update AS (
                 UPDATE advisory_account_data aad
                     SET systems_affected = ta.systems_affected_dst,
                         systems_status_divergent = ta.divergent
                     FROM to_update_advisories ta
                     WHERE aad.advisory_id = ta.advisory_id
                         AND aad.rh_account_id = NEW.rh_account_id
                         AND ta.systems_affected_dst > 0
             )
             -- Delete rows where count should be 0
             -- This needs to be written this way, and not a straight delete, because per PostgreSQL documentation
             -- All non-depending CTE queries are executed against same DB snapshot, and that means that
             -- Delete stmt will not pick up changes performed by the update, leaving us with rows which have count of 0

        DELETE
        FROM advisory_account_data aad
            USING to_update_advisories ta
        WHERE aad.rh_account_id = NEW.rh_account_id
          AND (aad.rh_account_id, aad.advisory_id) in (
            SELECT ta.rh_account_id, ta.advisory_id
            FROM to_update_advisories ta
            WHERE ta.systems_affected_dst = 0
        );
    ELSIF was_counted = FALSE AND should_count = TRUE THEN
        -- increment affected advisory counts for system, performs locking
        WITH to_update_advisories AS (
            SELECT aad.advisory_id,
                   aad.rh_account_id,
                   aad.systems_affected + 1                                                    as systems_affected_dst,
                   aad.systems_status_divergent + ternary(aad.status_id != sa.status_id, 1, 0) as divergent
            FROM advisory_account_data aad
                     INNER JOIN system_advisories sa ON aad.advisory_id = sa.advisory_id
            WHERE aad.rh_account_id = NEW.rh_account_id
              AND sa.system_id = NEW.id
              AND sa.when_patched IS NULL
            ORDER BY aad.advisory_id FOR UPDATE OF aad
        ),
             update as (
                 -- update rows with result from previous select, which locked them
                 UPDATE advisory_account_data ead
                     SET systems_affected = ta.systems_affected_dst,
                         systems_status_divergent = ta.divergent
                     FROM to_update_advisories ta
                     WHERE ead.advisory_id = ta.advisory_id
                         AND ead.rh_account_id = NEW.rh_account_id)

             -- We can't use `to_update_advisories` rows for insert, because they dont exist
        INSERT
        INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
        SELECT sa.advisory_id, NEW.rh_account_id, 1
        FROM system_advisories sa
        WHERE sa.system_id = NEW.id
          AND sa.when_patched IS NULL
          -- We system_advisory pairs which don't already have rows in to_update_advisories
          AND (NEW.rh_account_id, sa.advisory_id) NOT IN (
            SELECT ta.rh_account_id, ta.advisory_id
            FROM to_update_advisories ta
        )
        ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET systems_affected = advisory_account_data.systems_affected + EXCLUDED.systems_affected;
    ELSE
        RAISE EXCEPTION 'Shouldnt happen';
    END IF;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;


CREATE TRIGGER system_platform_on_update
    AFTER UPDATE OF opt_out, stale
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE on_system_update();
