-- index
DROP INDEX IF EXISTS system_inventory_workspaces_index;

-- rename the old workspaces field to groups
ALTER TABLE system_inventory DROP COLUMN workspaces;

-- add the new workspaces field
ALTER TABLE system_inventory ADD COLUMN workspaces JSONB;

-- copy the data from old inventory_hosts.groups to new workspaces field
UPDATE system_inventory si
SET workspaces = ih.groups
FROM inventory.hosts ih
WHERE ih.id = si.inventory_id;

-- create index on new workspaces field
CREATE INDEX IF NOT EXISTS system_inventory_workspaces_index ON system_inventory USING GIN (workspaces);
