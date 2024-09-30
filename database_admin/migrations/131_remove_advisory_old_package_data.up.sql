delete from advisory_metadata am where package_data is not json array and not exists (select 1 from advisory_account_data aad where aad.advisory_id = am.id);
update advisory_metadata set package_data = null where package_data is not json array;
