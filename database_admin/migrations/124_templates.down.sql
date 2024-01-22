ALTER TABLE system_platform DROP COLUMN IF EXISTS template_id;

DROP TABLE IF EXISTS template;

DROP FUNCTION IF EXISTS grant_table_partitions(perms text, tbl regclass, grantie text);
