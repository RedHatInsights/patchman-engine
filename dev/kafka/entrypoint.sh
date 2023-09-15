#!/bin/sh

/app/setup.sh 2>&1 | grep -v '^WARNING: Due to limitations in metric names' &

exec /etc/confluent/docker/run 2>&1 \
    | grep -v -E ' (TRACE|DEBUG|INFO) '
