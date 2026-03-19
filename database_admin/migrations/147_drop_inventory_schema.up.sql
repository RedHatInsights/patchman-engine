DO $$
BEGIN
  REVOKE cyndi_reader FROM listener; 
  EXCEPTION WHEN undefined_object THEN NULL; 
END
$$;

DO $$
BEGIN
  REVOKE cyndi_reader FROM evaluator; 
  EXCEPTION WHEN undefined_object THEN NULL; 
END
$$;

DO $$
BEGIN
  REVOKE cyndi_reader FROM manager; 
  EXCEPTION WHEN undefined_object THEN NULL; 
END
$$;

DO $$
BEGIN
  REVOKE cyndi_reader FROM vmaas_sync; 
  EXCEPTION WHEN undefined_object THEN NULL; 
END
$$;

DO $$
BEGIN
  REVOKE cyndi_admin FROM cyndi; 
  EXCEPTION WHEN undefined_object THEN NULL; 
END
$$;

REVOKE ALL PRIVILEGES ON SCHEMA inventory FROM cyndi_admin; 

REVOKE USAGE ON SCHEMA inventory FROM cyndi_reader; 

DROP VIEW IF EXISTS inventory.hosts;

DROP TABLE IF EXISTS inventory.hosts_v1_0;

DROP ROLE IF EXISTS cyndi_admin;

DROP ROLE IF EXISTS cyndi_reader;

DROP SCHEMA IF EXISTS inventory;
