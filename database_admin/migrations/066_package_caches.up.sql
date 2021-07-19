CREATE TABLE IF NOT EXISTS package_account_data
(
    name_id           INT NOT NULL,
    rh_account_id     INT NOT NULL,
    systems_installed INT NOT NULL DEFAULT 0,
    systems_updatable INT NOT NULL DEFAULT 0,

    CONSTRAINT package_name_id
        FOREIGN KEY (name_id) REFERENCES package_name (id),
    CONSTRAINT rh_account_id
        FOREIGN KEY (rh_account_id)
            REFERENCES rh_account (id),
    PRIMARY KEY (rh_account_id, name_id)
);


GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO manager;
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO evaluator;
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO listener;
GRANT SELECT, INSERT, UPDATE, DELETE ON package_account_data TO vmaas_sync;
