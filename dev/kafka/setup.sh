#!/bin/sh

#wait until kafka is ready
sleep 5

# create topics with multiple partitions for scaling
for topic in "platform.inventory.events" "patchman.evaluator.upload" \
             "patchman.evaluator.recalc" "platform.remediation-updates.patch" "platform.notifications.ingress" \
             "platform.payload-status" "test"
do
    until /usr/bin/kafka-topics --create --if-not-exists --topic $topic --partitions 1 --bootstrap-server kafka:9092 \
    --replication-factor 1; do
      echo "Unable to create topic $topic"
      sleep 1
    done
    echo "Topic $topic created successfully"
done
