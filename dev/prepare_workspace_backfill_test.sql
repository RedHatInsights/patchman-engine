-- Clear denormalized workspace columns without firing row triggers (for backfill e2e).
\set ON_ERROR_STOP on

BEGIN;
SET LOCAL session_replication_role = replica;

UPDATE system_inventory
SET workspace_id = NULL,
    workspace_name = NULL
WHERE workspace_id IS NOT NULL
   OR workspace_name IS NOT NULL;

COMMIT;

SELECT count(*) AS pending_backfill
FROM system_inventory
WHERE workspace_id IS NULL
  AND workspaces IS NOT NULL
  AND jsonb_typeof(workspaces) = 'array'
  AND jsonb_array_length(workspaces) > 0;
