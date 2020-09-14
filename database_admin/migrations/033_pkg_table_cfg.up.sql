ALTER TABLE package
    SET (fillfactor = 70);
ALTER TABLE system_package
    SET (fillfactor = 70);

-- Vacuum the tables after 5% of tuples are updated
ALTER TABLE package
    SET (autovacuum_vacuum_scale_factor = 0.05);
ALTER TABLE system_package
    SET (autovacuum_vacuum_scale_factor = 0.05);
