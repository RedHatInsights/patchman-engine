-- Add "first_reported" column, without "NOT NULL" constraint firstly".
ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS
        first_reported TIMESTAMP WITH TIME ZONE;

-- Add triggers to setup "first_reported" value before inserting new row to "system_platform" table.
SELECT create_table_partition_triggers('system_platform_set_first_reported',
                                       $$BEFORE INSERT$$,
                                       'system_platform',
                                       $$FOR EACH ROW EXECUTE PROCEDURE set_first_reported()$$);

-- For existing rows set "first_reported" value from "last_updated".
UPDATE system_platform SET first_reported = last_updated;

-- Now we can add "NOT NULL" constraint to the "first_reported" column.
ALTER TABLE system_platform
    ALTER COLUMN first_reported SET NOT NULL;

-- Drop generic function created in last migration.
DROP FUNCTION IF EXISTS drop_table_partition_triggers(name text, trig_type text, tbl regclass, trig_text text);
