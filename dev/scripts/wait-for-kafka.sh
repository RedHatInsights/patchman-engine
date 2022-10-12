#!/bin/bash

until curl -s $KAFKA_READY_ADDRESS >/dev/null ; do
  >&2 echo "Kafka topics not ready yet"
  sleep 1
done
