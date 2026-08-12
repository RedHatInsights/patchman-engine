ALTER TABLE system_inventory
    ADD COLUMN IF NOT EXISTS satellite_workload BOOLEAN NOT NULL DEFAULT false;
