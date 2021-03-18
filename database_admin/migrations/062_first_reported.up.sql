-- Let's create generic function to remove trigger from all partitions. It can be used in future too.
CREATE OR REPLACE FUNCTION drop_table_partition_triggers(name text, trig_type text, tbl regclass, trig_text text)
    RETURNS VOID AS
$$
DECLARE
    r record;
    trig_name text;
BEGIN
    FOR r IN SELECT child.relname
               FROM pg_inherits
               JOIN pg_class parent
                 ON pg_inherits.inhparent = parent.oid
               JOIN pg_class child
                 ON pg_inherits.inhrelid   = child.oid
              WHERE parent.relname = text(tbl)
    LOOP
        trig_name := name || substr(r.relname, length(text(tbl)) +1 );
        EXECUTE 'DROP TRIGGER IF EXISTS ' || trig_name || ' ON ' || r.relname;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Remove triggers setting "first_reported" value before inserting new row to "system_platform"
-- so we can delete "first_reported" column then.
SELECT drop_table_partition_triggers('system_platform_set_first_reported',
                                       $$BEFORE INSERT$$,
                                       'system_platform',
                                       $$FOR EACH ROW EXECUTE PROCEDURE set_first_reported()$$);

ALTER TABLE system_platform DROP COLUMN IF EXISTS first_reported;
