DO
$$
    DECLARE
        usr text;
    BEGIN
        FOR usr IN
            SELECT name
            FROM (VALUES ('evaluator'), ('listener'), ('manager'), ('vmaas_sync'), ('cyndi')) users (name)
            WHERE name NOT IN (SELECT rolname FROM pg_catalog.pg_roles)
            LOOP
                execute 'CREATE USER ' || usr || ';';
            END LOOP;
    END
$$