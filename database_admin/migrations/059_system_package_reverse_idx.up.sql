ALTER TABLE system_package
    ADD COLUMN IF NOT EXISTS name_id INTEGER REFERENCES package_name (id);
