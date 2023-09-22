DO $$
    DECLARE
      rownum INT;
      total INT;
      account_id INT;
      sys_id BIGINT;
      update_list TEXT[] := NULL;
      json TEXT := '';
      pkgname_id BIGINT;
    BEGIN
      -- order accounts so that we read from system_package by partitions
      FOR rownum, total, account_id IN
          SELECT row_number() over (), count(*) over (), id FROM
            ( SELECT id FROM rh_account order by hash_partition_id(id, 128), id) as o
      LOOP
          RAISE NOTICE 'Migrating account % (%/%)', account_id, rownum, total;
          FOR sys_id, update_list IN
              SELECT system_id,
                     array_agg(CONCAT('"', package_id, '":[', '{"evra":"' || i.evra || '","status":"Installable"}',
                                      CASE WHEN i.evra IS NOT NULL AND a.evra IS NOT NULL THEN ',' ELSE '' END,
                                      '{"evra":"' || a.evra || '","status":"Applicable"}', ']')) as update_list
                FROM system_package2 sp
                LEFT JOIN package i ON sp.installable_id = i.id
                LEFT JOIN package a ON sp.applicable_id = a.id
               WHERE sp.rh_account_id = account_id
               GROUP BY system_id
          LOOP
              json := '{' || array_to_string(update_list, ',') || '}';
              -- RAISE NOTICE 'system_package_data (%, %, %)', account_id, sys_id, json;
              INSERT INTO system_package_data VALUES (account_id, sys_id, json::jsonb); 
          END LOOP;

          FOR pkgname_id, update_list IN
              SELECT sp.name_id,
                     array_agg(CONCAT('"', sp.system_id, '":[{"evra":"', p.evra,'", "status": "Installed"}',
                               ',{"evra":"' || i.evra || '","status":"Installable"}',
                               ',{"evra":"' || a.evra || '","status":"Applicable"}', ']')) as update_list
                FROM system_package2 sp
                JOIN package p ON p.id = sp.package_id
                LEFT JOIN package i ON sp.installable_id = i.id
                LEFT JOIN package a ON sp.applicable_id = a.id
               WHERE sp.rh_account_id = account_id
               GROUP BY sp.name_id
          LOOP
              json := '{' || array_to_string(update_list, ',') || '}';
              -- RAISE NOTICE 'package_system_data (%, %, %)', account_id, pkgname_id, json;
              INSERT INTO package_system_data VALUES (account_id, pkgname_id, json::jsonb); 
          END LOOP;
      END LOOP;
    END;
$$
;    

