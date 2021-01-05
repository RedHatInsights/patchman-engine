CALL raise_notice('system_advisories_v2 migration:');

-- create partitioned tables
-- skip constraints to make initial import faster
-- system_advisories
CREATE TABLE IF NOT EXISTS system_advisories_v2
(
    rh_account_id  INT                      NOT NULL,
    system_id      INT                      NOT NULL,
    advisory_id    INT                      NOT NULL,
    first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
    when_patched   TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    status_id      INT                      DEFAULT 0
) PARTITION BY HASH (rh_account_id);

SELECT create_table_partitions('system_advisories_v2', 32,
                               $$WITH (fillfactor = '70', autovacuum_vacuum_scale_factor = '0.05')$$);

CALL raise_notice('partitoned table created');


-- data migration
TRUNCATE TABLE system_advisories_v2;
INSERT INTO system_advisories_v2 (
        rh_account_id,
        system_id,
        advisory_id,
        first_reported,
        when_patched,
        status_id
        )
    SELECT
        sp.rh_account_id,
        sa.system_id,
        sa.advisory_id,
        sa.first_reported,
        sa.when_patched,
        sa.status_id
    FROM system_advisories sa
    JOIN system_platform sp
      ON sp.id = sa.system_id;
DO
$$
    DECLARE
        old NUMERIC;
        new NUMERIC;
    BEGIN
        SELECT count(*) from system_advisories into old;
        SELECT count(*) from system_advisories_v2 into new;
        RAISE NOTICE 'data migrated';
        RAISE NOTICE '    row count: %', old;
        RAISE NOTICE '_v2 row count: %', new;
    END;
$$ LANGUAGE plpgsql;


-- enable constraints on new tables
ALTER TABLE system_advisories_v2
    DROP CONSTRAINT IF EXISTS  system_advisories_v2_pkey,
    ADD PRIMARY KEY (rh_account_id, system_id, advisory_id),
    DROP CONSTRAINT IF EXISTS system_platform_id,
    ADD CONSTRAINT system_platform_id
        FOREIGN KEY (rh_account_id, system_id)
            REFERENCES system_platform_v2 (rh_account_id, id),
    DROP CONSTRAINT IF EXISTS advisory_metadata_id,
    ADD CONSTRAINT advisory_metadata_id
        FOREIGN KEY (advisory_id)
            REFERENCES advisory_metadata (id),
    DROP CONSTRAINT IF EXISTS status_id,
    ADD CONSTRAINT status_id
        FOREIGN KEY (status_id)
            REFERENCES status (id);

CALL raise_notice('constraints created');

SELECT create_table_partition_triggers('system_advisories_set_first_reported',
                                       $$BEFORE INSERT$$,
                                       'system_advisories_v2',
                                       $$FOR EACH ROW EXECUTE PROCEDURE set_first_reported()$$);

CALL raise_notice('triggers created');

GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories_v2 TO evaluator;
-- manager needs to be able to update things like 'status' on a sysid/advisory combination, also needs to delete
GRANT UPDATE, DELETE ON system_advisories_v2 TO manager;
-- manager needs to be able to update opt_out column
GRANT UPDATE (opt_out) ON system_platform_v2 TO manager;
-- listener deletes systems, TODO: temporary added evaluator permissions to listener
GRANT SELECT, INSERT, UPDATE, DELETE ON system_advisories_v2 TO listener;
-- vmaas_sync needs to delete culled systems, which cascades to system_advisories
GRANT SELECT, DELETE ON system_advisories_v2 TO vmaas_sync;

CALL raise_notice('permission granted');
