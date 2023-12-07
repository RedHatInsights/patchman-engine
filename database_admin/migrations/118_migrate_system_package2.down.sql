/*
Loops through the PK(account_id, system_id, package_id) in chunks of 1000 and inserts it into the new table.
Assumptions:
1. the system_packages2 table is empty.
2. No new system_packages comes in when this is running.
*/

CREATE OR REPLACE PROCEDURE copy_system_packages()
    LANGUAGE plpgsql
AS
$$
DECLARE
    rows_inserted INTEGER := 0;
    account_idx   integer := 0;
    system_idx    bigint  := 0;
    package_idx   bigint  := 0;
BEGIN
    LOOP
        INSERT INTO system_package2
        SELECT rh_account_id,
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
        WHERE (rh_account_id = account_idx AND system_id = system_idx AND package_id > package_idx)
        OR (rh_account_id = account_idx AND system_id > system_idx)
        ORDER BY rh_account_id, system_id, package_id
        LIMIT 1000;

        GET DIAGNOSTICS rows_inserted = ROW_COUNT;

        COMMIT;

        /* I Originally did not include this if statement but I couldn't get the select to use the PK index when I had all 3 where clauses

        1. rh_account_id = account_idx AND system_id = system_idx AND package_id > package_idx
        2. rh_account_id = account_idx AND system_id > system_idx
        3. rh_account_id > account_idx

        So I just copied and pasted the same select / insert and used the final clause when the first two return 0 rows
        */
        IF rows_inserted = 0 THEN
            INSERT INTO system_package2
            SELECT rh_account_id,
                system_id,
                name_id,
                package_id,
                (SELECT id
                    FROM package
                    WHERE package.name_id = system_package.name_id
                    AND evra = JSONB_PATH_QUERY_ARRAY(update_data, '$[*] ? (@.status== "Installable").evra') ->> 0),
                (SELECT id
                    FROM package
                    WHERE package.name_id = system_package.name_id
                    AND evra = JSONB_PATH_QUERY_ARRAY(update_data, '$[*] ? (@.status== "Applicable").evra') ->> 0)
            FROM system_package
            WHERE rh_account_id > account_idx
            ORDER BY rh_account_id, system_id, package_id
            LIMIT 1000;

            GET DIAGNOSTICS rows_inserted = ROW_COUNT;

            COMMIT;
        END IF;

        EXIT WHEN rows_inserted = 0;

        /* Store the highest values of our account/system/package ids are up to.
        Should be O(1) because it can just look at the tail of the PK index*/
        SELECT rh_account_id, system_id, package_id
        INTO account_idx, system_idx, package_idx
        FROM system_package2
        ORDER BY rh_account_id DESC, system_id DESC, package_id DESC
        LIMIT 1;

    END LOOP;
END
$$;
