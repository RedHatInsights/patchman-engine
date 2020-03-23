#!/bin/bash

# Connect to database for manual administration
PGPASSWORD=$DB_PASSWD psql -d $DB_NAME -h $DB_HOST -U $DB_USER -p $DB_PORT
