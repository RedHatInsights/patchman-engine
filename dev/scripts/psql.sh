#!/bin/bash

if [[ -n $DB_SSLROOTCERT ]] ; then
    export PGSSLROOTCERT=$DB_SSLROOTCERT
    export PGSSLMODE=$DB_SSLMODE
fi
# Connect to database for manual administration
PGPASSWORD=$DB_ADMIN_PASSWD psql -d $DB_NAME -h $DB_HOST -U $DB_ADMIN_USER -p $DB_PORT
