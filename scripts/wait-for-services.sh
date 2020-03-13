#!/usr/bin/bash

set -e

cmd="$@"

if [ ! -z "$DB_HOST" ]; then
  >&2 echo "Checking if PostgreSQL is up"
  if [ ! -z "$WAIT_FOR_EMPTY_DB" ]; then
    CHECK_QUERY="\q" # Wait only for empty database.
  else
    CHECK_QUERY="SELECT * FROM schema_migrations;" # Wait even for database schema initialization.
  fi
  until PGPASSWORD="$DB_PASSWD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "${CHECK_QUERY}" -q 2>/dev/null; do
    >&2 echo "PostgreSQL is unavailable - sleeping"
    sleep 1
  done
else
  >&2 echo "Skipping PostgreSQL check"
fi

if [ ! -z "$KAFKA_ADDRESS" ] && echo "from kafka import KafkaConsumer" | python3 &> /dev/null; then
  >&2 echo "Checking if Kafka server is up"
  until echo "from kafka import KafkaConsumer;c=KafkaConsumer(bootstrap_servers=[\"$KAFKA_ADDRESS\"]);c.close()" | python3 &> /dev/null; do
    >&2 echo "Kafka server is unavailable - sleeping"
    sleep 1
  done
else
  >&2 echo "Skipping Kafka server check"
fi

>&2 echo "Everything is up - executing command"
exec $cmd
