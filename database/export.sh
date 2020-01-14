#!/bin/bash

set -u

export PGPASSWORD=${DB_PASSWD}
pg_dump -U ${DB_USER} -h ${DB_HOST} ${DB_NAME} -s -O -T schema_migrations