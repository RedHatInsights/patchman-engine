
if true ; then
    cat /${CONTAINER_SCRIPTS_PATH}/start/schema.sql | psql -d ${POSTGRESQL_DATABASE}
    #psql -c "ALTER USER spm_admin WITH PASSWORD '${POSTGRESQL_WRITER_PASSWORD}'" -d ${POSTGRESQL_DATABASE}
else
    echo "Schema initialization skipped."
fi
