-- Drop the trigger just to be safe
DROP TRIGGER IF EXISTS system_platform_on_update on system_platform;



CREATE OR REPLACE FUNCTION on_system_update()
    RETURNS TRIGGER
AS
$system_update$
DECLARE
    was_counted  BOOLEAN;
    should_count BOOLEAN;
    change       INT;
BEGIN
    -- Ignore not yet evaluated systems
    IF TG_OP != 'UPDATE' OR NEW.last_evaluation IS NULL THEN
        RETURN NEW;
    END IF;

    was_counted := OLD.opt_out = FALSE AND OLD.stale = FALSE;
    should_count := NEW.opt_out = FALSE AND NEW.stale = FALSE;

    -- Determine what change we are performing
    IF was_counted and NOT should_count THEN
        change := -1;
    ELSIF NOT was_counted AND should_count THEN
        change := 1;
    ELSE
        -- No change
        RETURN NEW;
    END IF;

    -- Select all changed rows, lock them
    WITH to_update_advisories AS (
        SELECT aad.advisory_id,
               aad.rh_account_id,
               -- Desired count depends on old count + change
               aad.systems_affected + change                                                    as systems_affected_dst,
               -- Divergent count is the same, only depends on advisory_account_data status being different
               aad.systems_status_divergent + ternary(aad.status_id != sa.status_id, change, 0) as divergent
        FROM advisory_account_data aad
                 INNER JOIN system_advisories sa ON aad.advisory_id = sa.advisory_id
             -- Filter advisory_account_data only for advisories affectign this system & belonging to system account
        WHERE aad.rh_account_id = NEW.rh_account_id
          AND sa.system_id = NEW.id
          AND sa.when_patched IS NULL
        ORDER BY aad.advisory_id FOR UPDATE OF aad),
         -- Where count > 0, update existing rows
         update AS (
             UPDATE advisory_account_data aad
                 SET systems_affected = ta.systems_affected_dst,
                     systems_status_divergent = ta.divergent
                 FROM to_update_advisories ta
                 WHERE aad.advisory_id = ta.advisory_id
                     AND aad.rh_account_id = NEW.rh_account_id
                     AND ta.systems_affected_dst > 0
         ),
         -- Where count = 0, delete existing rows
         delete AS (
             DELETE
                 FROM advisory_account_data aad
                     USING to_update_advisories ta
                     WHERE aad.rh_account_id = NEW.rh_account_id
                         AND (aad.rh_account_id, aad.advisory_id) in (
                             SELECT ta.rh_account_id, ta.advisory_id
                             FROM to_update_advisories ta
                             WHERE ta.systems_affected_dst = 0
                         )
         )
         -- If we have system affected && no exisiting advisory_account_data entry, we insert new rows
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
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

CREATE TRIGGER system_platform_on_update
    AFTER UPDATE OF opt_out, stale
    ON system_platform
    FOR EACH ROW
EXECUTE PROCEDURE on_system_update();
