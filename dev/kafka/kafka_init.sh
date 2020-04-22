#!/bin/sh

set -x

# wait until kafka starts
>&2 echo "Checking if Kafka server is up"
until /usr/bin/kafka-topics --list --zookeeper zookeeper:2181 &> /dev/null; do
  >&2 echo "Kafka server is unavailable - sleeping"
  sleep 1
done

# create topics with multiple partitions for scaling
for topic in "platform.inventory.host-egress" "platform.inventory.events" "patchman.evaluator.upload" \
             "patchman.evaluator.recalc" "test"
do
    /usr/bin/kafka-topics --create --topic $topic --partitions 1 --zookeeper zookeeper:2181 --replication-factor 1
done
