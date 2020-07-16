ALTER TABLE system_platform
    DROP COLUMN IF EXISTS packages_installed;

ALTER TABLE system_platform
    DROP COLUMN IF EXISTS packages_updatable;
