
-- migrate syste_package partitions one by one
DO
$$
    DECLARE
        part text;
    BEGIN
        FOR part IN (SELECT tablename from pg_tables t where t.tablename ~ '^system_package_[0-9]+$')
            LOOP
                RAISE NOTICE 'Copying the % partition', part;
                EXECUTE 'INSERT INTO system_package2 (rh_account_id, system_id, name_id, package_id, installable_id, applicable_id)
                        ( SELECT rh_account_id, system_id, name_id, package_id, NULL, NULL FROM ' ||
                        part || ') ON CONFLICT DO NOTHING';
            END LOOP;
    END
$$ LANGUAGE plpgsql;
