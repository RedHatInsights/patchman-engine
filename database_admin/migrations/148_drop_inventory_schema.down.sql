CREATE SCHEMA IF NOT EXISTS inventory;

CREATE USER cyndi;

-- The admin ROLE that allows the inventory schema to be managed
DO $$
BEGIN
  CREATE ROLE cyndi_admin;
  EXCEPTION WHEN DUPLICATE_OBJECT THEN
    RAISE NOTICE 'cyndi_admin already exists';
END
$$;
GRANT ALL PRIVILEGES ON SCHEMA inventory TO cyndi_admin;

-- The reader ROLE that provides SELECT access to the inventory.hosts view
DO $$
BEGIN
  CREATE ROLE cyndi_reader;
  EXCEPTION WHEN DUPLICATE_OBJECT THEN
    RAISE NOTICE 'cyndi_reader already exists';
END
$$;
GRANT USAGE ON SCHEMA inventory TO cyndi_reader;

-- The application user is granted the reader role only to eliminate any interference with Cyndi
GRANT cyndi_reader to listener;
GRANT cyndi_reader to evaluator;
GRANT cyndi_reader to manager;
GRANT cyndi_reader TO vmaas_sync;

GRANT cyndi_admin to cyndi;