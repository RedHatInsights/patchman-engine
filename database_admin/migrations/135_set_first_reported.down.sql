CREATE OR REPLACE FUNCTION set_first_reported()
    RETURNS TRIGGER AS
$set_first_reported$
BEGIN
    IF NEW.first_reported IS NULL THEN
        NEW.first_reported := CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$set_first_reported$
    LANGUAGE 'plpgsql';

ALTER TABLE system_advisories ALTER COLUMN first_reported DROP DEFAULT;

SELECT create_table_partition_triggers('system_advisories_set_first_reported',
                                       $$BEFORE INSERT$$,
                                       'system_advisories',
                                       $$FOR EACH ROW EXECUTE PROCEDURE set_first_reported()$$);
