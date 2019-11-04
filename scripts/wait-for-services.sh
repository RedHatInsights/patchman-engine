#!/usr/bin/bash

set -e

cmd="$@"

if [ ! -z "$POSTGRESQL_DATABASE" ]; then
  >&2 echo "Checking if PostgreSQL is up"
  until PGPASSWORD="$POSTGRESQL_PASSWORD" psql -h "$POSTGRESQL_HOST" -p "$POSTGRESQL_PORT" -U "$POSTGRESQL_USER" -d "$POSTGRESQL_DATABASE" -c '\q' -q 2>/dev/null; do
    >&2 echo "PostgreSQL is unavailable - sleeping"
    sleep 1
  done
else
  >&2 echo "Skipping PostgreSQL check"
fi

if [ ! -z "$KAFKA_HOST" ] && echo "from kafka import KafkaConsumer" | python3 &> /dev/null; then
  >&2 echo "Checking if Kafka server is up"
  until echo "from kafka import KafkaConsumer;c=KafkaConsumer(bootstrap_servers=[\"$KAFKA_HOST:$KAFKA_PORT\"]);c.close()" | python3 &> /dev/null; do
    >&2 echo "Kafka server is unavailable - sleeping"
    sleep 1
  done
else
  >&2 echo "Skipping Kafka server check"
fi

>&2 echo "Everything is up - executing command"
exec $cmd
