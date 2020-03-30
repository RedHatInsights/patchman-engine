ALTER TABLE system_platform
    ADD COLUMN IF NOT EXISTS
        display_name TEXT NOT NULL CHECK (NOT (empty(display_name)))
            DEFAULT '__REPLACE__';

-- For existing systems, use inventory_id
UPDATE system_platform
SET display_name = inventory_id
WHERE system_platform.display_name = '__REPLACE__';

-- Require user to supply display_name
ALTER TABLE system_platform
    ALTER COLUMN display_name DROP DEFAULT;
