DO
$$
  DECLARE
      deleted integer;
  BEGIN
     SELECT count(*) INTO deleted FROM (
            SELECT delete_system(inventory_id) FROM system_platform sp
            LEFT JOIN inventory.hosts ih ON sp.inventory_id = ih.id
            WHERE ih.id IS NULL ) x;
     RAISE NOTICE '%', deleted;
  EXCEPTION
     -- inventory.hosts is not in our schema, skip if not presented
     WHEN others THEN
        RAISE NOTICE 'Unable to clean systems data!';
  END
$$ LANGUAGE plpgsql;
