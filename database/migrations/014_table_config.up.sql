ALTER TABLE system_platform
    SET (fillfactor = 70);
ALTER TABLE system_advisories
    SET (fillfactor = 70);
ALTER TABLE advisory_account_data
    SET (fillfactor = 70);

-- Vacuum the tables after 5% of tuples are updated
ALTER TABLE system_platform
    SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE system_advisories
    SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE advisory_account_data
    SET (autovacuum_vacuum_scale_factor = 0.05);

DROP INDEX IF EXISTS system_advisories_status_id_idx;