-- Smoke test for the SQL job runner. Does not modify data.
DO $$
BEGIN
    RAISE NOTICE 'SQL job smoke test completed successfully';
END $$;
