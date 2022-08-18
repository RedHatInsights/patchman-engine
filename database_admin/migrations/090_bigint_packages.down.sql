ALTER TABLE package ALTER COLUMN id TYPE INT,
                    ALTER COLUMN name_id TYPE INT,
                    ALTER COLUMN advisory_id TYPE INT;

-- estimate: Time: 3801816.370 ms (01:03:21.816)
ALTER TABLE system_package ALTER COLUMN package_id TYPE INT,
                    ALTER COLUMN system_id TYPE INT,
                    ALTER COLUMN name_id TYPE INT;
