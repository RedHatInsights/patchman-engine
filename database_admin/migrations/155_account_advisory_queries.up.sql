CREATE INDEX ON account_advisory (systems_applicable);
CREATE INDEX ON account_advisory (systems_installable);

CREATE OR REPLACE FUNCTION on_system_update_account_advisory()
    RETURNS TRIGGER
AS
$system_update_account_advisory$
DECLARE
    was_counted       BOOLEAN;
    should_count      BOOLEAN;
    workspace_changed BOOLEAN;
BEGIN
    IF TG_OP != 'UPDATE' OR NOT EXISTS (
        SELECT 1
        FROM system_patch
        WHERE system_id = NEW.id
          AND rh_account_id = NEW.rh_account_id
          AND last_evaluation IS NOT NULL
    ) THEN
        RETURN NEW;
    END IF;

    was_counted := OLD.stale = FALSE AND OLD.workspace_id IS NOT NULL;
    should_count := NEW.stale = FALSE AND NEW.workspace_id IS NOT NULL;
    workspace_changed := OLD.workspace_id IS DISTINCT FROM NEW.workspace_id;

    -- Decrement from old workspace
    IF was_counted AND (NOT should_count OR workspace_changed) THEN
        INSERT
          INTO account_advisory (advisory_id, rh_account_id, workspace_id, systems_installable, systems_applicable)
        SELECT sa.advisory_id, OLD.rh_account_id, OLD.workspace_id,
               CASE WHEN sa.status_id = 0 THEN -1 ELSE 0 END,
               -1
          FROM system_advisories sa
         WHERE sa.system_id = OLD.id AND sa.rh_account_id = OLD.rh_account_id
         ORDER BY sa.advisory_id
            ON CONFLICT (rh_account_id, workspace_id, advisory_id) DO UPDATE
               SET systems_installable = account_advisory.systems_installable + EXCLUDED.systems_installable,
                   systems_applicable = account_advisory.systems_applicable + EXCLUDED.systems_applicable;
    END IF;

    -- Increment in new workspace
    IF should_count AND (NOT was_counted OR workspace_changed) THEN
        INSERT
          INTO account_advisory (advisory_id, rh_account_id, workspace_id, systems_installable, systems_applicable)
        SELECT sa.advisory_id, NEW.rh_account_id, NEW.workspace_id,
               CASE WHEN sa.status_id = 0 THEN 1 ELSE 0 END,
               1
          FROM system_advisories sa
         WHERE sa.system_id = NEW.id AND sa.rh_account_id = NEW.rh_account_id
         ORDER BY sa.advisory_id
            ON CONFLICT (rh_account_id, workspace_id, advisory_id) DO UPDATE
               SET systems_installable = account_advisory.systems_installable + EXCLUDED.systems_installable,
                   systems_applicable = account_advisory.systems_applicable + EXCLUDED.systems_applicable;
    END IF;

    RETURN NEW;
END;
$system_update_account_advisory$ LANGUAGE plpgsql;

SELECT create_table_partition_triggers('system_inventory_on_update_account_advisory',
                                       $$AFTER UPDATE$$,
                                       'system_inventory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE on_system_update_account_advisory()$$);

CREATE OR REPLACE FUNCTION refresh_account_advisory_caches_multi(advisory_ids_in INTEGER[] DEFAULT NULL,
                                                                  rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_account_advisory$
BEGIN
    PERFORM aa.rh_account_id, aa.workspace_id, aa.advisory_id
    FROM account_advisory aa
    WHERE (aa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
      AND (aa.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
        FOR UPDATE OF aa;

    WITH current_counts AS (
        SELECT sa.advisory_id, sa.rh_account_id, si.workspace_id,
               count(sa.*) FILTER (WHERE sa.status_id = 0) AS systems_installable,
               count(sa.*) AS systems_applicable
          FROM system_advisories sa
          JOIN system_inventory si
            ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id
          JOIN system_patch sp
            ON si.id = sp.system_id AND sp.rh_account_id = si.rh_account_id
         WHERE sp.last_evaluation IS NOT NULL
           AND si.stale = FALSE
           AND si.workspace_id IS NOT NULL
           AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
           AND (si.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY sa.advisory_id, sa.rh_account_id, si.workspace_id
    ),
        upserted AS (
            INSERT INTO account_advisory (advisory_id, rh_account_id, workspace_id, systems_installable, systems_applicable)
                 SELECT advisory_id, rh_account_id, workspace_id, systems_installable, systems_applicable
                   FROM current_counts
            ON CONFLICT (rh_account_id, workspace_id, advisory_id) DO UPDATE SET
                     systems_installable = EXCLUDED.systems_installable,
                     systems_applicable = EXCLUDED.systems_applicable
         )
    DELETE FROM account_advisory
     WHERE (advisory_id, rh_account_id, workspace_id) NOT IN (SELECT advisory_id, rh_account_id, workspace_id FROM current_counts)
       AND (advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
       AND (rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL);
END;
$refresh_account_advisory$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_account_advisory_caches(advisory_id_in INTEGER DEFAULT NULL,
                                                            rh_account_id_in INTEGER DEFAULT NULL)
    RETURNS VOID AS
$refresh_account_advisory$
BEGIN
    IF advisory_id_in IS NOT NULL THEN
        PERFORM refresh_account_advisory_caches_multi(ARRAY [advisory_id_in], rh_account_id_in);
    ELSE
        PERFORM refresh_account_advisory_caches_multi(NULL, rh_account_id_in);
    END IF;
END;
$refresh_account_advisory$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION backfill_account_advisory(rh_account_id_in INTEGER)
    RETURNS VOID AS
$backfill$
BEGIN
    PERFORM refresh_account_advisory_caches_multi(NULL, rh_account_id_in);
END;
$backfill$ LANGUAGE plpgsql;
