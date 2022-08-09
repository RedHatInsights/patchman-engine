ALTER TABLE rh_account ALTER COLUMN id TYPE INT;

ALTER TABLE baseline ALTER COLUMN id TYPE INT;
ALTER TABLE baseline ALTER COLUMN rh_account_id TYPE INT;

ALTER TABLE system_platform ALTER COLUMN id TYPE INT;
ALTER TABLE system_platform ALTER COLUMN rh_account_id TYPE INT;
ALTER TABLE system_platform ALTER COLUMN baseline_id TYPE INT;

ALTER TABLE package_name ALTER COLUMN id TYPE INT;

ALTER TABLE package ALTER COLUMN id TYPE INT;
ALTER TABLE package ALTER COLUMN name_id TYPE INT;

ALTER TABLE system_package ALTER COLUMN rh_account_id TYPE INT;
ALTER TABLE system_package ALTER COLUMN system_id TYPE INT;
ALTER TABLE system_package ALTER COLUMN package_id TYPE INT;
ALTER TABLE system_package ALTER COLUMN name_id TYPE INT;

ALTER TABLE system_advisories ALTER COLUMN rh_account_id TYPE INT;
ALTER TABLE system_advisories ALTER COLUMN system_id TYPE INT;
ALTER TABLE system_advisories ALTER COLUMN advisory_id TYPE INT;

ALTER TABLE system_repo ALTER COLUMN rh_account_id TYPE INT;
ALTER TABLE system_repo ALTER COLUMN system_id TYPE INT;
ALTER TABLE system_repo ALTER COLUMN repo_id TYPE INT;