#!/bin/sh

# https://github.com/wurstmeister/kafka-docker/issues/389#issuecomment-800814529
sleep 20
/app/setup.sh 2>&1 | grep -v '^WARNING: Due to limitations in metric names' &

exec /etc/confluent/docker/run 2>&1 \
    | grep -v -E ' (TRACE|DEBUG|INFO) '
