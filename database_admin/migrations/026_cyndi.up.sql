CREATE SCHEMA IF NOT EXISTS inventory;

DO
$$
    BEGIN
        -- The admin ROLE that allows the inventory schema to be managed
        CREATE ROLE cyndi_admin;
        -- The reader ROLE that provides SELECT access to the inventory.hosts view
        CREATE ROLE cyndi_reader;
    EXCEPTION
        WHEN DUPLICATE_OBJECT THEN NULL;
    END
$$;

GRANT ALL PRIVILEGES ON SCHEMA inventory TO cyndi_admin;
-- The application user is granted the reader role only to eliminate any interference with Cyndi
GRANT USAGE ON SCHEMA inventory TO cyndi_reader;

GRANT cyndi_reader to listener;
GRANT cyndi_reader to evaluator;
GRANT cyndi_reader to manager;

GRANT cyndi_admin to cyndi;