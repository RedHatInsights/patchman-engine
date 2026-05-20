DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_attribute a
                    JOIN pg_class c ON c.oid = a.attrelid
                    JOIN pg_namespace n ON n.oid = c.relnamespace
                    WHERE n.nspname = 'public' AND c.relname = 'system_inventory' AND a.attname = 'workspace_id'
                          AND a.attnum > 0 AND NOT a.attisdropped)
THEN
    ALTER TABLE system_inventory
        ADD COLUMN workspaces JSONB;

    UPDATE system_inventory
        SET workspaces = JSONB_BUILD_ARRAY(JSONB_BUILD_OBJECT('id', workspace_id, 'name', workspace_name));

    CREATE INDEX IF NOT EXISTS system_inventory_workspaces_index ON system_inventory USING GIN (workspaces);

    ALTER TABLE system_inventory
        DROP COLUMN workspace_id,
        DROP COLUMN workspace_name;
END IF;
END $$;

