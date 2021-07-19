CREATE TABLE IF NOT EXISTS package_account_data
(
    name_id           INT NOT NULL,
    rh_account_id     INT NOT NULL,
    systems_installed INT NOT NULL DEFAULT 0,
    systems_updatable INT NOT NULL DEFAULT 0,

    CONSTRAINT package_name_id
        FOREIGN KEY (name_id) REFERENCES package_name (id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
    PRIMARY KEY (rh_account_id, name_id)
);


GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO listener;
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO vmaas_sync;


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
          AND sa.rh_account_id = NEW.rh_account_id
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
                     WHERE aad.rh_account_id = ta.rh_account_id
                         AND aad.advisory_id = ta.advisory_id
                         AND ta.systems_affected_dst <= 0
         )
         -- If we have system affected && no exisiting advisory_account_data entry, we insert new rows
    INSERT
    INTO advisory_account_data (advisory_id, rh_account_id, systems_affected)
    SELECT sa.advisory_id, NEW.rh_account_id, 1
    FROM system_advisories sa
    WHERE sa.system_id = NEW.id
      AND sa.rh_account_id = NEW.rh_account_id
      AND sa.when_patched IS NULL
      AND change > 0
      -- We system_advisory pairs which don't already have rows in to_update_advisories
      AND (NEW.rh_account_id, sa.advisory_id) NOT IN (
        SELECT ta.rh_account_id, ta.advisory_id
        FROM to_update_advisories ta
    )
    ON CONFLICT (advisory_id, rh_account_id)
        DO UPDATE SET systems_affected = advisory_account_data.systems_affected + EXCLUDED.systems_affected;

    -- Update package cache table
    WITH to_update_packages AS (
        SELECT pad.name_id,
               pad.rh_account_id,
               pad.systems_installed + change                         as systems_installed_dst,
               pad.systems_updatable
                   + ternary(spkg.latest_evra IS NOT NULL, change, 0) as systems_updatable_dst
        FROM package_account_data pad
                 INNER JOIN system_package spkg on pad.name_id = spkg.name_id
        WHERE pad.rh_account_id = NEW.rh_account_id
          AND spkg.system_id = NEW.id
          AND spkg.rh_account_id = NEW.rh_account_id
        ORDER BY pad.name_id FOR UPDATE OF pad
    ),
         update as (
             UPDATE package_account_data pad
                 SET systems_installed = ta.systems_installed_dst,
                     systems_updatable = ta.systems_updatable_dst
                 FROM to_update_packages ta
                 WHERE pad.name_id = ta.name_id
                     AND pad.rh_account_id = NEW.rh_account_id
                     AND ta.systems_installed_dst > 0
         ),
         delete AS (
             DELETE FROM package_account_data pad
                 USING to_update_packages ta
                 WHERE pad.rh_account_id = ta.rh_account_id
                     AND pad.name_id = ta.name_id
                     AND ta.systems_installed_dst <= 0
         )
    INSERT
    INTO package_account_data (name_id, rh_account_id, systems_installed, systems_updatable)
    SELECT spkg.name_id, NEW.rh_account_id, 1
    FROM system_package spkg
    WHERE spkg.system_id = NEW.id
      AND spkg.rh_account_id = NEW.rh_account_id
      AND change > 0
      AND (NEW.rh_account_id, spkg.name_id) NOT IN (
        SELECT ta.rh_account_id, ta.name_id
        FROM to_update_packages ta
    )
    ON CONFLICT (name_id, rh_account_id)
        DO UPDATE SET systems_installed = package_account_data.systems_installed + excluded.systems_installed,
                      systems_updatable = package_account_data.systems_updatable + excluded.systems_updatable;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;



CREATE OR REPLACE FUNCTION refresh_package_caches_multi(name_ids_in INTEGER[] DEFAULT NULL,
                                                        rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_packages$
BEGIN
    -- Lock rows
    PERFORM pad.rh_account_id, pad.name_id
    FROM package_account_data pad
    WHERE (pad.name_id = ANY (name_ids_in) OR name_ids_in IS NULL)
      AND (pad.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        FOR UPDATE OF pad;

    WITH current_counts AS (
        SELECT spkg.name_id,
               sp.rh_account_id,
               count(spkg.system_id)                                    as systems_installed,
               count(spkg.system_id WHERE spkg.latest_evra IS NOT NULL) as systems_updatable
        FROM system_package spkg
                 INNER JOIN system_platform sp
                            ON spkg.rh_account_id = sp.rh_account_id
                                AND spkg.system_id = sp.id
        WHERE sp.last_evaluation IS NOT NULL
          AND sp.stale = FALSE
          AND (spkg.name_id = ANY (name_ids_in) OR name_ids_in IS NULL)
          AND (sp.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        GROUP BY spkg.name_id, sp.rh_account_id
    ),
         upserted AS (
             INSERT INTO package_account_data (name_id, rh_account_id, systems_installed, systems_updatable)
                 SELECT name_id, rh_account_id, systems_installed, systems_updatable
                 FROM current_counts
                 ON CONFLICT (name_id, rh_account_id) DO UPDATE SET
                     systems_installed = EXCLUDED.systems_installed,
                     systems_updatable = EXCLUDED > systems_updatable
         )
    DELETE
    FROM package_account_data
    WHERE (name_id, rh_account_id) NOT IN (SELECT name_id, rh_account_id FROM current_counts)
      AND (name_id = ANY (name_ids_in) OR name_ids_in IS NULL)
      AND (rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);
END;
$refresh_packages$ language plpgsql;
