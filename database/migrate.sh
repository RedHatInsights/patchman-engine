#!/bin/bash

set -o pipefail

export PGHOST=$DB_HOST
export PGUSER=$DB_USER
export PGDATABASE=$DB_NAME
export PGPORT=$DB_PORT
export PGPASSWORD=$DB_PASSWD

psql -a -f /database/schema/create_users.sql

/migrate -source file://database/migrations -database postgres://$DB_HOST/$DB_NAME$MIGRATE_DB_URL_PARAMS up

sleep infinity
