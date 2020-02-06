#!/usr/bin/bash

# we cain either create the database from scratch, or upgrade running database
if $PG_INITIALIZED; then
  echo "Creating database from scratch"
  # Need to make sure admin role has createrole attribute
  psql -c "ALTER USER ${POSTGRESQL_USER} WITH CREATEROLE" -d ${POSTGRESQL_DATABASE}
  psql -c "GRANT ALL PRIVILEGES ON DATABASE ${POSTGRESQL_DATABASE} TO ${POSTGRESQL_USER}"  -d ${POSTGRESQL_DATABASE}
  psql -c "ALTER USER ${POSTGRESQL_USER} WITH SUPERUSER"  -d ${POSTGRESQL_DATABASE}

  # Create users if they don't exist
  psql -d ${POSTGRESQL_DATABASE} -f ${CONTAINER_SCRIPTS_PATH}/start/create_users.sql


  #echo "Initializing the database through migrations"
  ${CONTAINER_SCRIPTS_PATH}/migrate.sh up
  # Create schema from scratch
  #psql -d ${POSTGRESQL_DATABASE} -f ${CONTAINER_SCRIPTS_PATH}/start/create_schema.sql
else
  echo "Already initialized - Migrating the database"
  ${CONTAINER_SCRIPTS_PATH}/migrate.sh up
fi

echo "Setting user passwords"
# Set specific password for each user. If the users are already created, change the password.
# This is performed on each startup in order to ensure users have latest pasword
psql -c "ALTER USER listener WITH PASSWORD '${LISTENER_PASSWORD}'" -d ${POSTGRESQL_DATABASE}
psql -c "ALTER USER evaluator WITH PASSWORD '${EVALUATOR_PASSWORD}'" -d ${POSTGRESQL_DATABASE}
psql -c "ALTER USER manager WITH PASSWORD '${MANAGER_PASSWORD}'" -d ${POSTGRESQL_DATABASE}
psql -c "ALTER USER vmaas_sync WITH PASSWORD '${VMAAS_SYNC_PASSWORD}'" -d ${POSTGRESQL_DATABASE}