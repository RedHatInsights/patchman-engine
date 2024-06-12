ALTER TABLE template ADD COLUMN IF NOT EXISTS arch TEXT CHECK (NOT empty(arch));
ALTER TABLE template ADD COLUMN IF NOT EXISTS version TEXT CHECK (NOT empty(version));
ALTER TABLE system_platform ADD COLUMN IF NOT EXISTS arch TEXT CHECK (NOT empty(arch));
