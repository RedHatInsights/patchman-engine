CREATE OR REPLACE PROCEDURE copy_system_packages()
    LANGUAGE plpgsql
AS
$$
DECLARE
    cnt           bigint := 0;
    prev_cnt      bigint := 0;
    rows_inserted bigint := 0;
    account       int    := 0;
BEGIN
    FOR account IN (SELECT id from rh_account ORDER BY hash_partition_id(id, 128), id)
        LOOP
            INSERT INTO system_package2
            SELECT system_package.rh_account_id,
                   system_id,
                   name_id,
                   package_id,
                   (SELECT id
                    FROM package
                    WHERE package.name_id = system_package.name_id
                      AND evra =
                          JSONB_PATH_QUERY_ARRAY(update_data, '$[*] ? (@.status== "Installable").evra') ->> 0),
                   (SELECT id
                    FROM package
                    WHERE package.name_id = system_package.name_id
                      AND evra = JSONB_PATH_QUERY_ARRAY(update_data, '$[*] ? (@.status== "Applicable").evra') ->> 0)
            FROM system_package
            JOIN system_platform ON system_platform.id = system_package.system_id AND system_platform.rh_account_id = system_package.rh_account_id
            WHERE system_package.rh_account_id = account;
            COMMIT;

            GET DIAGNOSTICS rows_inserted = ROW_COUNT;

            cnt := cnt + rows_inserted;
            IF (cnt/1000000)::int > (prev_cnt/1000000)::int THEN
                RAISE NOTICE 'inserted % rows, account: %, partition: %', cnt, account, hash_partition_id(account, 128);
                prev_cnt := cnt;
            END IF;
        END LOOP;
END
$$;
