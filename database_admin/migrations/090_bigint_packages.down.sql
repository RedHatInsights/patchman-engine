ALTER TABLE package ALTER COLUMN id TYPE INT,
                    ALTER COLUMN name_id TYPE INT,
                    ALTER COLUMN advisory_id TYPE INT;

ALTER TABLE system_package ALTER COLUMN package_id TYPE INT,
                    ALTER COLUMN system_id TYPE INT,
                    ALTER COLUMN name_id TYPE INT;
