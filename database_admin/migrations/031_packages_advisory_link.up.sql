ALTER TABLE package
    ADD COLUMN IF NOT EXISTS
        advisory_id INT REFERENCES advisory_metadata (id);
