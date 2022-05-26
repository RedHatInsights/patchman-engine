ALTER TABLE rh_account ALTER COLUMN name DROP NOT NULL;
ALTER TABLE rh_account ADD COLUMN org_id TEXT UNIQUE;
ALTER TABLE rh_account ADD CHECK (name IS NOT NULL OR org_id IS NOT NULL);
ALTER TABLE rh_account ADD CONSTRAINT rh_account_org_id_check CHECK (NOT empty(org_id))
