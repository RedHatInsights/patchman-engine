
if true ; then
    cat /${CONTAINER_SCRIPTS_PATH}/start/schema.sql | psql -d ${POSTGRESQL_DATABASE}
else
    echo "Schema initialization skipped."
fi
