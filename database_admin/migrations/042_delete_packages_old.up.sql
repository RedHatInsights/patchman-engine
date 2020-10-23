DROP TABLE system_package;


ALTER TABLE system_package_v1
    RENAME TO system_package;


DO
$$
    DECLARE
        row text;
    BEGIN
        FOR row IN (SELECT tablename from pg_tables t where t.tablename like 'system_package_v1_%')
            LOOP
                EXECUTE 'ALTER TABLE ' || row || ' rename to ' || replace(row, '_v1', '');
            END LOOP;
    END
$$ LANGUAGE plpgsql;

DO
$$
    DECLARE
        row text;
    BEGIN
        FOR row IN (SELECT indexname from pg_indexes t where t.indexname like 'system_package_v1_%')
            LOOP
                EXECUTE 'ALTER INDEX ' || row || ' rename to ' || replace(row, '_v1', '');
            END LOOP;
    END
$$ LANGUAGE plpgsql;

ALTER TABLE system_package
    RENAME CONSTRAINT system_package_v1_package_id_fkey TO system_package_package_id_fkey;

ALTER TABLE system_package
    RENAME CONSTRAINT system_package_v1_rh_account_id_fkey TO system_package_rh_account_id_fkey;