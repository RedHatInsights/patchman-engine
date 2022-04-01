ALTER TABLE baseline ADD CHECK (not empty(name));

ALTER TABLE baseline ADD COLUMN description TEXT;

ALTER TABLE baseline ADD CONSTRAINT baseline_rh_account_id_name_key UNIQUE(rh_account_id, name);
