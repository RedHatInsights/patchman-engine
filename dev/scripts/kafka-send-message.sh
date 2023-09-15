#!/bin/sh

# send message from stdin to kafka topic

TOPIC=${1:-platform.inventory.events}
podman exec -i kafka sh -c " tr '\n' ' ' | kafka-console-producer --topic $TOPIC --broker-list localhost:9092 "
