#!/bin/bash

set -u

MIGRATIONS_DIR=${MIGRATIONS_DIR:-database/migrations}
DB_URL=${DB_URL:-"postgres:///${DB_NAME}?host=/var/run/postgresql/"}
CMD=${@:-up}

# Run the migration command, by default updating to newest DB version
migrate \
  -source file://${MIGRATIONS_DIR} \
  -database $DB_URL $CMD
