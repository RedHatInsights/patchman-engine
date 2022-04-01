ALTER TABLE baseline DROP COLUMN IF EXISTS description;

ALTER TABLE baseline DROP CONSTRAINT IF EXISTS baseline_rh_account_id_name_key;
