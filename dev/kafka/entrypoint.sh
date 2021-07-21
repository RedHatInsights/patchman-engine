#!/bin/sh

/app/setup.sh &

exec /etc/confluent/docker/run
