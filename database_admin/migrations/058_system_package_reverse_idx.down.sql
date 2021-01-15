DROP INDEX IF EXISTS system_package_package_system_idx;

ALTER TABLE system_package
    DROP COLUMN IF EXISTS name_id;
