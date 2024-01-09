ALTER TABLE system_platform
    DROP COLUMN IF EXISTS packages_applicable;
ALTER TABLE system_platform
    RENAME COLUMN packages_installable TO packages_updatable;
