DO $$
BEGIN
    RAISE NOTICE 'No down migration for edge systems marking!';
END;
$$ LANGUAGE plpgsql;
