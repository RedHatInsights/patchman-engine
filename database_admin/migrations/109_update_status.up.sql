CREATE OR REPLACE FUNCTION update_status(update_data jsonb)
    RETURNS TEXT as
$$
DECLARE
    len int;
BEGIN
    len = jsonb_array_length(update_data);
    IF len IS NULL or len = 0 THEN
        RETURN 'None';
    END IF;
    len = jsonb_array_length(jsonb_path_query_array(update_data, '$ ? (@.status == "Installable")'));
    IF len > 0 THEN
        RETURN 'Installable';
    END IF;
    RETURN 'Applicable';
END;
$$ LANGUAGE 'plpgsql';

