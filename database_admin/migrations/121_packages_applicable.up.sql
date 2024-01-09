ALTER TABLE system_platform
    RENAME COLUMN packages_updatable TO packages_installable;
ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS packages_applicable
        INT NOT NULL DEFAULT 0;
