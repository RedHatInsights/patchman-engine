CREATE TABLE IF NOT EXISTS account_advisory
(
    advisory_id              BIGINT NOT NULL,
    rh_account_id            INT    NOT NULL,
    workspace_id             TEXT   NOT NULL,
    systems_applicable       INT    NOT NULL DEFAULT 0,
    systems_installable      INT    NOT NULL DEFAULT 0,
    notified                 TIMESTAMP WITH TIME ZONE NULL,
    CONSTRAINT account_advisory_advisory_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    UNIQUE (advisory_id, rh_account_id, workspace_id),
    PRIMARY KEY (rh_account_id, workspace_id, advisory_id)
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('account_advisory', 32,
    $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')
      TABLESPACE pg_default$$);

SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'manager');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'evaluator');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'listener');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'account_advisory', 'vmaas_sync');

CREATE INDEX ON account_advisory (rh_account_id, workspace_id);
