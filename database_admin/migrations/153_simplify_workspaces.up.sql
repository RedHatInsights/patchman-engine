ALTER TABLE system_inventory
    ADD COLUMN workspace_id UUID,
    ADD COLUMN workspace_name TEXT CHECK (NOT empty(workspace_name));

UPDATE system_inventory
    SET workspace_id = (workspaces->0->>'id')::UUID,
        workspace_name = workspaces->0->>'name';

CREATE INDEX IF NOT EXISTS system_inventory_workspace_id_index ON system_inventory (workspace_id);
CREATE INDEX IF NOT EXISTS system_inventory_workspace_name_index ON system_inventory (workspace_name);
