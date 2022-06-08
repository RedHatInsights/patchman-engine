#!/bin/bash

# Connect to database for manual administration
PGSSLMODE=$DB_SSLMODE PGSSLROOTCERT=$DB_SSLROOTCERT PGPASSWORD=$DB_ADMIN_PASSWD psql -d $DB_NAME -h $DB_HOST -U $DB_ADMIN_USER -p $DB_PORT
