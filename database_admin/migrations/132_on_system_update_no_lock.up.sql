CREATE OR REPLACE FUNCTION on_system_update()
-- this trigger updates advisory_account_data when server changes its stale flag
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

    was_counted := OLD.stale = FALSE;
    should_count := NEW.stale = FALSE;

    -- Determine what change we are performing
    IF was_counted and NOT should_count THEN
        change := -1;
    ELSIF NOT was_counted AND should_count THEN
        change := 1;
    ELSE
        -- No change
        RETURN NEW;
    END IF;

    -- find advisories linked to the server
    WITH to_update_advisories AS (
        SELECT aad.advisory_id,
               aad.rh_account_id,
               case when sa.status_id = 0 then change else 0 end as systems_installable_change,
               change as systems_applicable_change
          FROM advisory_account_data aad
          JOIN system_advisories sa ON aad.advisory_id = sa.advisory_id
          -- Filter advisory_account_data only for advisories affectign this system & belonging to system account
         WHERE aad.rh_account_id =  NEW.rh_account_id
           AND sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
         ORDER BY aad.advisory_id),
         -- update existing rows
         update AS (
            UPDATE advisory_account_data aad
               SET systems_installable = aad.systems_installable + ta.systems_installable_change,
                   systems_applicable = aad.systems_applicable + ta.systems_applicable_change
              FROM to_update_advisories ta
             WHERE aad.advisory_id = ta.advisory_id
               AND aad.rh_account_id = NEW.rh_account_id
         )
    -- If we have system affected && no exisiting advisory_account_data entry, we insert new rows
    INSERT
      INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
    SELECT sa.advisory_id, NEW.rh_account_id,
           case when sa.status_id = 0 then 1 else 0 end as systems_installable,
           1 as systems_applicable
    FROM system_advisories sa
    WHERE sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
      AND change > 0
      -- create only rows which are not already in to_update_advisories
      AND (NEW.rh_account_id, sa.advisory_id) NOT IN (
            SELECT ta.rh_account_id, ta.advisory_id
              FROM to_update_advisories ta
    )
    ON CONFLICT (advisory_id, rh_account_id) DO UPDATE
        SET systems_installable = advisory_account_data.systems_installable + EXCLUDED.systems_installable,
            systems_applicable = advisory_account_data.systems_applicable + EXCLUDED.systems_applicable;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

-- indexes for filtering systems_applicable, systems_installable
CREATE INDEX ON advisory_account_data (systems_applicable);
CREATE INDEX ON advisory_account_data (systems_installable);
