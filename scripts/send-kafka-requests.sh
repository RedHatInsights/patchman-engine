#!/bin/sh

DIR=$(dirname $0)

cd $DIR

TOPIC="host.packages"

for _ in $(seq 1 20); do
  input="./data/body/$(ls ./data/body | shuf -n 1)"
  ./kafka/bin/kafka-console-producer.sh --broker-list localhost:9092 --topic $TOPIC < $input &
done
