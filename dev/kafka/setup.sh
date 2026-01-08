#!/bin/sh

#wait until kafka is ready
sleep 5

# create topics with multiple partitions for scaling
for topic in \
            "patchman.evaluator.recalc" \
            "patchman.evaluator.upload" \
            "patchman.evaluator.user-evaluation" \
            "platform.content-sources.template" \
            "platform.inventory.events" \
            "platform.inventory.host-apps" \
            "platform.notifications.ingress" \
            "platform.payload-status" \
            "platform.remediation-updates.patch" \
            "test"
do
    until /opt/kafka/bin/kafka-topics.sh --create --if-not-exists --topic $topic \
        --partitions 1 --bootstrap-server kafka:9092 --replication-factor 1; do
      echo "Unable to create topic $topic"
      sleep 1
    done
    echo "Topic $topic created successfully"
done
# start simple http server so other components can check that kafka has fully started
while : ; do
    nc -lk -p 9099 -e echo -e "HTTP/1.1 200 OK\n\nTOPICS READY"
done
