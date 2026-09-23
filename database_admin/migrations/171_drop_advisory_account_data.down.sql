-- advisory_account_data
CREATE TABLE IF NOT EXISTS advisory_account_data
(
    advisory_id              BIGINT NOT NULL,
    rh_account_id            INT NOT NULL,
    systems_applicable       INT NOT NULL DEFAULT 0,
    systems_installable      INT NOT NULL DEFAULT 0,
    notified                 TIMESTAMP WITH TIME ZONE NULL,
    CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
    UNIQUE (advisory_id, rh_account_id),
    PRIMARY KEY (rh_account_id, advisory_id)
) WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
  TABLESPACE pg_default;

-- manager user needs to change this table for opt-out functionality
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO manager;
-- evaluator user needs to change this table
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO evaluator;
-- listner user needs to change this table when deleting system
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO listener;
-- vmaas_sync needs to update stale mark, which creates and deletes advisory_account_data
GRANT SELECT, INSERT, UPDATE, DELETE ON advisory_account_data TO vmaas_sync;

-- indexes for filtering systems_applicable, systems_installable
CREATE INDEX ON advisory_account_data (systems_applicable);
CREATE INDEX ON advisory_account_data (systems_installable);
GRANT DELETE ON advisory_account_data TO vmaas_sync;

CREATE OR REPLACE FUNCTION backfill_account_advisory(rh_account_id_in INTEGER)
    RETURNS VOID AS
$backfill$
BEGIN
    PERFORM refresh_account_advisory_caches_multi(NULL, rh_account_id_in);

    -- copy `notified` for all `workspace_id`s per account
    UPDATE account_advisory aa
    SET notified = aad.notified
        FROM advisory_account_data aad
    WHERE aa.advisory_id = aad.advisory_id
        AND aa.rh_account_id = aad.rh_account_id
        AND aa.rh_account_id = rh_account_id_in
        AND aad.notified IS NOT NULL;
END;
$backfill$ LANGUAGE plpgsql;

ALTER TABLE rh_account ADD COLUMN valid_advisory_cache BOOLEAN NOT NULL DEFAULT FALSE;

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
               count(sa.*) as systems_applicable
          FROM system_advisories sa
          JOIN system_inventory si
            ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id
          JOIN system_patch sp
            ON si.id = sp.system_id AND sp.rh_account_id = si.rh_account_id
         WHERE sp.last_evaluation IS NOT NULL
           AND si.stale = FALSE
           AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
           AND (si.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
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

CREATE OR REPLACE FUNCTION refresh_advisory_caches(advisory_id_in INTEGER DEFAULT NULL,
                                                   rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_advisory$
BEGIN
    IF advisory_id_in IS NOT NULL THEN
        PERFORM refresh_advisory_caches_multi(ARRAY [advisory_id_in], rh_account_id_in);
    ELSE
        PERFORM refresh_advisory_caches_multi(NULL, rh_account_id_in);
    END IF;
END;
$refresh_advisory$ language plpgsql;

-- Not adding the following functions, because it was not used anywhere:
-- - refresh_advisory_cached_counts(advisory_name varchar)
-- - refresh_advisory_account_cached_counts(advisory_name varchar, rh_account_name varchar)
-- - refresh_account_cached_counts(rh_account_in varchar)

CREATE OR REPLACE FUNCTION refresh_all_cached_counts()
    RETURNS void AS
$refresh_all_cached_counts$
BEGIN
    PERFORM refresh_system_caches(NULL, NULL);
    PERFORM refresh_advisory_caches(NULL, NULL);
END;
$refresh_all_cached_counts$
    LANGUAGE 'plpgsql';

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
    IF TG_OP != 'UPDATE' OR NOT EXISTS (
        SELECT 1
        FROM system_patch
        WHERE system_id = NEW.id 
          AND rh_account_id = NEW.rh_account_id
          AND last_evaluation IS NOT NULL
    ) THEN
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

    -- insert/update advisories linked to the server
    INSERT
      INTO advisory_account_data (advisory_id, rh_account_id, systems_installable, systems_applicable)
    SELECT sa.advisory_id, NEW.rh_account_id,
           case when sa.status_id = 0 then change else 0 end as systems_installable,
           change as systems_applicable
      FROM system_advisories sa
     WHERE sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
     ORDER BY sa.advisory_id
        ON CONFLICT (advisory_id, rh_account_id) DO UPDATE
           SET systems_installable = advisory_account_data.systems_installable + EXCLUDED.systems_installable,
               systems_applicable = advisory_account_data.systems_applicable + EXCLUDED.systems_applicable;
    RETURN NEW;
END;
$system_update$ LANGUAGE plpgsql;

SELECT create_table_partition_triggers('system_inventory_on_update',
                                       $$AFTER UPDATE$$,
                                       'system_inventory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE on_system_update()$$);
