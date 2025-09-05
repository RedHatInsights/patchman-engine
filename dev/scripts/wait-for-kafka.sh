#!/bin/bash

until [[ "$(curl -s $KAFKA_READY_ADDRESS)" == "TOPICS READY" ]] ; do
  >&2 echo "Kafka topics not ready yet"
  sleep 1
done
