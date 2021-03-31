#!/bin/bash

# wait for all migrations applied
WAIT_FOR_FULL_DB=1 ./scripts/wait-for-services.sh

# fill database with testing data
PGPASSWORD=$DB_PASSWD psql -h $DB_HOST -d $DB_NAME -U $DB_USER -p $DB_PORT \
                       -v ON_ERROR_STOP=1 \
                       -a -f ./dev/test_data.sql
