ALTER TABLE deleted_system ADD CONSTRAINT deleted_system_pkey PRIMARY KEY (inventory_id);
ALTER TABLE deleted_system DROP CONSTRAINT deleted_system_inventory_id_key;

ALTER TABLE system_repo ADD CONSTRAINT system_repo_pkey PRIMARY KEY (rh_account_id, system_id, repo_id);
ALTER TABLE system_repo DROP CONSTRAINT system_repo_rh_account_id_system_id_repo_id_key;

ALTER TABLE timestamp_kv ADD CONSTRAINT timestamp_kv_pkey PRIMARY KEY (name);
ALTER TABLE timestamp_kv DROP CONSTRAINT timestamp_kv_name_key;