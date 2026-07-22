ALTER TABLE system_inventory
    DROP COLUMN IF EXISTS crowdstrike_workload,
    DROP COLUMN IF EXISTS ibm_db2_workload,
    DROP COLUMN IF EXISTS intersystems_workload,
    DROP COLUMN IF EXISTS oracle_db_workload,
    DROP COLUMN IF EXISTS rhel_ai_workload;
