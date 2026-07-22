ALTER TABLE system_inventory
    ADD COLUMN IF NOT EXISTS crowdstrike_workload  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS ibm_db2_workload      BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS intersystems_workload BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS oracle_db_workload    BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS rhel_ai_workload      BOOLEAN NOT NULL DEFAULT false;
