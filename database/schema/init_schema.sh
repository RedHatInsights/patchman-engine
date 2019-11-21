#!/usr/bin/bash

if $PG_INITIALIZED ; then
    # Create users if they don't exist
    psql -d ${POSTGRESQL_DATABASE} -f ${CONTAINER_SCRIPTS_PATH}/start/create_users.sql

    # Need to make sure admin role has createrole attribute
    psql -c "ALTER USER ${POSTGRESQL_USER} WITH CREATEROLE" -d ${POSTGRESQL_DATABASE}

    # Init database schema
    psql -U ${POSTGRESQL_USER} -d ${POSTGRESQL_DATABASE} -f ${CONTAINER_SCRIPTS_PATH}/start/create_schema.sql

else
  echo "Schema initialization skipped."
fi

echo "Setting user passwords"
# Set specific password for each user. If the users are already created, change the password.
psql -c "ALTER USER listener WITH PASSWORD '${LISTENER_PASSWORD}'" -d ${POSTGRESQL_DATABASE}
psql -c "ALTER USER manager WITH PASSWORD '${MANAGER_PASSWORD}'" -d ${POSTGRESQL_DATABASE}
