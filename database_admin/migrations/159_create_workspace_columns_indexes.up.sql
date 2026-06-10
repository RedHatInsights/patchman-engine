SELECT drop_table_partition_triggers('system_inventory_sync_workspace',
                                     $$BEFORE INSERT OR UPDATE OF workspaces$$,
                                     'system_inventory',
                                     $$FOR EACH ROW EXECUTE PROCEDURE sync_system_inventory_workspace()$$);

DROP FUNCTION IF EXISTS sync_system_inventory_workspace();

CREATE INDEX IF NOT EXISTS system_inventory_workspace_id_index ON system_inventory (workspace_id);
