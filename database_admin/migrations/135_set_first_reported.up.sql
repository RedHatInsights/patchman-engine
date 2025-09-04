ALTER TABLE system_advisories ALTER COLUMN first_reported SET DEFAULT CURRENT_TIMESTAMP;

select drop_table_partition_triggers('system_advisories_set_first_reported', '', 'system_advisories', '');

DROP FUNCTION IF EXISTS set_first_reported();

