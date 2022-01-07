#!/usr/bin/bash

set -e

cmd="$@"

export PGSSLMODE=$DB_SSLMODE
export PGSSLROOTCERT=$DB_SSLROOTCERT

if [ ! -z "$DB_HOST" ]; then
  >&2 echo "Checking if PostgreSQL is up"
  if [ ! -z "$WAIT_FOR_EMPTY_DB" ]; then
    CHECK_QUERY="\q" # Wait only for empty database.
  elif [ ! -z "$WAIT_FOR_FULL_DB" ]; then
    # Wait for full schema, all migrations, e.g. before tests (schema_migrations.dirty='f').
    CHECK_QUERY="SELECT 1/count(*) FROM schema_migrations WHERE dirty='f';"
  else
    # Wait for created schema.
    CHECK_QUERY="SELECT * FROM schema_migrations;"
  fi
  until PGPASSWORD="$DB_PASSWD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "${CHECK_QUERY}" -q 2>/dev/null; do
    >&2 echo "PostgreSQL is unavailable - sleeping (host: $DB_HOST, port: $DB_PORT, user: $DB_USER, db_name: $DB_NAME)"
    sleep 1
  done
else
  >&2 echo "Skipping PostgreSQL check"
fi

>&2 echo "Everything is up - executing command"
exec $cmd
