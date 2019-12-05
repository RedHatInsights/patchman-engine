#!/bin/sh

#set -e
set -x

DIR=$(dirname $0)

cd $DIR

# run zookeeper
./kafka/bin/zookeeper-server-start.sh -daemon kafka/config/zookeeper.properties

# run kafka
./kafka/bin/kafka-server-start.sh -daemon kafka/config/server.properties

# wait until kafka starts
>&2 echo "Checking if Kafka server is up"
until ./kafka/bin/kafka-topics.sh --list --zookeeper localhost:2181 &> /dev/null; do
  >&2 echo "Kafka server is unavailable - sleeping"
  sleep 1
done

# create topics with multiple partitions for scaling
for topic in "platform.upload.available" "platform.inventory.events"
do
    ./kafka/bin/kafka-topics.sh --create --topic $topic --partitions 1 --zookeeper localhost:2181 --replication-factor 1
done

# run upload mock
exec ./wait-for-services.sh ./platform
