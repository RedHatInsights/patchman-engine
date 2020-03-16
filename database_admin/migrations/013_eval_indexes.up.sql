ALTER TABLE system_advisories
    DROP CONSTRAINT IF EXISTS system_advisories_system_id_advisory_id_key;

ALTER TABLE system_advisories
    DROP CONSTRAINT system_advisories_pkey;

ALTER TABLE system_advisories
    DROP COLUMN id;

ALTER TABLE system_advisories
    ADD PRIMARY KEY (system_id, advisory_id);

ALTER TABLE advisory_account_data
    DROP CONSTRAINT IF EXISTS advisory_account_data_rh_account_id_advisory_id_key;

ALTER TABLE advisory_account_data
    ADD CONSTRAINT advisory_account_data_pkey PRIMARY KEY (rh_account_id, advisory_id);
