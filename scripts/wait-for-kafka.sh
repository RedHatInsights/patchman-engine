#!/bin/sh

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

TOPICS="host.packages"

for topic in ${TOPICS}
do
    ./kafka/bin/kafka-topics.sh --create --topic $topic --partitions 10 --zookeeper localhost:2181 --replication-factor 1
done


echo "Kafka is running, sleeping"
while true; do
    sleep 1
done