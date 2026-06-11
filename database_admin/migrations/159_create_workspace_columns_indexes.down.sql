DROP INDEX IF EXISTS system_inventory_workspace_id_index;

CREATE OR REPLACE FUNCTION sync_system_inventory_workspace()
    RETURNS TRIGGER AS
$$
BEGIN
    IF NEW.workspaces IS NOT NULL
        AND jsonb_typeof(NEW.workspaces) = 'array'
        AND jsonb_array_length(NEW.workspaces) > 0
    THEN
        NEW.workspace_id := (NEW.workspaces->0->>'id')::UUID;
        NEW.workspace_name := NEW.workspaces->0->>'name';
        IF NEW.workspace_name IS NULL OR empty(NEW.workspace_name) THEN
            RAISE EXCEPTION 'workspace_name must not be empty';
        END IF;
    ELSIF TG_OP = 'INSERT' THEN
        RAISE EXCEPTION 'workspaces required';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

SELECT create_table_partition_triggers('system_inventory_sync_workspace',
                                       $$BEFORE INSERT OR UPDATE OF workspaces$$,
                                       'system_inventory',
                                       $$FOR EACH ROW EXECUTE PROCEDURE sync_system_inventory_workspace()$$);
