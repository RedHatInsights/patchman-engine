#!/bin/sh

/setup.sh &

exec /etc/confluent/docker/run
