ALTER TABLE baseline ADD CHECK (not empty(name));

ALTER TABLE baseline ADD COLUMN description TEXT;

UPDATE baseline SET name = name || ' ' || id
 WHERE (rh_account_id, name) in (SELECT rh_account_id, name FROM baseline GROUP BY rh_account_id, name HAVING count(*) > 1);
ALTER TABLE baseline ADD CONSTRAINT baseline_rh_account_id_name_key UNIQUE(rh_account_id, name);
