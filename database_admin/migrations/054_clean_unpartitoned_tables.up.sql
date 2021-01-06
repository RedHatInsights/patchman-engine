CALL raise_notice('MIGRATION 50');

-- drop old tables
DROP TABLE system_advisories_v1;
DROP TABLE system_platform_v1;

CALL raise_notice('_v1 tables removed');