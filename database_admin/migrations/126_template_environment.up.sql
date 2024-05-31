ALTER TABLE template ADD COLUMN IF NOT EXISTS environment_id TEXT CHECK (NOT empty(environment_id));

UPDATE template set environment_id = REPLACE(uuid::text, '-', '');

ALTER TABLE template ALTER COLUMN environment_id SET NOT NULL;
