DO
$$
    BEGIN
        RAISE EXCEPTION 'Down migration is not supported';
    END;
$$ LANGUAGE plpgsql;