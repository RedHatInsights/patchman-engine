DROP FUNCTION IF EXISTS system_advisories_count(system_id_in BIGINT, advisory_type_id_in INT) CASCADE;

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

    -- find advisories linked to the server and lock them
    WITH to_update_advisories AS (
        SELECT aad.advisory_id,
               aad.rh_account_id,
               -- Desired count depends on old count + change
               aad.systems_installable + case when sa.status_id = 0 then change else 0 end as systems_installable_dst,
               aad.systems_applicable + case when sa.status_id = 1 then change else 0 end as systems_applicable_dst
          FROM advisory_account_data aad
          JOIN system_advisories sa ON aad.advisory_id = sa.advisory_id
          -- Filter advisory_account_data only for advisories affectign this system & belonging to system account
         WHERE aad.rh_account_id =  NEW.rh_account_id
           AND sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
         ORDER BY aad.advisory_id FOR UPDATE OF aad),
         -- Where count > 0, update existing rows
         update AS (
            UPDATE advisory_account_data aad
               SET systems_installable = ta.systems_installable_dst,
                   systems_applicable = ta.systems_applicable_dst
              FROM to_update_advisories ta
             WHERE aad.advisory_id = ta.advisory_id
               AND aad.rh_account_id = NEW.rh_account_id
               AND (ta.systems_installable_dst > 0 OR ta.systems_applicable_dst > 0)
         ),
         -- Where count = 0, delete existing rows
         delete AS (
            DELETE
              FROM advisory_account_data aad
             USING to_update_advisories ta
             WHERE aad.rh_account_id = ta.rh_account_id
               AND aad.advisory_id = ta.advisory_id
               AND ta.systems_installable_dst <= 0
               AND ta.systems_applicable_dst <= 0
         )
    -- If we have system affected && no exisiting advisory_account_data entry, we insert new rows
    INSERT
      INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
    SELECT sa.advisory_id, NEW.rh_account_id,
           case when sa.status_id = 0 then 1 else 0 end as systems_installable,
           case when sa.status_id = 1 then 1 else 0 end as systems_applicable
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
        SELECT sa.advisory_id, sa.rh_account_id,
               count(sa.*) filter (where sa.status_id = 0) as systems_installable,
               count(sa.*) filter (where sa.status_id = 1) as systems_applicable
          FROM system_advisories sa
          JOIN system_platform sp
            ON sa.rh_account_id = sp.rh_account_id AND sa.system_id = sp.id
         WHERE sp.last_evaluation IS NOT NULL
           AND sp.stale = FALSE
           AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
           AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY sa.advisory_id, sa.rh_account_id
    ),
        upserted AS (
            INSERT INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
                 SELECT advisory_id, rh_account_id, systems_installable, systems_applicable
                   FROM current_counts
            ON CONFLICT (advisory_id, rh_account_id) DO UPDATE SET
                     systems_installable = EXCLUDED.systems_installable,
                     systems_applicable = EXCLUDED.systems_applicable
         )
    DELETE FROM advisory_account_data
     WHERE (advisory_id, rh_account_id) NOT IN (SELECT advisory_id, rh_account_id FROM current_counts)
       AND (advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
       AND (rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);
END;
$refresh_advisory$ language plpgsql;
