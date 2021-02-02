ALTER TABLE system_package
    ADD COLUMN IF NOT EXISTS name_id INTEGER REFERENCES package_name (id);

UPDATE system_package
SET name_id = (select pn.id
               from package_name pn
                        JOIN package p on pn.id = p.name_id
               where p.id = system_package.package_id);

DELETE FROM system_package WHERE name_id IS NULL;

ALTER TABLE system_package
    ALTER COLUMN name_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS system_package_name_pkg_system_idx
    ON system_package (rh_account_id, name_id, package_id, system_id)
    INCLUDE (latest_evra);
