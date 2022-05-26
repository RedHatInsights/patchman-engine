ALTER TABLE rh_account ALTER COLUMN name ADD NOT NULL;
ALTER TABLE rh_account DROP CONSTRAINT rh_account_org_id_check;
ALTER TABLE rh_account DROP CONSTRAINT rh_account_check;
ALTER TABLE rh_account DROP COLUMN org_id;
