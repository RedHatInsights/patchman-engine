-- Pending predicate must match workspacePendingPredicate in tasks/workspace_backfill/workspace_backfill.go
\set ON_ERROR_STOP on

SELECT count(*) AS pending
FROM system_inventory
WHERE workspace_id IS NULL
  AND workspaces IS NOT NULL
  AND jsonb_typeof(workspaces) = 'array'
  AND jsonb_array_length(workspaces) > 0
  AND workspaces->0->>'id' IS NOT NULL
  AND workspaces->0->>'name' IS NOT NULL
  AND NOT empty(workspaces->0->>'name');

SELECT count(*) AS mismatched
FROM system_inventory
WHERE workspace_id IS NOT NULL
  AND workspaces IS NOT NULL
  AND (
    workspace_id::text IS DISTINCT FROM workspaces->0->>'id'
    OR workspace_name IS DISTINCT FROM workspaces->0->>'name'
  );
