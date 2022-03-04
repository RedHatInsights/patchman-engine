-- tested performance on RDS with generated data
-- system_package: 584M rows, 111GB table, 216GB total
-- CREATE INDEX Time: 624372.198 ms (10:24.372)

CREATE INDEX IF NOT EXISTS system_package_package_id_idx on system_package (package_id);
