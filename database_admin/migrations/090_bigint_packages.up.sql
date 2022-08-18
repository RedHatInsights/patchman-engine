ALTER TABLE package ALTER COLUMN id TYPE BIGINT,
                    ALTER COLUMN name_id TYPE BIGINT,
                    ALTER COLUMN advisory_id TYPE BIGINT;

-- estimate: Time: 3801816.370 ms (01:03:21.816)
ALTER TABLE system_package ALTER COLUMN package_id TYPE BIGINT,
                    ALTER COLUMN system_id TYPE BIGINT,
                    ALTER COLUMN name_id TYPE BIGINT;
