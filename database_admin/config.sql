-- In test/CI/QA environment
-- Log statements which take more than 2s
DO
$$
    DECLARE
      dbname text;
    BEGIN
      SELECT current_database() into dbname;
      IF dbname = 'patchman' THEN
        EXECUTE 'ALTER DATABASE ' || dbname || ' SET log_min_duration_statement = 2000;';
      END IF;
    END
$$ LANGUAGE plpgsql;
