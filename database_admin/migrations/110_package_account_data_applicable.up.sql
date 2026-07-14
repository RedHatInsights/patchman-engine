TRUNCATE package_account_data;
UPDATE rh_account SET valid_package_cache = FALSE;
ALTER TABLE package_account_data RENAME COLUMN systems_updatable TO systems_installable;
ALTER TABLE package_account_data RENAME CONSTRAINT package_account_data_systems_updatable_not_null
                                                TO package_account_data_systems_installable_not_null;
ALTER TABLE package_account_data ADD COLUMN systems_applicable INT NOT NULL DEFAULT 0;
