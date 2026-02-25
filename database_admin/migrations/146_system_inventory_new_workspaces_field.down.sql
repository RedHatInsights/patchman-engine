-- index
DROP INDEX IF EXISTS system_inventory_workspaces_index;

-- drop the new workspaces field
ALTER TABLE system_inventory DROP COLUMN workspaces;

-- rename the old workspaces field to groups
ALTER TABLE system_inventory ADD COLUMN workspaces TEXT ARRAY CHECK (array_length(workspaces,1) > 0 or workspaces is null);

UPDATE system_inventory si
SET workspaces = ARRAY(SELECT jsonb_array_elements(ih.groups)->>'id'),
FROM inventory.hosts ih
WHERE ih.id = si.inventory_id;

-- create index on new workspaces field
CREATE INDEX IF NOT EXISTS system_inventory_workspaces_index ON system_inventory USING GIN (workspaces);
