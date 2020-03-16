#!/usr/bin/bash

export PGHOST=$DB_HOST
export PGUSER=$DB_ADMIN_USER
export PGPASSWORD=$DB_ADMIN_PASSWD
export PGDATABASE=$DB_NAME
export PGPORT=$DB_PORT
export PGSSLMODE=$DB_SSLMODE

WAIT_FOR_EMPTY_DB=1 /database_admin/wait-for-services.sh

DB_INITIALIZED=$(psql -c "\d" | grep schema_migrations | wc -l)

# we cain either create the database from scratch, or upgrade running database
if [[ $DB_INITIALIZED == "0" ]]; then
  echo "Creating database from scratch"
  # Need to make sure admin role has createrole attribute
  psql -c "ALTER USER ${DB_USER} WITH CREATEROLE"
  psql -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER}"
  psql -c "ALTER USER ${DB_USER} WITH SUPERUSER"

  # Create users if they don't exist
  psql -f /database_admin/schema/create_users.sql

  echo "Initializing the database through migrations"
  /database_admin/migrate.sh up
else
  echo "Already initialized - Migrating the database"
  /database_admin/migrate.sh up
fi

echo "Setting user passwords"
# Set specific password for each user. If the users are already created, change the password.
# This is performed on each startup in order to ensure users have latest pasword
psql -c "ALTER USER listener WITH PASSWORD '${LISTENER_PASSWORD}'"
psql -c "ALTER USER evaluator WITH PASSWORD '${EVALUATOR_PASSWORD}'"
psql -c "ALTER USER manager WITH PASSWORD '${MANAGER_PASSWORD}'"
psql -c "ALTER USER vmaas_sync WITH PASSWORD '${VMAAS_SYNC_PASSWORD}'"
