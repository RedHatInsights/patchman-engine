#!/bin/bash

set -u

# accept either DB_NAME or or postgresql database, allowing running from database container and locally
DB_NAME=${DB_NAME:-${POSTGRESQL_DATABASE}}
# By default load migrations from local directory
MIGRATIONS_DIR=${MIGRATIONS_DIR:-database/migrations}
# By default use local postgresql instance and connect over socket
DB_URL=${DB_URL:-"postgres:///${DB_NAME}?host=/var/run/postgresql/"}
# by default update to latest version
CMD=${@:-up}

migrate \
  -source file://${MIGRATIONS_DIR} \
  -database $DB_URL $CMD
