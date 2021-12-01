ALTER TABLE system_platform DROP COLUMN IF EXISTS baseline_id;
ALTER TABLE system_platform DROP COLUMN IF EXISTS baseline_uptodate;
DROP TABLE IF EXISTS baseline;
