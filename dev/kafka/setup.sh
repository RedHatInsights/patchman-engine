#!/bin/sh

# create topics with multiple partitions for scaling
for topic in "platform.inventory.host-egress" "platform.inventory.events" "patchman.evaluator.upload" \
             "patchman.evaluator.recalc" "platform.remediation-updates.patch" "test"
do
    until /usr/bin/kafka-topics --create --if-not-exists --topic $topic --partitions 1 --zookeeper zookeeper:2181 \
    --replication-factor 1; do
      echo "Unable to create topic $topic"
      sleep 1
    done
    echo "Topic $topic created successfully"
done
