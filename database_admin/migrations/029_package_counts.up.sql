ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS packages_installed
        INT NOT NULL DEFAULT 0;

ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS packages_updatable
        INT NOT NULL DEFAULT 0;

