CREATE TABLE IF NOT EXISTS template_advisory
(
    rh_account_id INT    NOT NULL,
    template_id   BIGINT NOT NULL,
    advisory_id   BIGINT NOT NULL,
    PRIMARY KEY (rh_account_id, template_id, advisory_id),
    CONSTRAINT template_advisory_template_id
        FOREIGN KEY (rh_account_id, template_id)
            REFERENCES template (rh_account_id, id) ON DELETE CASCADE,
    CONSTRAINT template_advisory_advisory_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id) ON DELETE CASCADE
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('template_advisory', 16,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

CREATE INDEX ON template_advisory (rh_account_id, advisory_id);

SELECT grant_table_partitions('SELECT', 'template_advisory', 'manager');
SELECT grant_table_partitions('SELECT', 'template_advisory', 'evaluator');
SELECT grant_table_partitions('SELECT, INSERT, UPDATE, DELETE', 'template_advisory', 'listener');
SELECT grant_table_partitions('SELECT', 'template_advisory', 'vmaas_sync');
