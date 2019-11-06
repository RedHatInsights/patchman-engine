#!/bin/sh

DIR=$(dirname $0)

cd $DIR

TOPIC="host.packages"

for i in $(seq 1 ${GENERATE_MESSAGES}); do
  echo "Sending message $i"
  input="./data/body/${i}.json"
  ./kafka/bin/kafka-console-producer.sh --broker-list localhost:9092 --topic $TOPIC < $input &
done
