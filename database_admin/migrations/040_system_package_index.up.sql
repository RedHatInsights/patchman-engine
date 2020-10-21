ALTER TABLE system_package
    DROP COLUMN IF EXISTS updatable;

ALTER TABLE system_package
    ADD COLUMN IF NOT EXISTS latest_evra
        TEXT GENERATED ALWAYS AS ( ((update_data ->> -1)::jsonb ->> 'evra')::text) STORED;

ALTER TABLE system_package
    DROP CONSTRAINT IF EXISTS system_package_pkey;

DROP INDEX IF EXISTS system_package_pkey;

ALTER TABLE system_package
    ADD PRIMARY KEY (rh_account_id, system_id, package_id) INCLUDE (latest_evra);
