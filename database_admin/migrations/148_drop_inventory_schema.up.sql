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

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'inventory') THEN
    EXECUTE 'REVOKE ALL PRIVILEGES ON SCHEMA inventory FROM cyndi_admin';
  END IF;
END
$$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'inventory') THEN
    EXECUTE 'REVOKE USAGE ON SCHEMA inventory FROM cyndi_reader';
  END IF;
END
$$;

DROP ROLE IF EXISTS cyndi_admin;

DROP ROLE IF EXISTS cyndi_reader;

DROP SCHEMA IF EXISTS inventory CASCADE;

DROP USER IF EXISTS cyndi;
