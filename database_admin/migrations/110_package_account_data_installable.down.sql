TRUNCATE package_account_data;
UPDATE rh_account SET valid_package_cache = FALSE;
ALTER TABLE package_account_data RENAME COLUMN systems_installable TO systems_updatable;
ALTER TABLE package_account_data DROP COLUMN systems_applicable;
