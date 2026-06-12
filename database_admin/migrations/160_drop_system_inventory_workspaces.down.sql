ALTER TABLE system_inventory ADD COLUMN IF NOT EXISTS workspaces JSONB;

UPDATE system_inventory
   SET workspaces = JSONB_BUILD_ARRAY(JSONB_BUILD_OBJECT('id', workspace_id, 'name', workspace_name))
 WHERE workspace_id IS NOT NULL
   AND workspace_name IS NOT NULL
   AND workspaces IS NULL;

CREATE INDEX IF NOT EXISTS system_inventory_workspaces_index ON system_inventory USING GIN (workspaces);

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
        SELECT sa.advisory_id, sa.rh_account_id, (ws->>'id')::UUID AS workspace_id,
               count(sa.*) FILTER (WHERE sa.status_id = 0) AS systems_installable,
               count(sa.*) AS systems_applicable
          FROM system_advisories sa
          JOIN system_inventory si
            ON sa.rh_account_id = si.rh_account_id AND sa.system_id = si.id
          JOIN system_patch sp
            ON si.id = sp.system_id AND sp.rh_account_id = si.rh_account_id
         CROSS JOIN LATERAL jsonb_array_elements(si.workspaces) AS ws
         WHERE sp.last_evaluation IS NOT NULL
           AND si.stale = FALSE
           AND si.workspaces IS NOT NULL
           AND (sa.advisory_id = ANY (advisory_ids_in) OR advisory_ids_in IS NULL)
           AND (si.rh_account_id = rh_account_id_in OR rh_account_id_in IS NULL)
         GROUP BY sa.advisory_id, sa.rh_account_id, (ws->>'id')::UUID
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
