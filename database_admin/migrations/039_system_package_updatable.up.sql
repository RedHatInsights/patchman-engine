ALTER TABLE system_package
    ADD COLUMN IF NOT EXISTS updatable BOOL GENERATED ALWAYS AS ( update_data IS NOT NULL ) STORED;

ALTER TABLE system_package
    DROP CONSTRAINT IF EXISTS system_package_pkey;

DROP INDEX IF EXISTS system_package_pkey;

ALTER TABLE system_package
    ADD PRIMARY KEY (rh_account_id, system_id, package_id) INCLUDE (updatable);
