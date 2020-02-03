DROP FUNCTION IF EXISTS delete_culled_systems;

ALTER TABLE system_platform
    DROP COLUMN IF EXISTS stale;
ALTER TABLE system_platform
    DROP COLUMN IF EXISTS culled_timestamp;
ALTER TABLE system_platform
    DROP COLUMN IF EXISTS stale_warning_timestamp;
ALTER TABLE system_platform
    DROP COLUMN IF EXISTS stale_timestamp;