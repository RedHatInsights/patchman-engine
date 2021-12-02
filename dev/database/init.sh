#!/bin/bash

export PGUSER=$POSTGRES_USER
export PGPASSWORD=$POSTGRES_PASSWORD
export PGDATABASE=$POSTGRES_DB

# allow to create users for patchman database admin user
psql -c "ALTER USER ${POSTGRES_USER} WITH CREATEROLE"
psql -c "ALTER USER ${POSTGRES_USER} WITH SUPERUSER"
