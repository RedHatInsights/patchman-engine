#!/usr/bin/bash

export PGHOST=$DB_HOST
export PGUSER=$DB_USER
export PGPASSWORD=$DB_PASSWD
export PGDATABASE=$DB_NAME
export PGPORT=$DB_PORT
export PGSSLMODE=$DB_SSLMODE

WAIT_FOR_EMPTY_DB=1 ./scripts/wait-for-services.sh

DB_INITIALIZED=$(psql -c "\d" | grep schema_migrations | wc -l)

# we cain either create the database from scratch, or upgrade running database
if [[ $DB_INITIALIZED == "0" ]]; then
  # Create users if they don't exist
  echo "Creating application components users"
  psql -f ./database_admin/schema/create_users.sql
fi

echo "Migrating the database"
./database_admin/migrate.sh

echo "Setting user passwords"
# Set specific password for each user. If the users are already created, change the password.
# This is performed on each startup in order to ensure users have latest pasword
psql -c "ALTER USER listener WITH PASSWORD '${LISTENER_PASSWORD}'"
psql -c "ALTER USER evaluator WITH PASSWORD '${EVALUATOR_PASSWORD}'"
psql -c "ALTER USER manager WITH PASSWORD '${MANAGER_PASSWORD}'"
psql -c "ALTER USER vmaas_sync WITH PASSWORD '${VMAAS_SYNC_PASSWORD}'"
